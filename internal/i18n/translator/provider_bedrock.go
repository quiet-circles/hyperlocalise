package translator

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"
)

type BedrockProvider struct{}

func NewBedrockProvider() *BedrockProvider { return &BedrockProvider{} }

func (p *BedrockProvider) Name() string { return ProviderBedrock }

func (p *BedrockProvider) Translate(ctx context.Context, req Request) (string, error) {
	region := strings.TrimSpace(os.Getenv("AWS_REGION"))
	if region == "" {
		region = strings.TrimSpace(os.Getenv("AWS_DEFAULT_REGION"))
	}
	if region == "" {
		return "", fmt.Errorf("bedrock provider: AWS region is required (AWS_REGION or AWS_DEFAULT_REGION)")
	}

	accessKeyID := strings.TrimSpace(os.Getenv("AWS_ACCESS_KEY_ID"))
	secretAccessKey := strings.TrimSpace(os.Getenv("AWS_SECRET_ACCESS_KEY"))
	sessionToken := strings.TrimSpace(os.Getenv("AWS_SESSION_TOKEN"))
	if accessKeyID == "" || secretAccessKey == "" {
		return "", fmt.Errorf("bedrock provider: AWS credentials are required (AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY)")
	}

	endpoint := fmt.Sprintf("https://bedrock-runtime.%s.amazonaws.com", region)
	modelID := strings.TrimSpace(req.Model)
	payload := bedrockConverseRequest{
		System: []bedrockContent{{Text: buildSystemPrompt(req)}},
		Messages: []bedrockMessage{
			{
				Role: "user",
				Content: []bedrockContent{
					{Text: buildUserPrompt(req)},
				},
			},
		},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("bedrock provider: marshal request: %w", err)
	}

	path := "/model/" + url.PathEscape(modelID) + "/converse"
	requestURL := endpoint + path
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, strings.NewReader(string(payloadBytes)))
	if err != nil {
		return "", fmt.Errorf("bedrock provider: create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	if err := signBedrockRequest(httpReq, payloadBytes, region, accessKeyID, secretAccessKey, sessionToken, time.Now().UTC()); err != nil {
		return "", fmt.Errorf("bedrock provider: sign request: %w", err)
	}

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("bedrock provider request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("bedrock provider: read response: %w", err)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("bedrock provider: unexpected status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	text, usage, err := responseTextFromBedrock(body)
	if err != nil {
		return "", fmt.Errorf("bedrock provider response: %w", err)
	}
	SetUsage(ctx, usage)

	return text, nil
}

type bedrockConverseRequest struct {
	Messages []bedrockMessage `json:"messages"`
	System   []bedrockContent `json:"system,omitempty"`
}

type bedrockMessage struct {
	Role    string           `json:"role"`
	Content []bedrockContent `json:"content"`
}

type bedrockContent struct {
	Text string `json:"text"`
}

type bedrockConverseResponse struct {
	Output struct {
		Message struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		} `json:"message"`
	} `json:"output"`
	Usage bedrockTokenUsage `json:"usage"`
}

type bedrockTokenUsage struct {
	InputTokens  int `json:"inputTokens"`
	OutputTokens int `json:"outputTokens"`
	TotalTokens  int `json:"totalTokens"`
}

func responseTextFromBedrock(body []byte) (string, Usage, error) {
	var resp bedrockConverseResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", Usage{}, fmt.Errorf("decode response: %w", err)
	}

	builder := strings.Builder{}
	for _, block := range resp.Output.Message.Content {
		builder.WriteString(block.Text)
	}

	text := strings.TrimSpace(builder.String())
	if text == "" {
		return "", Usage{}, fmt.Errorf("no text generated")
	}
	usage := Usage{
		PromptTokens:     resp.Usage.InputTokens,
		CompletionTokens: resp.Usage.OutputTokens,
		TotalTokens:      resp.Usage.TotalTokens,
	}
	if usage.TotalTokens == 0 {
		usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	}

	return text, usage, nil
}

func signBedrockRequest(
	req *http.Request,
	payload []byte,
	region string,
	accessKeyID string,
	secretAccessKey string,
	sessionToken string,
	now time.Time,
) error {
	if req.URL == nil {
		return fmt.Errorf("request URL is nil")
	}

	const (
		service       = "bedrock-runtime"
		algorithm     = "AWS4-HMAC-SHA256"
		amzDateLayout = "20060102T150405Z"
		dateLayout    = "20060102"
	)

	amzDate := now.Format(amzDateLayout)
	dateStamp := now.Format(dateLayout)
	payloadHash := sha256Hex(payload)

	req.Header.Set("Host", req.URL.Host)
	req.Header.Set("X-Amz-Date", amzDate)
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)
	if sessionToken != "" {
		req.Header.Set("X-Amz-Security-Token", sessionToken)
	}

	signedHeaderNames := []string{"content-type", "host", "x-amz-content-sha256", "x-amz-date"}
	if sessionToken != "" {
		signedHeaderNames = append(signedHeaderNames, "x-amz-security-token")
	}
	sort.Strings(signedHeaderNames)

	canonicalHeaders := strings.Builder{}
	for _, name := range signedHeaderNames {
		canonicalHeaders.WriteString(name)
		canonicalHeaders.WriteString(":")
		canonicalHeaders.WriteString(strings.TrimSpace(req.Header.Get(name)))
		canonicalHeaders.WriteString("\n")
	}

	signedHeaders := strings.Join(signedHeaderNames, ";")
	canonicalRequest := strings.Join([]string{
		req.Method,
		req.URL.EscapedPath(),
		"",
		canonicalHeaders.String(),
		signedHeaders,
		payloadHash,
	}, "\n")

	credentialScope := strings.Join([]string{dateStamp, region, service, "aws4_request"}, "/")
	stringToSign := strings.Join([]string{
		algorithm,
		amzDate,
		credentialScope,
		sha256Hex([]byte(canonicalRequest)),
	}, "\n")

	signingKey := buildAWSV4SigningKey(secretAccessKey, dateStamp, region, service)
	signature := hex.EncodeToString(hmacSHA256(signingKey, stringToSign))

	auth := strings.Join([]string{
		algorithm + " Credential=" + accessKeyID + "/" + credentialScope,
		"SignedHeaders=" + signedHeaders,
		"Signature=" + signature,
	}, ", ")
	req.Header.Set("Authorization", auth)

	return nil
}

func buildAWSV4SigningKey(secretAccessKey, dateStamp, region, service string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secretAccessKey), dateStamp)
	kRegion := hmacSHA256(kDate, region)
	kService := hmacSHA256(kRegion, service)
	return hmacSHA256(kService, "aws4_request")
}

func hmacSHA256(key []byte, data string) []byte {
	hash := hmac.New(sha256.New, key)
	_, _ = hash.Write([]byte(data))
	return hash.Sum(nil)
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

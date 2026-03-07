package translator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	envPromptDebugEnabled = "HYPERLOCALISE_PROMPT_DEBUG"
	envPromptDebugPath    = "HYPERLOCALISE_PROMPT_DEBUG_FILE"
	envGenericDebug       = "DEBUG"
	defaultPromptLogPath  = ".hyperlocalise/logs/prompt.log"
)

type promptDebugLogger struct {
	mu sync.Mutex
}

type promptDebugEvent struct {
	Timestamp      string `json:"timestamp"`
	Event          string `json:"event"`
	Provider       string `json:"provider"`
	Model          string `json:"model"`
	Source         string `json:"source,omitempty"`
	TargetLanguage string `json:"target_language"`
	SystemPrompt   string `json:"system_prompt,omitempty"`
	UserPrompt     string `json:"user_prompt,omitempty"`
	Output         string `json:"output,omitempty"`
	Error          string `json:"error,omitempty"`
	DurationMS     int64  `json:"duration_ms,omitempty"`
}

var translatorPromptDebugLogger promptDebugLogger

func logPromptCall(req Request, providerName, systemPrompt, userPrompt string) {
	translatorPromptDebugLogger.write(promptDebugEvent{
		Timestamp:      time.Now().UTC().Format(time.RFC3339Nano),
		Event:          "prompt_call",
		Provider:       providerName,
		Model:          strings.TrimSpace(req.Model),
		Source:         strings.TrimSpace(req.Source),
		TargetLanguage: strings.TrimSpace(req.TargetLanguage),
		SystemPrompt:   systemPrompt,
		UserPrompt:     userPrompt,
	})
}

func logPromptResult(req Request, providerName, output string, err error, duration time.Duration) {
	event := promptDebugEvent{
		Timestamp:      time.Now().UTC().Format(time.RFC3339Nano),
		Event:          "prompt_result",
		Provider:       providerName,
		Model:          strings.TrimSpace(req.Model),
		Source:         strings.TrimSpace(req.Source),
		TargetLanguage: strings.TrimSpace(req.TargetLanguage),
		Output:         output,
		DurationMS:     duration.Milliseconds(),
	}
	if err != nil {
		event.Error = err.Error()
	}
	translatorPromptDebugLogger.write(event)
}

func (l *promptDebugLogger) write(event promptDebugEvent) {
	enabled, path := resolvePromptDebugConfig()
	if !enabled {
		return
	}

	data, err := json.Marshal(event)
	if err != nil {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer func() {
		_ = f.Close()
	}()

	_, _ = f.Write(append(data, '\n'))
}

func resolvePromptDebugConfig() (bool, string) {
	enabled := parsePromptDebugBool(os.Getenv(envPromptDebugEnabled))
	if !enabled {
		enabled = parsePromptDebugBool(os.Getenv(envGenericDebug))
	}
	path := strings.TrimSpace(os.Getenv(envPromptDebugPath))
	if path == "" {
		path = defaultPromptLogPath
	}
	return enabled, path
}

func parsePromptDebugBool(raw string) bool {
	if raw == "" {
		return false
	}
	parsed, err := strconv.ParseBool(strings.TrimSpace(raw))
	if err == nil {
		return parsed
	}

	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "on", "yes", "y":
		return true
	case "off", "no", "n":
		return false
	default:
		return false
	}
}

package translator

import (
	"context"
	"errors"
	"testing"

	"go.jetify.com/ai/api"
)

type fakeProvider struct {
	name   string
	result string
	err    error
}

func (p fakeProvider) Name() string { return p.name }

func (p fakeProvider) Translate(_ context.Context, _ Request) (string, error) {
	if p.err != nil {
		return "", p.err
	}
	return p.result, nil
}

func TestRegisterRejectsDuplicateProvider(t *testing.T) {
	t.Parallel()

	tool := &Tool{providers: map[string]Provider{}}
	provider := fakeProvider{name: "openai"}

	if err := tool.Register(provider); err != nil {
		t.Fatalf("register provider: %v", err)
	}

	if err := tool.Register(provider); err == nil {
		t.Fatalf("expected duplicate registration error")
	}
}

func TestTranslateRejectsUnknownProvider(t *testing.T) {
	t.Parallel()

	tool := &Tool{providers: map[string]Provider{}}
	_, err := tool.Translate(context.Background(), Request{
		Source:         "hello",
		TargetLanguage: "fr",
		ModelProvider:  "unknown",
		Model:          "gpt-5",
	})
	if err == nil {
		t.Fatalf("expected unknown provider error")
	}
}

func TestTranslateUsesRegisteredProvider(t *testing.T) {
	t.Parallel()

	tool := &Tool{providers: map[string]Provider{}}
	if err := tool.Register(fakeProvider{name: ProviderOpenAI, result: "bonjour"}); err != nil {
		t.Fatalf("register provider: %v", err)
	}

	translated, err := tool.Translate(context.Background(), Request{
		Source:         "hello",
		TargetLanguage: "fr",
		Model:          "gpt-5",
	})
	if err != nil {
		t.Fatalf("translate: %v", err)
	}
	if translated != "bonjour" {
		t.Fatalf("unexpected translation: %q", translated)
	}
}

func TestNewRegistersDefaultProviders(t *testing.T) {
	t.Parallel()

	tool := New()

	if _, ok := tool.providers[ProviderOpenAI]; !ok {
		t.Fatalf("expected %q provider to be registered", ProviderOpenAI)
	}

	if _, ok := tool.providers[ProviderLMStudio]; !ok {
		t.Fatalf("expected %q provider to be registered", ProviderLMStudio)
	}

	if _, ok := tool.providers[ProviderGroq]; !ok {
		t.Fatalf("expected %q provider to be registered", ProviderGroq)
	}

	if _, ok := tool.providers[ProviderOllama]; !ok {
		t.Fatalf("expected %q provider to be registered", ProviderOllama)
	}
}

func TestResponseText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		resp    *api.Response
		want    string
		wantErr bool
	}{
		{
			name: "single text block",
			resp: &api.Response{Content: []api.ContentBlock{
				&api.TextBlock{Text: "bonjour"},
			}},
			want: "bonjour",
		},
		{
			name:    "empty content",
			resp:    &api.Response{},
			want:    "",
			wantErr: true,
		},
		{
			name:    "nil response",
			resp:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := responseText(tc.resp)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}

			if err != nil {
				t.Fatalf("responseText error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("responseText = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestProviderErrorIsWrapped(t *testing.T) {
	t.Parallel()

	tool := &Tool{providers: map[string]Provider{}}
	baseErr := errors.New("boom")
	if err := tool.Register(fakeProvider{name: ProviderOpenAI, err: baseErr}); err != nil {
		t.Fatalf("register provider: %v", err)
	}

	_, err := tool.Translate(context.Background(), Request{
		Source:         "hello",
		TargetLanguage: "fr",
		Model:          "gpt-5",
	})
	if !errors.Is(err, baseErr) {
		t.Fatalf("expected wrapped provider error")
	}
}

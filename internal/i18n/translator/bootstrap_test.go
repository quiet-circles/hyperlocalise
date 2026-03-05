package translator

import (
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestRegisterBuiltinsRegistersExpectedProviders(t *testing.T) {
	t.Parallel()

	tool := &Tool{providers: map[string]Provider{}}

	if err := RegisterBuiltins(tool); err != nil {
		t.Fatalf("register built-ins: %v", err)
	}

	got := make([]string, 0, len(tool.providers))
	for name := range tool.providers {
		got = append(got, name)
	}
	slices.Sort(got)
	want := []string{
		ProviderAnthropic,
		ProviderAzureOpenAI,
		ProviderBedrock,
		ProviderGemini,
		ProviderGroq,
		ProviderLMStudio,
		ProviderMistral,
		ProviderOllama,
		ProviderOpenAI,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected registered providers: got %v want %v", got, want)
	}
}

func TestRegisterBuiltinsRejectsDuplicates(t *testing.T) {
	t.Parallel()

	tool := &Tool{providers: map[string]Provider{}}

	if err := RegisterBuiltins(tool); err != nil {
		t.Fatalf("register built-ins first time: %v", err)
	}
	if err := RegisterBuiltins(tool); err == nil || !strings.Contains(err.Error(), "already registered") {
		t.Fatalf("expected duplicate registration error, got %v", err)
	}
}

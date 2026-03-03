package cmd

import (
	"testing"

	"github.com/quiet-circles/hyperlocalise/internal/i18n/runsvc"
)

func TestContextMemoryPhaseMessageIncludesProgressAndFallback(t *testing.T) {
	got := contextMemoryPhaseMessage(runsvc.Event{
		ContextMemoryTotal:     5,
		ContextMemoryProcessed: 2,
		ContextMemoryFallbacks: 1,
		Message:                "context memory fallback for scope",
	})
	want := "Building context memory... (2/5, fallback=1) context memory fallback for scope"
	if got != want {
		t.Fatalf("context memory phase message = %q, want %q", got, want)
	}
}

func TestContextMemoryPhaseMessageWithoutTotals(t *testing.T) {
	got := contextMemoryPhaseMessage(runsvc.Event{Message: "starting context memory generation"})
	want := "Building context memory... starting context memory generation"
	if got != want {
		t.Fatalf("context memory phase message = %q, want %q", got, want)
	}
}

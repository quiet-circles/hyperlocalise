package runsvc

import (
	"testing"

	"github.com/quiet-circles/hyperlocalise/internal/config"
)

func TestParserModeForSourceDetectsStrictFormatJSByContent(t *testing.T) {
	mode := parserModeForSource("tests/misc/en-US.json", []byte(`{"hello":{"defaultMessage":"Hello"}}`))
	if mode != "formatjs" {
		t.Fatalf("mode=%q, want formatjs", mode)
	}
}

func TestParserModeForSourceDetectsPlainJSONByContent(t *testing.T) {
	mode := parserModeForSource("tests/misc/en-US.json", []byte(`{"hello":"Hello"}`))
	if mode != "json" {
		t.Fatalf("mode=%q, want json", mode)
	}
}

func TestParserModeForSourceDetectsARBByExtension(t *testing.T) {
	mode := parserModeForSource("tests/misc/app_en.arb", []byte(`{"hello":"Hello","@hello":{"description":"Greeting"}}`))
	if mode != "arb" {
		t.Fatalf("mode=%q, want arb", mode)
	}
}

func TestResolveRetrievalSnapshotUsesExplicitVersion(t *testing.T) {
	cfg := &config.I18NConfig{}
	cfg.Cache.RetrievalCorpusSnapshotVersion = "snapshot-v42"
	got := resolveRetrievalSnapshot(cfg)
	if got != "snapshot-v42" {
		t.Fatalf("snapshot=%q, want snapshot-v42", got)
	}
}

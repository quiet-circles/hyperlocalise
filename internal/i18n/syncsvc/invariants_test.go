package syncsvc

import (
	"strings"
	"testing"

	"github.com/quiet-circles/hyperlocalise/internal/i18n/icuparser"
	"github.com/quiet-circles/hyperlocalise/internal/i18n/storage"
)

func TestValidateEntryInvariantICUParityUsesParsedStructure(t *testing.T) {
	base := "{count, plural, one {{name} invited} other {{name} and # others invited}}"
	candidate := "{count, plural, one {{name} invited}}"

	diags := validateEntryInvariant(storage.Entry{Value: candidate}, storage.Entry{Value: base})
	if len(diags) == 0 {
		t.Fatalf("expected ICU parity mismatch")
	}
	if !strings.Contains(strings.Join(diags, " | "), "ICU parity mismatch") {
		t.Fatalf("unexpected diags: %#v", diags)
	}
}

func TestValidateEntryInvariantPlaceholderParityUsesICUParserPackage(t *testing.T) {
	// Sanity check the validator is comparing parser outputs from the extracted package.
	base := storage.Entry{Value: "Hi {name} on {ts, date, ::yyyyMMdd}"}
	candidate := storage.Entry{Value: "Hi {name}"}

	diags := validateEntryInvariant(candidate, base)
	if len(diags) == 0 {
		t.Fatalf("expected placeholder parity mismatch")
	}
	if _, err := icuparser.ParseInvariant(base.Value); err != nil {
		t.Fatalf("parser package parse failed: %v", err)
	}
	if !strings.Contains(strings.Join(diags, " | "), "placeholder parity mismatch") {
		t.Fatalf("unexpected diags: %#v", diags)
	}
}

func TestValidateEntryInvariantAcceptsApostrophesWithoutLosingPlaceholderParity(t *testing.T) {
	base := storage.Entry{Value: "It's {name}"}
	candidate := storage.Entry{Value: "It''s {name}"}

	diags := validateEntryInvariant(candidate, base)
	if len(diags) != 0 {
		t.Fatalf("expected no invariant diagnostics, got %#v", diags)
	}
}

func TestValidateEntryInvariantFlagsInvalidBraceInTagBody(t *testing.T) {
	base := storage.Entry{Value: "<b>{name}</b>"}
	candidate := storage.Entry{Value: "<b>{name}}</b>"}

	diags := validateEntryInvariant(candidate, base)
	if len(diags) == 0 {
		t.Fatalf("expected invalid ICU/braces structure diagnostic")
	}
	if !strings.Contains(strings.Join(diags, " | "), "invalid ICU/braces structure in candidate") {
		t.Fatalf("unexpected diags: %#v", diags)
	}
}

func TestValidateEntryInvariantFlagsTranslatedICUKeyword(t *testing.T) {
	base := storage.Entry{Value: "{plan, select, free {Free plan} other {Custom plan}}"}
	candidate := storage.Entry{Value: "{plan, 选择, free {免费计划} other {自定义计划}}"}

	diags := validateEntryInvariant(candidate, base)
	if len(diags) == 0 {
		t.Fatalf("expected ICU parity mismatch")
	}
	if !strings.Contains(strings.Join(diags, " | "), "ICU parity mismatch") {
		t.Fatalf("unexpected diags: %#v", diags)
	}
}

func TestValidateEntryInvariantFlagsTranslatedPlaceholderName(t *testing.T) {
	base := storage.Entry{Value: "{name} mentioned you in {projectName}"}
	candidate := storage.Entry{Value: "{ten} da nhac den ban trong {projectName}"}

	diags := validateEntryInvariant(candidate, base)
	if len(diags) == 0 {
		t.Fatalf("expected placeholder parity mismatch")
	}
	if !strings.Contains(strings.Join(diags, " | "), "placeholder parity mismatch") {
		t.Fatalf("unexpected diags: %#v", diags)
	}
}

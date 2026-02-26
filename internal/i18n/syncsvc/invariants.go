package syncsvc

import (
	"fmt"
	"slices"
	"strings"

	"github.com/quiet-circles/hyperlocalise/internal/i18n/icuparser"
	"github.com/quiet-circles/hyperlocalise/internal/i18n/storage"
)

const (
	conflictReasonInvariantViolation = "invariant_violation"
)

func validateEntryInvariant(candidate, baseline storage.Entry) []string {
	baseInv, baseErr := icuparser.ParseInvariant(baseline.Value)
	candInv, candErr := icuparser.ParseInvariant(candidate.Value)

	var diags []string
	if baseErr != nil || candErr != nil {
		if baseErr == nil && candErr != nil {
			diags = append(diags, fmt.Sprintf("invalid ICU/braces structure in candidate: %v", candErr))
		}
		return diags
	}

	if !slices.Equal(baseInv.Placeholders, candInv.Placeholders) {
		diags = append(diags, fmt.Sprintf(
			"placeholder parity mismatch (expected %v, got %v)",
			baseInv.Placeholders,
			candInv.Placeholders,
		))
	}
	if !equalICUParity(baseInv.ICUBlocks, candInv.ICUBlocks) {
		diags = append(diags, fmt.Sprintf(
			"ICU parity mismatch (expected %s, got %s)",
			formatICUBlocks(baseInv.ICUBlocks),
			formatICUBlocks(candInv.ICUBlocks),
		))
	}

	return diags
}

func equalICUParity(a, b []icuparser.BlockSignature) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Arg != b[i].Arg || a[i].Type != b[i].Type || !slices.Equal(a[i].Options, b[i].Options) {
			return false
		}
	}
	return true
}

func formatICUBlocks(blocks []icuparser.BlockSignature) string {
	if len(blocks) == 0 {
		return "[]"
	}
	parts := make([]string, 0, len(blocks))
	for _, b := range blocks {
		parts = append(parts, fmt.Sprintf("%s:%s%v", b.Arg, b.Type, b.Options))
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func formatInvariantWarning(prefix string, id storage.EntryID, diags []string) string {
	label := id.Locale + "/" + id.Key
	if strings.TrimSpace(id.Context) != "" {
		label += " [" + id.Context + "]"
	}
	return fmt.Sprintf("%s: %s: %s", prefix, label, strings.Join(diags, "; "))
}

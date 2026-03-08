package syncsvc

import (
	"fmt"
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

	if !icuparser.SamePlaceholderSet(baseInv.Placeholders, candInv.Placeholders) {
		diags = append(diags, fmt.Sprintf(
			"placeholder parity mismatch (expected %v, got %v)",
			baseInv.Placeholders,
			candInv.Placeholders,
		))
	}
	if !icuparser.SameICUBlocks(baseInv.ICUBlocks, candInv.ICUBlocks) {
		diags = append(diags, fmt.Sprintf(
			"ICU parity mismatch (expected %s, got %s)",
			icuparser.FormatICUBlocks(baseInv.ICUBlocks),
			icuparser.FormatICUBlocks(candInv.ICUBlocks),
		))
	}
	if icuparser.HasDuplicatePounds(candInv.ICUBlocks) {
		diags = append(diags, fmt.Sprintf(
			"duplicate # tokens in ICU plural/selectordinal branch (got %s)",
			icuparser.FormatICUBlocks(candInv.ICUBlocks),
		))
	}

	return diags
}

func formatInvariantWarning(prefix string, id storage.EntryID, diags []string) string {
	label := id.Locale + "/" + id.Key
	if strings.TrimSpace(id.Context) != "" {
		label += " [" + id.Context + "]"
	}
	return fmt.Sprintf("%s: %s: %s", prefix, label, strings.Join(diags, "; "))
}

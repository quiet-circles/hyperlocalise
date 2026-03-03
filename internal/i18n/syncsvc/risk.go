package syncsvc

import (
	"math"
	"strings"
	"unicode/utf8"

	"github.com/quiet-circles/hyperlocalise/internal/i18n/storage"
)

const (
	RiskCodeLengthSpike     = "length_spike"
	RiskCodePlaceholderEdit = "placeholder_edit"

	defaultLengthSpikeRatio = 1.8
	minBaselineLength       = 8
)

type RiskChange struct {
	ID              storage.EntryID `json:"id"`
	Code            string          `json:"code"`
	Message         string          `json:"message"`
	BaselineLength  int             `json:"baseline_length,omitempty"`
	CandidateLength int             `json:"candidate_length,omitempty"`
	Ratio           float64         `json:"ratio,omitempty"`
}

func detectRiskyChanges(id storage.EntryID, baseline, candidate string, invariantDiags []string) []RiskChange {
	risky := make([]RiskChange, 0, 2)

	if hasPlaceholderEdit(invariantDiags) {
		risky = append(risky, RiskChange{
			ID:      id,
			Code:    RiskCodePlaceholderEdit,
			Message: "placeholder or ICU structure edited",
		})
	}

	baselineLen := utf8.RuneCountInString(strings.TrimSpace(baseline))
	candidateLen := utf8.RuneCountInString(strings.TrimSpace(candidate))
	if spike, ratio := isLengthSpike(candidateLen, baselineLen, defaultLengthSpikeRatio); spike {
		risky = append(risky, RiskChange{
			ID:              id,
			Code:            RiskCodeLengthSpike,
			Message:         "candidate value length increased sharply",
			BaselineLength:  baselineLen,
			CandidateLength: candidateLen,
			Ratio:           ratio,
		})
	}

	return risky
}

func isLengthSpike(candidateLen, baselineLen int, ratioThreshold float64) (bool, float64) {
	if baselineLen < minBaselineLength {
		return false, 0
	}
	if candidateLen <= baselineLen {
		return false, 0
	}
	ratio := float64(candidateLen) / float64(baselineLen)
	if ratio < ratioThreshold {
		return false, ratio
	}
	return true, math.Round(ratio*100) / 100
}

func hasPlaceholderEdit(diags []string) bool {
	for _, d := range diags {
		if strings.Contains(d, "placeholder parity mismatch") ||
			strings.Contains(d, "ICU parity mismatch") ||
			strings.Contains(d, "invalid ICU/braces structure") {
			return true
		}
	}
	return false
}

func compareRiskChange(a, b RiskChange) int {
	if c := compareEntryID(a.ID, b.ID); c != 0 {
		return c
	}
	if c := strings.Compare(a.Code, b.Code); c != 0 {
		return c
	}
	return strings.Compare(a.Message, b.Message)
}

package scoring

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/quiet-circles/hyperlocalise/internal/i18n/icuparser"
)

const (
	HardFailEmptyOutput     = "empty_output"
	HardFailSourceCopied    = "source_copied_unchanged"
	HardFailMalformedICU    = "malformed_icu"
	HardFailPlaceholderDrop = "placeholder_integrity_failed"
)

type Weights struct {
	PlaceholderIntegrity float64
	ReferenceExact       float64
	ReferenceNormalized  float64
	ReferenceSimilarity  float64
}

type Result struct {
	PlaceholderIntegrity float64            `json:"placeholderIntegrity"`
	ReferenceExact       *float64           `json:"referenceExact,omitempty"`
	ReferenceNormalized  *float64           `json:"referenceNormalized,omitempty"`
	ReferenceSimilarity  *float64           `json:"referenceSimilarity,omitempty"`
	WeightedAggregate    float64            `json:"weightedAggregate"`
	HardFails            []string           `json:"hardFails,omitempty"`
	Details              map[string]float64 `json:"details,omitempty"`
}

type Evaluator struct {
	weights Weights
}

func NewEvaluator() *Evaluator {
	return &Evaluator{weights: Weights{
		PlaceholderIntegrity: 0.4,
		ReferenceExact:       0.2,
		ReferenceNormalized:  0.2,
		ReferenceSimilarity:  0.2,
	}}
}

func (e *Evaluator) Evaluate(source, translated, reference string) Result {
	result := Result{Details: map[string]float64{}}

	srcTrimmed := strings.TrimSpace(source)
	translatedTrimmed := strings.TrimSpace(translated)
	referenceTrimmed := strings.TrimSpace(reference)

	result.PlaceholderIntegrity = placeholderIntegrityScore(srcTrimmed, translatedTrimmed)
	result.Details["placeholderIntegrity"] = round3(result.PlaceholderIntegrity)

	hardFailSet := map[string]struct{}{}
	if translatedTrimmed == "" {
		hardFailSet[HardFailEmptyOutput] = struct{}{}
	}
	if normalizeText(source) == normalizeText(translated) {
		hardFailSet[HardFailSourceCopied] = struct{}{}
	}

	srcInv, srcErr := icuparser.ParseInvariant(srcTrimmed)
	translatedInv, translatedErr := icuparser.ParseInvariant(translatedTrimmed)
	if srcErr == nil && (len(srcInv.Placeholders) > 0 || len(srcInv.ICUBlocks) > 0) && translatedErr != nil {
		hardFailSet[HardFailMalformedICU] = struct{}{}
	}
	if srcErr == nil && translatedErr == nil && !sameBlocks(srcInv.ICUBlocks, translatedInv.ICUBlocks) {
		hardFailSet[HardFailPlaceholderDrop] = struct{}{}
	}
	if result.PlaceholderIntegrity < 1 {
		hardFailSet[HardFailPlaceholderDrop] = struct{}{}
	}

	numerator := result.PlaceholderIntegrity * e.weights.PlaceholderIntegrity
	denominator := e.weights.PlaceholderIntegrity

	if referenceTrimmed != "" {
		exact := 0.0
		if translatedTrimmed == referenceTrimmed {
			exact = 1
		}
		norm := 0.0
		if normalizeText(translatedTrimmed) == normalizeText(referenceTrimmed) {
			norm = 1
		}
		sim := tokenF1(referenceTrimmed, translatedTrimmed)
		result.ReferenceExact = &exact
		result.ReferenceNormalized = &norm
		result.ReferenceSimilarity = &sim
		result.Details["referenceExact"] = round3(exact)
		result.Details["referenceNormalized"] = round3(norm)
		result.Details["referenceSimilarity"] = round3(sim)
		numerator += exact*e.weights.ReferenceExact + norm*e.weights.ReferenceNormalized + sim*e.weights.ReferenceSimilarity
		denominator += e.weights.ReferenceExact + e.weights.ReferenceNormalized + e.weights.ReferenceSimilarity
	}

	if denominator > 0 {
		result.WeightedAggregate = numerator / denominator
	}

	if len(hardFailSet) > 0 {
		result.HardFails = make([]string, 0, len(hardFailSet))
		for fail := range hardFailSet {
			result.HardFails = append(result.HardFails, fail)
		}
		sort.Strings(result.HardFails)
		result.WeightedAggregate = 0
	}

	result.WeightedAggregate = round3(result.WeightedAggregate)
	return result
}

var (
	bracePlaceholderPattern  = regexp.MustCompile(`\{\s*([A-Za-z_$][A-Za-z0-9_.$-]*)\s*\}`)
	printfPlaceholderPattern = regexp.MustCompile(`%(?:\[[0-9]+\])?[-+#0 ]*(?:\d+|\*)?(?:\.(?:\d+|\*))?[hlLzjt]*[bcdeEfFgGosxXqvTt]`)
)

func placeholderIntegrityScore(source, translated string) float64 {
	sourceTokens := placeholderTokens(source)
	if len(sourceTokens) == 0 {
		return 1
	}
	translatedTokens := placeholderTokens(translated)

	sourceCount := map[string]int{}
	for _, token := range sourceTokens {
		sourceCount[token]++
	}
	translatedCount := map[string]int{}
	for _, token := range translatedTokens {
		translatedCount[token]++
	}

	matched := 0
	for token, count := range sourceCount {
		matched += min(count, translatedCount[token])
	}
	return float64(matched) / float64(len(sourceTokens))
}

func placeholderTokens(s string) []string {
	tokens := make([]string, 0)
	inv, err := icuparser.ParseInvariant(s)
	if err == nil {
		for _, ph := range inv.Placeholders {
			tokens = append(tokens, fmt.Sprintf("icu:%s", ph))
		}
		for _, block := range inv.ICUBlocks {
			tokens = append(tokens, fmt.Sprintf("icu-block:%s:%s:%s", block.Arg, block.Type, strings.Join(block.Options, ",")))
		}
	}
	for _, match := range bracePlaceholderPattern.FindAllStringSubmatch(s, -1) {
		tokens = append(tokens, fmt.Sprintf("brace:%s", match[1]))
	}
	for _, match := range printfPlaceholderPattern.FindAllString(s, -1) {
		tokens = append(tokens, fmt.Sprintf("printf:%s", match))
	}
	sort.Strings(tokens)
	return dedupAdjacent(tokens)
}

func sameBlocks(a, b []icuparser.BlockSignature) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Arg != b[i].Arg || a[i].Type != b[i].Type || strings.Join(a[i].Options, "|") != strings.Join(b[i].Options, "|") {
			return false
		}
	}
	return true
}

func tokenF1(reference, candidate string) float64 {
	r := tokenize(reference)
	c := tokenize(candidate)
	if len(r) == 0 && len(c) == 0 {
		return 1
	}
	if len(r) == 0 || len(c) == 0 {
		return 0
	}
	rCount := map[string]int{}
	for _, tok := range r {
		rCount[tok]++
	}
	cCount := map[string]int{}
	for _, tok := range c {
		cCount[tok]++
	}
	matches := 0
	for tok, cnt := range rCount {
		matches += min(cnt, cCount[tok])
	}
	precision := float64(matches) / float64(len(c))
	recall := float64(matches) / float64(len(r))
	if precision+recall == 0 {
		return 0
	}
	return 2 * precision * recall / (precision + recall)
}

func tokenize(s string) []string {
	s = normalizeText(s)
	if s == "" {
		return nil
	}
	return strings.Fields(s)
}

func normalizeText(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return ""
	}
	var b strings.Builder
	lastSpace := false
	for _, r := range s {
		if unicode.IsPunct(r) && r != '_' && r != '$' && r != '%' && r != '{' && r != '}' {
			continue
		}
		if unicode.IsSpace(r) {
			if !lastSpace {
				b.WriteByte(' ')
				lastSpace = true
			}
			continue
		}
		lastSpace = false
		b.WriteRune(r)
	}
	return strings.TrimSpace(b.String())
}

func dedupAdjacent(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	out := items[:1]
	for i := 1; i < len(items); i++ {
		if items[i] != items[i-1] {
			out = append(out, items[i])
		}
	}
	return out
}

func round3(v float64) float64 {
	return math.Round(v*1000) / 1000
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

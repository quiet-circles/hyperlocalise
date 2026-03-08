package icuparser

import (
	"fmt"
	"slices"
	"sort"
	"strings"
)

type Invariant struct {
	Placeholders []string
	ICUBlocks    []BlockSignature
}

type BlockSignature struct {
	Arg     string
	Type    string
	Options []string
	Pounds  []int
}

func ParseInvariant(s string) (Invariant, error) {
	elems, err := Parse(s, nil)
	if err != nil {
		normalized := normalizeMustachePlaceholders(s)
		elems, err = Parse(normalized, nil)
		if err != nil {
			return Invariant{}, err
		}
	}

	inv := Invariant{}
	collectInvariantFromElements(elems, &inv, "")

	sort.Strings(inv.Placeholders)
	sort.Slice(inv.ICUBlocks, func(i, j int) bool {
		if inv.ICUBlocks[i].Arg != inv.ICUBlocks[j].Arg {
			return inv.ICUBlocks[i].Arg < inv.ICUBlocks[j].Arg
		}
		if inv.ICUBlocks[i].Type != inv.ICUBlocks[j].Type {
			return inv.ICUBlocks[i].Type < inv.ICUBlocks[j].Type
		}
		left := strings.Join(inv.ICUBlocks[i].Options, "\x00") + "|" + formatPoundCounts(inv.ICUBlocks[i].Pounds)
		right := strings.Join(inv.ICUBlocks[j].Options, "\x00") + "|" + formatPoundCounts(inv.ICUBlocks[j].Pounds)
		return left < right
	})
	return inv, nil
}

func SamePlaceholderSet(a, b []string) bool {
	return slicesEqual(uniqueStrings(a), uniqueStrings(b))
}

func SameICUBlocks(a, b []BlockSignature) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Arg != b[i].Arg || a[i].Type != b[i].Type || !slicesEqual(a[i].Options, b[i].Options) {
			return false
		}
	}
	return true
}

func HasDuplicatePounds(blocks []BlockSignature) bool {
	for _, block := range blocks {
		for _, count := range block.Pounds {
			if count > 1 {
				return true
			}
		}
	}
	return false
}

func FormatICUBlocks(blocks []BlockSignature) string {
	if len(blocks) == 0 {
		return "[]"
	}
	parts := make([]string, 0, len(blocks))
	for _, b := range blocks {
		if hasNonZeroPounds(b.Pounds) {
			parts = append(parts, fmt.Sprintf("%s:%s%v#%v", b.Arg, b.Type, b.Options, b.Pounds))
			continue
		}
		parts = append(parts, fmt.Sprintf("%s:%s%v", b.Arg, b.Type, b.Options))
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func normalizeMustachePlaceholders(s string) string {
	var b strings.Builder

	for i := 0; i < len(s); {
		if i+3 < len(s) && s[i] == '{' && s[i+1] == '{' {
			j := i + 2
			for j < len(s) && s[j] != '}' {
				j++
			}
			if j+1 < len(s) && s[j] == '}' && s[j+1] == '}' {
				name := strings.TrimSpace(s[i+2 : j])
				if isPlaceholderName(name) {
					// Convert moustache placeholders to ICU-style arguments for fallback parsing.
					b.WriteByte('{')
					b.WriteString(name)
					b.WriteByte('}')
					i = j + 2
					continue
				}
			}
		}
		b.WriteByte(s[i])
		i++
	}

	return b.String()
}

func collectInvariantFromElements(elems []Element, inv *Invariant, pluralArg string) {
	for _, el := range elems {
		collectInvariantFromElement(el, inv, pluralArg)
	}
}

func collectInvariantFromElement(el Element, inv *Invariant, pluralArg string) {
	switch v := el.(type) {
	case ArgumentElement:
		appendPlaceholder(inv, v.Value)
	case NumberElement:
		appendPlaceholder(inv, v.Value)
	case DateElement:
		appendPlaceholder(inv, v.Value)
	case TimeElement:
		appendPlaceholder(inv, v.Value)
	case SelectElement:
		appendSelectBlockInvariant(inv, v, pluralArg)
	case PluralElement:
		appendPluralBlockInvariant(inv, v)
	case TagElement:
		collectInvariantFromElements(v.Children, inv, pluralArg)
	case PoundElement:
		if pluralArg != "" {
			appendPlaceholder(inv, pluralArg)
		}
		return
	case LiteralElement:
		return
	default:
		// Defensive no-op for future element types.
		return
	}
}

func appendPlaceholder(inv *Invariant, value string) {
	if isPlaceholderName(value) {
		inv.Placeholders = append(inv.Placeholders, value)
	}
}

func appendSelectBlockInvariant(inv *Invariant, v SelectElement, pluralArg string) {
	inv.ICUBlocks = append(inv.ICUBlocks, BlockSignature{
		Arg:     v.Value,
		Type:    "select",
		Options: sortedSelectors(v.Options),
	})
	for _, opt := range v.Options {
		collectInvariantFromElements(opt.Value, inv, pluralArg)
	}
}

func appendPluralBlockInvariant(inv *Invariant, v PluralElement) {
	blockType := "plural"
	if v.Type() == TypeSelectOrdinal {
		blockType = "selectordinal"
	}
	appendPlaceholder(inv, v.Value)
	sortedOptions, poundCounts := sortedPluralOptionSignatures(v.Options)

	inv.ICUBlocks = append(inv.ICUBlocks, BlockSignature{
		Arg:     v.Value,
		Type:    blockType,
		Options: sortedOptions,
		Pounds:  poundCounts,
	})
	for _, opt := range v.Options {
		collectInvariantFromElements(opt.Value, inv, v.Value)
	}
}

func sortedSelectors(opts []SelectOption) []string {
	out := make([]string, 0, len(opts))
	for _, o := range opts {
		out = append(out, o.Selector)
	}
	sort.Strings(out)
	return out
}

func sortedPluralSelectors(opts []PluralOption) []string {
	out := make([]string, 0, len(opts))
	for _, o := range opts {
		out = append(out, o.Selector)
	}
	sort.Strings(out)
	return out
}

func sortedPluralOptionSignatures(opts []PluralOption) ([]string, []int) {
	type optionSig struct {
		selector string
		pounds   int
	}
	sigs := make([]optionSig, 0, len(opts))
	for _, o := range opts {
		sigs = append(sigs, optionSig{selector: o.Selector, pounds: countPounds(o.Value)})
	}
	sort.Slice(sigs, func(i, j int) bool {
		return sigs[i].selector < sigs[j].selector
	})
	selectors := make([]string, 0, len(sigs))
	pounds := make([]int, 0, len(sigs))
	for _, sig := range sigs {
		selectors = append(selectors, sig.selector)
		pounds = append(pounds, sig.pounds)
	}
	if !hasNonZeroPounds(pounds) {
		pounds = nil
	}
	return selectors, pounds
}

func countPounds(elems []Element) int {
	total := 0
	for _, el := range elems {
		switch v := el.(type) {
		case PoundElement:
			total++
		case TagElement:
			total += countPounds(v.Children)
		case SelectElement:
			for _, opt := range v.Options {
				total += countPounds(opt.Value)
			}
		case PluralElement:
			for _, opt := range v.Options {
				total += countPounds(opt.Value)
			}
		}
	}
	return total
}

func hasNonZeroPounds(values []int) bool {
	for _, v := range values {
		if v != 0 {
			return true
		}
	}
	return false
}

func formatPoundCounts(values []int) string {
	if len(values) == 0 {
		return ""
	}
	parts := make([]string, len(values))
	for i, v := range values {
		parts[i] = fmt.Sprintf("%d", v)
	}
	return strings.Join(parts, ",")
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	sorted := append([]string(nil), values...)
	slices.Sort(sorted)
	out := make([]string, 0, len(sorted))
	var last string
	for i, value := range sorted {
		if i == 0 || value != last {
			out = append(out, value)
			last = value
		}
	}
	return out
}

func slicesEqual[T comparable](a, b []T) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func isPlaceholderName(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	for i, r := range s {
		if i == 0 {
			if !isPlaceholderFirstRune(r) {
				return false
			}
			continue
		}
		if !isPlaceholderSubsequentRune(r) {
			return false
		}
	}
	return true
}

func isASCIIDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

func isPlaceholderFirstRune(r rune) bool {
	return isASCIILetter(r) || r == '_' || r == '$'
}

func isPlaceholderSubsequentRune(r rune) bool {
	return isASCIILetter(r) || isASCIIDigitRune(r) || r == '_' || r == '.' || r == '-' || r == '$'
}

func isASCIILetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

func isASCIIDigitRune(r rune) bool {
	return r >= '0' && r <= '9'
}

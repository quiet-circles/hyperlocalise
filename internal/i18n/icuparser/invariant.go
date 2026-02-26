package icuparser

import (
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
}

func ParseInvariant(s string) (Invariant, error) {
	elems, err := Parse(s, nil)
	if err != nil {
		return Invariant{}, err
	}

	inv := Invariant{}
	collectInvariantFromElements(elems, &inv)

	sort.Strings(inv.Placeholders)
	sort.Slice(inv.ICUBlocks, func(i, j int) bool {
		if inv.ICUBlocks[i].Arg != inv.ICUBlocks[j].Arg {
			return inv.ICUBlocks[i].Arg < inv.ICUBlocks[j].Arg
		}
		if inv.ICUBlocks[i].Type != inv.ICUBlocks[j].Type {
			return inv.ICUBlocks[i].Type < inv.ICUBlocks[j].Type
		}
		return strings.Join(inv.ICUBlocks[i].Options, "\x00") < strings.Join(inv.ICUBlocks[j].Options, "\x00")
	})
	return inv, nil
}

func collectInvariantFromElements(elems []Element, inv *Invariant) {
	for _, el := range elems {
		switch v := el.(type) {
		case ArgumentElement:
			if isPlaceholderName(v.Value) {
				inv.Placeholders = append(inv.Placeholders, v.Value)
			}
		case NumberElement:
			if isPlaceholderName(v.Value) {
				inv.Placeholders = append(inv.Placeholders, v.Value)
			}
		case DateElement:
			if isPlaceholderName(v.Value) {
				inv.Placeholders = append(inv.Placeholders, v.Value)
			}
		case TimeElement:
			if isPlaceholderName(v.Value) {
				inv.Placeholders = append(inv.Placeholders, v.Value)
			}
		case SelectElement:
			inv.ICUBlocks = append(inv.ICUBlocks, BlockSignature{
				Arg:     v.Value,
				Type:    "select",
				Options: sortedSelectors(v.Options),
			})
			for _, opt := range v.Options {
				collectInvariantFromElements(opt.Value, inv)
			}
		case PluralElement:
			blockType := "plural"
			if v.Type() == TypeSelectOrdinal {
				blockType = "selectordinal"
			}
			inv.ICUBlocks = append(inv.ICUBlocks, BlockSignature{
				Arg:     v.Value,
				Type:    blockType,
				Options: sortedPluralSelectors(v.Options),
			})
			for _, opt := range v.Options {
				collectInvariantFromElements(opt.Value, inv)
			}
		case TagElement:
			collectInvariantFromElements(v.Children, inv)
		case LiteralElement, PoundElement:
			continue
		default:
			// Defensive no-op for future element types.
			continue
		}
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

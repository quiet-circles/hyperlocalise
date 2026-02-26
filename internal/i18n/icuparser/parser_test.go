package icuparser

import (
	"reflect"
	"strings"
	"testing"
)

func TestParserFeatureParitySubset(t *testing.T) {
	tests := []struct {
		name             string
		msg              string
		wantPlaceholders []string
		wantICU          []BlockSignature
	}{
		{
			name:             "plain placeholder argument",
			msg:              "Hi {name}",
			wantPlaceholders: []string{"name"},
		},
		{
			name:             "simple formatter argument is preserved for parity",
			msg:              "Date {ts, date, ::yyyyMMdd}",
			wantPlaceholders: []string{"ts"},
		},
		{
			name:             "plural with selectors and nested placeholders",
			msg:              "Hi {name}. {count, plural, =0 {No items} one {One item for {name}} other {{count} items}}",
			wantPlaceholders: []string{"count", "name", "name"},
			wantICU: []BlockSignature{{
				Arg:     "count",
				Type:    "plural",
				Options: []string{"=0", "one", "other"},
			}},
		},
		{
			name:             "nested select plural",
			msg:              "{gender, select, male {{count, plural, one {He has one} other {He has #}}} female {{count, plural, one {She has one} other {She has #}}} other {They have {count}}}",
			wantPlaceholders: []string{"count"},
			wantICU: []BlockSignature{
				{Arg: "count", Type: "plural", Options: []string{"one", "other"}},
				{Arg: "count", Type: "plural", Options: []string{"one", "other"}},
				{Arg: "gender", Type: "select", Options: []string{"female", "male", "other"}},
			},
		},
		{
			name:             "plural offset accepted",
			msg:              "{count, plural, offset:1 =0 {Nobody} one {{name}} other {{name} and # others}}",
			wantPlaceholders: []string{"name", "name"},
			wantICU: []BlockSignature{{
				Arg:     "count",
				Type:    "plural",
				Options: []string{"=0", "one", "other"},
			}},
		},
		{
			name:             "selectordinal accepted",
			msg:              "{pos, selectordinal, one {#st} two {#nd} few {#rd} other {#th}}",
			wantPlaceholders: nil,
			wantICU: []BlockSignature{{
				Arg:     "pos",
				Type:    "selectordinal",
				Options: []string{"few", "one", "other", "two"},
			}},
		},
		{
			name:             "quoted braces are treated as literals",
			msg:              "'{not-an-arg}' and '' quote and {actual}",
			wantPlaceholders: []string{"actual"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseInvariant(tt.msg)
			if err != nil {
				t.Fatalf("parse invariant: %v", err)
			}
			if !reflect.DeepEqual(got.Placeholders, tt.wantPlaceholders) {
				t.Fatalf("unexpected placeholders: got %#v want %#v", got.Placeholders, tt.wantPlaceholders)
			}
			if !reflect.DeepEqual(got.ICUBlocks, tt.wantICU) {
				t.Fatalf("unexpected ICU blocks: got %#v want %#v", got.ICUBlocks, tt.wantICU)
			}
		})
	}
}

func TestParserInvalidSyntax(t *testing.T) {
	tests := []struct {
		name        string
		msg         string
		errContains string
	}{
		{name: "unbalanced plural", msg: "{count, plural, one {ok} other {missing}", errContains: "unclosed"},
		{name: "missing option body", msg: "{count, plural, one other {x}}", errContains: "ICU option body"},
		{name: "unexpected top level closing brace", msg: "hello }", errContains: "unexpected closing brace"},
		{name: "plural requires options", msg: "{count, plural, }", errContains: "missing options"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseInvariant(tt.msg)
			if err == nil {
				t.Fatalf("expected parse error")
			}
			if !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errContains)) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

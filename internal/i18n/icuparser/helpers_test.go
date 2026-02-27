package icuparser

import "testing"

func TestIsASCIIDigitRune(t *testing.T) {
	tests := []struct {
		name string
		r    rune
		want bool
	}{
		{name: "zero", r: '0', want: true},
		{name: "nine", r: '9', want: true},
		{name: "letter", r: 'a', want: false},
		{name: "unicode digit", r: '٣', want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isASCIIDigitRune(tt.r); got != tt.want {
				t.Fatalf("unexpected result: got %v want %v", got, tt.want)
			}
		})
	}
}

func TestSkipQuotedLiteral(t *testing.T) {
	tests := []struct {
		name    string
		src     string
		wantPos int
	}{
		{
			name:    "normal quoted literal",
			src:     "'abc' rest",
			wantPos: 5,
		},
		{
			name:    "escaped apostrophe within quoted literal",
			src:     "'a''b' rest",
			wantPos: 6,
		},
		{
			name:    "unclosed quoted literal advances to end",
			src:     "'abc",
			wantPos: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := astParser{src: tt.src, pos: 0}
			p.skipQuotedLiteral()
			if p.pos != tt.wantPos {
				t.Fatalf("unexpected parser position: got %d want %d", p.pos, tt.wantPos)
			}
		})
	}
}

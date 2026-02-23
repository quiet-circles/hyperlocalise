package convert

import (
	"testing"
)

func TestToInteger(t *testing.T) {
	testCases := []struct {
		name     string
		input    interface{}
		expected int
		err      bool
	}{
		{
			name:     "valid int",
			input:    1,
			expected: 1,
			err:      false,
		},
		{
			name:  "invalid string",
			input: "s",
			err:   true,
		},
		{
			name:  "invalid bool",
			input: false,
			err:   true,
		},
	}

	for _, tc := range testCases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			res, err := ToInteger(tc.input)

			if tc.err {
				if err == nil {
					t.Fatalf("expected conversion error for %v", tc.input)
				}

				if got, want := err.Error(), errConversionError(tc.input).Error(); got != want {
					t.Fatalf("unexpected error: got %q want %q", got, want)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				if res != tc.expected {
					t.Fatalf("unexpected result: got %d want %d", res, tc.expected)
				}
			}
		})
	}
}

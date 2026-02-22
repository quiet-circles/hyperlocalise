package example

import (
	"testing"
)

func TestSum(t *testing.T) {
	testCases := []struct {
		name     string
		a        int
		b        int
		expected int
	}{
		{
			name:     "1 plus 2",
			a:        1,
			b:        2,
			expected: 3,
		},
		{
			name:     "2 plus 2",
			a:        2,
			b:        2,
			expected: 4,
		},
	}

	for _, tc := range testCases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			if got := Add(tc.a, tc.b); got != tc.expected {
				t.Fatalf("unexpected sum: got %d want %d", got, tc.expected)
			}
		})
	}
}

func TestMultiply(t *testing.T) {
	testCases := []struct {
		name     string
		a        int
		b        int
		expected int
	}{
		{
			name:     "1 times 2",
			a:        1,
			b:        2,
			expected: 2,
		},
		{
			name:     "2 times 2",
			a:        2,
			b:        2,
			expected: 4,
		},
	}

	for _, tc := range testCases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			if got := Multiply(tc.a, tc.b); got != tc.expected {
				t.Fatalf("unexpected product: got %d want %d", got, tc.expected)
			}
		})
	}
}

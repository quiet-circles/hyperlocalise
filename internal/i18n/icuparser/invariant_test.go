package icuparser

import "testing"

func TestSamePlaceholderSetIgnoresOrderAndDuplicates(t *testing.T) {
	if !SamePlaceholderSet([]string{"b", "a", "b"}, []string{"a", "b"}) {
		t.Fatalf("expected placeholder sets to match")
	}
}

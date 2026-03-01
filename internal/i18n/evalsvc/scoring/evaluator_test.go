package scoring

import (
	"slices"
	"testing"
)

func TestEvaluatorDetectsPlaceholderDrop(t *testing.T) {
	e := NewEvaluator()
	got := e.Evaluate("Hello {name}, total is %s", "Bonjour, total est %s", "")

	if got.PlaceholderIntegrity >= 1 {
		t.Fatalf("expected placeholder integrity penalty, got %+v", got)
	}
	if !slices.Contains(got.HardFails, HardFailPlaceholderDrop) {
		t.Fatalf("expected placeholder hard fail, got %+v", got.HardFails)
	}
	if got.WeightedAggregate != 0 {
		t.Fatalf("expected hard-failed weighted aggregate=0, got %v", got.WeightedAggregate)
	}
}

func TestEvaluatorHandlesICUPluralIntegrity(t *testing.T) {
	e := NewEvaluator()
	source := "{count, plural, one {# file} other {# files}} uploaded by {name}"
	translated := "{count, plural, one {# fichier} other {# fichiers}} téléchargés par {name}"

	got := e.Evaluate(source, translated, "")
	if got.PlaceholderIntegrity != 1 {
		t.Fatalf("expected full ICU placeholder integrity, got %+v", got)
	}
	if len(got.HardFails) != 0 {
		t.Fatalf("expected no hard fails, got %+v", got.HardFails)
	}
}

func TestEvaluatorDetectsMalformedICU(t *testing.T) {
	e := NewEvaluator()
	source := "{count, plural, one {One} other {Many}}"
	translated := "{count, plural, one {Uno} other {Muchos}"

	got := e.Evaluate(source, translated, "")
	if !slices.Contains(got.HardFails, HardFailMalformedICU) {
		t.Fatalf("expected malformed ICU hard fail, got %+v", got.HardFails)
	}
}

func TestEvaluatorReferenceScores(t *testing.T) {
	e := NewEvaluator()
	got := e.Evaluate("Pay now", "Payer maintenant", "Payer maintenant!")

	if got.ReferenceExact == nil || *got.ReferenceExact != 0 {
		t.Fatalf("expected exact mismatch, got %+v", got.ReferenceExact)
	}
	if got.ReferenceNormalized == nil || *got.ReferenceNormalized != 1 {
		t.Fatalf("expected normalized match, got %+v", got.ReferenceNormalized)
	}
	if got.ReferenceSimilarity == nil || *got.ReferenceSimilarity < 0.9 {
		t.Fatalf("expected high similarity score, got %+v", got.ReferenceSimilarity)
	}
}

func TestEvaluatorHardFailSourceCopied(t *testing.T) {
	e := NewEvaluator()
	got := e.Evaluate("Save", "Save", "Enregistrer")
	if !slices.Contains(got.HardFails, HardFailSourceCopied) {
		t.Fatalf("expected source copied hard fail, got %+v", got.HardFails)
	}
	if got.WeightedAggregate != 0 {
		t.Fatalf("expected aggregate hard fail to 0, got %v", got.WeightedAggregate)
	}
}

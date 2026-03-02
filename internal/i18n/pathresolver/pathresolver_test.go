package pathresolver

import "testing"

func TestResolveTargetPathLocaleDirForRootOutput(t *testing.T) {
	got := ResolveTargetPath("docs/{{localeDir}}/index.mdx", "en", "en")
	if got != "docs/index.mdx" {
		t.Fatalf("ResolveTargetPath root output = %q, want %q", got, "docs/index.mdx")
	}
}

func TestResolveTargetPathLocaleDirForNonSourceTarget(t *testing.T) {
	got := ResolveTargetPath("docs/{{localeDir}}/index.mdx", "en", "fr")
	if got != "docs/fr/index.mdx" {
		t.Fatalf("ResolveTargetPath locale dir = %q, want %q", got, "docs/fr/index.mdx")
	}
}

func TestResolveTargetPathBackCompat(t *testing.T) {
	got := ResolveTargetPath("lang/[locale].{{target}}.json", "en", "es")
	if got != "lang/es.es.json" {
		t.Fatalf("ResolveTargetPath legacy template = %q, want %q", got, "lang/es.es.json")
	}
}

func TestResolveSourcePath(t *testing.T) {
	got := ResolveSourcePath("content/{{source}}/[locale].json", "en")
	if got != "content/en/en.json" {
		t.Fatalf("ResolveSourcePath = %q, want %q", got, "content/en/en.json")
	}
}

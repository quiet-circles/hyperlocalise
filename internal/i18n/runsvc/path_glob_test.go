package runsvc

import (
	"os"
	"path/filepath"
	"testing"
)

func TestShouldIgnoreSourcePathScenarios(t *testing.T) {
	targets := []string{"fr", "es", "zh"}
	tests := []struct {
		name   string
		path   string
		target []string
		want   bool
	}{
		{name: "locale as middle segment", path: "docs/fr/index.mdx", target: targets, want: true},
		{name: "nested locale segment", path: "docs/es/guides/quickstart.mdx", target: targets, want: true},
		{name: "no locale segment", path: "docs/guides/quickstart.mdx", target: targets, want: false},
		{name: "single segment path", path: "index.mdx", target: targets, want: false},
		{name: "locale as first segment", path: "fr/docs/index.mdx", target: targets, want: false},
		{name: "locale only in filename", path: "docs/guides/fr.mdx", target: targets, want: false},
		{name: "windows style separators treated literally on unix", path: `docs\zh\index.mdx`, target: targets, want: false},
		{name: "empty target locales", path: "docs/fr/index.mdx", target: nil, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldIgnoreSourcePath(tt.path, tt.target)
			if got != tt.want {
				t.Fatalf("shouldIgnoreSourcePath(%q, %v) = %t, want %t", tt.path, tt.target, got, tt.want)
			}
		})
	}
}

func TestResolveSourcePathsNoGlobReturnsInput(t *testing.T) {
	pattern := filepath.Join(t.TempDir(), "docs", "index.mdx")
	paths, err := resolveSourcePaths(pattern)
	if err != nil {
		t.Fatalf("resolveSourcePaths returned error: %v", err)
	}
	if len(paths) != 1 || paths[0] != pattern {
		t.Fatalf("resolveSourcePaths(%q) = %v, want [%q]", pattern, paths, pattern)
	}
}

func TestResolveSourcePathsSingleStarSorted(t *testing.T) {
	dir := t.TempDir()
	docs := filepath.Join(dir, "docs")
	if err := os.MkdirAll(docs, 0o755); err != nil {
		t.Fatalf("mkdir docs: %v", err)
	}
	files := []string{"b.mdx", "a.mdx", "c.txt"}
	for _, name := range files {
		if err := os.WriteFile(filepath.Join(docs, name), []byte(name), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	pattern := filepath.Join(docs, "*.mdx")
	got, err := resolveSourcePaths(pattern)
	if err != nil {
		t.Fatalf("resolveSourcePaths returned error: %v", err)
	}

	want := []string{
		filepath.Join(docs, "a.mdx"),
		filepath.Join(docs, "b.mdx"),
	}
	if len(got) != len(want) {
		t.Fatalf("resolveSourcePaths(%q) length = %d, want %d (%v)", pattern, len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("resolveSourcePaths(%q)[%d] = %q, want %q", pattern, i, got[i], want[i])
		}
	}
}

func TestResolveSourcePathsSingleStarInvalidPattern(t *testing.T) {
	_, err := resolveSourcePaths("[")
	if err == nil {
		t.Fatal("expected filepath.Glob error for invalid pattern")
	}
}

func TestResolveSourcePathsDoublestarRecursiveSortedAndFilesOnly(t *testing.T) {
	dir := t.TempDir()
	docs := filepath.Join(dir, "docs")
	if err := os.MkdirAll(filepath.Join(docs, "guides", "nested"), 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(docs, "api"), 0o755); err != nil {
		t.Fatalf("mkdir api: %v", err)
	}

	paths := []string{
		filepath.Join(docs, "index.mdx"),
		filepath.Join(docs, "guides", "quickstart.mdx"),
		filepath.Join(docs, "guides", "nested", "advanced.mdx"),
		filepath.Join(docs, "api", "reference.txt"),
	}
	for _, path := range paths {
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	pattern := filepath.Join(docs, "**", "*.mdx")
	got, err := resolveSourcePaths(pattern)
	if err != nil {
		t.Fatalf("resolveSourcePaths returned error: %v", err)
	}

	want := []string{
		filepath.Join(docs, "guides", "nested", "advanced.mdx"),
		filepath.Join(docs, "guides", "quickstart.mdx"),
		filepath.Join(docs, "index.mdx"),
	}
	if len(got) != len(want) {
		t.Fatalf("resolveSourcePaths(%q) length = %d, want %d (%v)", pattern, len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("resolveSourcePaths(%q)[%d] = %q, want %q", pattern, i, got[i], want[i])
		}
	}
}

func TestResolveSourcePathsDoublestarNonexistentBaseDir(t *testing.T) {
	pattern := filepath.Join(t.TempDir(), "nonexistent", "**", "*.mdx")
	_, err := resolveSourcePaths(pattern)
	if err == nil {
		t.Fatal("expected error for nonexistent base directory, got nil")
	}
}

func TestResolveTargetPathSourceOutsideBase(t *testing.T) {
	_, err := resolveTargetPath("docs/**/*.mdx", "out/fr/**/*.mdx", "other/file.mdx")
	if err == nil {
		t.Fatal("expected error when source path escapes source glob base")
	}
}

func TestResolveTargetPathScenarios(t *testing.T) {
	tests := []struct {
		name          string
		sourcePattern string
		targetPattern string
		sourcePath    string
		want          string
		wantErr       bool
	}{
		{
			name:          "no source glob returns target pattern directly",
			sourcePattern: "docs/index.mdx",
			targetPattern: "docs/fr/index.mdx",
			sourcePath:    "docs/index.mdx",
			want:          "docs/fr/index.mdx",
		},
		{
			name:          "glob mapping preserves relative structure",
			sourcePattern: "docs/**/*.mdx",
			targetPattern: "docs/fr/**/*.mdx",
			sourcePath:    "docs/guides/quickstart.mdx",
			want:          filepath.Join("docs", "fr", "guides", "quickstart.mdx"),
		},
		{
			name:          "source has glob but target does not",
			sourcePattern: "docs/**/*.mdx",
			targetPattern: "docs/fr/index.mdx",
			sourcePath:    "docs/index.mdx",
			wantErr:       true,
		},
		{
			name:          "single-star mapping",
			sourcePattern: "docs/*.mdx",
			targetPattern: "docs/fr/*.mdx",
			sourcePath:    "docs/index.mdx",
			want:          filepath.Join("docs", "fr", "index.mdx"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveTargetPath(tt.sourcePattern, tt.targetPattern, tt.sourcePath)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("resolveTargetPath(%q, %q, %q) expected error, got nil", tt.sourcePattern, tt.targetPattern, tt.sourcePath)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveTargetPath(%q, %q, %q) returned error: %v", tt.sourcePattern, tt.targetPattern, tt.sourcePath, err)
			}
			if got != tt.want {
				t.Fatalf("resolveTargetPath(%q, %q, %q) = %q, want %q", tt.sourcePattern, tt.targetPattern, tt.sourcePath, got, tt.want)
			}
		})
	}
}

func TestBaseDirForDoublestar(t *testing.T) {
	tests := []struct {
		pattern string
		want    string
	}{
		{pattern: "docs/**/*.mdx", want: "docs"},
		{pattern: "docs/**/nested/*.mdx", want: "docs"},
		{pattern: "**/*.mdx", want: "."},
		{pattern: "docs/*.mdx", want: "docs"},
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			got := baseDirForDoublestar(tt.pattern)
			if got != tt.want {
				t.Fatalf("baseDirForDoublestar(%q) = %q, want %q", tt.pattern, got, tt.want)
			}
		})
	}
}

func TestGlobBaseDir(t *testing.T) {
	tests := []struct {
		pattern string
		want    string
	}{
		{pattern: "docs/**/*.mdx", want: "docs"},
		{pattern: "docs/*/index.mdx", want: "docs"},
		{pattern: "docs/??/index.mdx", want: "docs"},
		{pattern: "[ab].mdx", want: "."},
		{pattern: "docs/index.mdx", want: "docs"},
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			got := globBaseDir(tt.pattern)
			if got != tt.want {
				t.Fatalf("globBaseDir(%q) = %q, want %q", tt.pattern, got, tt.want)
			}
		})
	}
}

func TestGlobToRegexMatchingSemantics(t *testing.T) {
	type probe struct {
		path string
		want bool
	}
	tests := []struct {
		name    string
		pattern string
		probes  []probe
	}{
		{
			name:    "single star does not cross slash",
			pattern: "docs/*.mdx",
			probes: []probe{
				{path: "docs/index.mdx", want: true},
				{path: "docs/a/b.mdx", want: false},
			},
		},
		{
			name:    "question matches one char",
			pattern: "docs/?.mdx",
			probes: []probe{
				{path: "docs/a.mdx", want: true},
				{path: "docs/ab.mdx", want: false},
			},
		},
		{
			name:    "doublestar slash supports zero or many dirs",
			pattern: "docs/**/index.mdx",
			probes: []probe{
				{path: "docs/index.mdx", want: true},
				{path: "docs/guides/index.mdx", want: true},
				{path: "docs/guides/nested/index.mdx", want: true},
			},
		},
		{
			name:    "doublestar without slash spans everything",
			pattern: "docs/**.mdx",
			probes: []probe{
				{path: "docs/index.mdx", want: true},
				{path: "docs/guides/quickstart.mdx", want: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			re, err := globToRegex(tt.pattern)
			if err != nil {
				t.Fatalf("globToRegex(%q) returned error: %v", tt.pattern, err)
			}
			for _, p := range tt.probes {
				if got := re.MatchString(p.path); got != p.want {
					t.Fatalf("globToRegex(%q) matches %q = %t, want %t (regex=%q)", tt.pattern, p.path, got, p.want, re.String())
				}
			}
		})
	}
}

func TestGlobToRegexCharacterClassEscaped(t *testing.T) {
	// globToRegex processes [ ] characters through QuoteMeta, so character
	// classes like [abc] are escaped and treated as literal brackets.
	re, err := globToRegex("docs/[ab].mdx")
	if err != nil {
		t.Fatalf("globToRegex returned error: %v", err)
	}
	if re.MatchString("docs/a.mdx") {
		t.Fatalf("globToRegex(%q) should NOT match %q (brackets are escaped)", "docs/[ab].mdx", "docs/a.mdx")
	}
	if !re.MatchString("docs/[ab].mdx") {
		t.Fatalf("globToRegex(%q) should match literal brackets path", "docs/[ab].mdx")
	}
}

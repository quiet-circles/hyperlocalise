package runsvc

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestShouldIgnoreSourcePath(t *testing.T) {
	tests := []struct {
		name          string
		sourcePath    string
		targetLocales []string
		want          bool
	}{
		{
			name:          "single segment path never ignored",
			sourcePath:    "app.json",
			targetLocales: []string{"fr", "de"},
			want:          false,
		},
		{
			name:          "no intermediate locale segment",
			sourcePath:    filepath.Join("messages", "app.json"),
			targetLocales: []string{"fr", "de"},
			want:          false,
		},
		{
			name:          "locale segment in middle is ignored",
			sourcePath:    filepath.Join("messages", "fr", "app.json"),
			targetLocales: []string{"fr", "de"},
			want:          true,
		},
		{
			name:          "first segment is not treated as locale",
			sourcePath:    filepath.Join("fr", "app.json"),
			targetLocales: []string{"fr", "de"},
			want:          false,
		},
		{
			name:          "last segment is filename not locale",
			sourcePath:    filepath.Join("messages", "fr"),
			targetLocales: []string{"fr"},
			want:          false,
		},
		{
			name:          "empty target locales never ignore",
			sourcePath:    filepath.Join("messages", "fr", "app.json"),
			targetLocales: nil,
			want:          false,
		},
		{
			name:          "normalizes slash separated input",
			sourcePath:    "messages/fr/app.json",
			targetLocales: []string{"fr"},
			want:          true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldIgnoreSourcePath(tc.sourcePath, tc.targetLocales); got != tc.want {
				t.Fatalf("shouldIgnoreSourcePath(%q, %v) = %v, want %v", tc.sourcePath, tc.targetLocales, got, tc.want)
			}
		})
	}
}

func TestResolveSourcePathsWithoutGlobReturnsInput(t *testing.T) {
	got, err := resolveSourcePaths("messages/en/app.json")
	if err != nil {
		t.Fatalf("resolveSourcePaths returned error: %v", err)
	}
	want := []string{"messages/en/app.json"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("resolveSourcePaths returned %v, want %v", got, want)
	}
}

func TestResolveSourcePathsInvalidSingleStarPatternReturnsError(t *testing.T) {
	_, err := resolveSourcePaths("messages/[.json")
	if err == nil {
		t.Fatal("resolveSourcePaths expected an error for invalid glob pattern")
	}
}

func TestResolveSourcePathsSingleStarSortsMatches(t *testing.T) {
	dir := t.TempDir()
	alpha := filepath.Join(dir, "a.json")
	beta := filepath.Join(dir, "b.json")
	nested := filepath.Join(dir, "nested", "c.json")
	for _, p := range []string{beta, alpha, nested} {
		if err := mkdirAndWriteFile(p); err != nil {
			t.Fatalf("failed to create %s: %v", p, err)
		}
	}

	got, err := resolveSourcePaths(filepath.Join(dir, "*.json"))
	if err != nil {
		t.Fatalf("resolveSourcePaths returned error: %v", err)
	}
	want := []string{alpha, beta}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("resolveSourcePaths returned %v, want %v", got, want)
	}
}

func TestResolveSourcePathsDoubleStarMatchesRecursively(t *testing.T) {
	dir := t.TempDir()
	rootFile := filepath.Join(dir, "messages.json")
	nestedFile := filepath.Join(dir, "nested", "deeper", "messages.json")
	nonMatch := filepath.Join(dir, "nested", "deeper", "messages.txt")
	for _, p := range []string{rootFile, nestedFile, nonMatch} {
		if err := mkdirAndWriteFile(p); err != nil {
			t.Fatalf("failed to create %s: %v", p, err)
		}
	}

	got, err := resolveSourcePaths(filepath.Join(dir, "**", "*.json"))
	if err != nil {
		t.Fatalf("resolveSourcePaths returned error: %v", err)
	}
	want := []string{nestedFile, rootFile}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("resolveSourcePaths returned %v, want %v", got, want)
	}
}

func TestResolveSourcePathsDoubleStarInfixMatchesAcrossLevels(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, "en", "core", "app.json")
	nested := filepath.Join(dir, "en", "core", "feature", "app.json")
	nonMatchName := filepath.Join(dir, "en", "core", "feature", "other.json")
	for _, p := range []string{root, nested, nonMatchName} {
		if err := mkdirAndWriteFile(p); err != nil {
			t.Fatalf("failed to create %s: %v", p, err)
		}
	}

	pattern := filepath.Join(dir, "en", "**", "app.json")
	got, err := resolveSourcePaths(pattern)
	if err != nil {
		t.Fatalf("resolveSourcePaths returned error: %v", err)
	}
	want := []string{nested, root}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("resolveSourcePaths returned %v, want %v", got, want)
	}
}

func TestResolveSourcePathsNoDoubleStarMatchesReturnsEmptySlice(t *testing.T) {
	dir := t.TempDir()
	got, err := resolveSourcePaths(filepath.Join(dir, "**", "*.json"))
	if err != nil {
		t.Fatalf("resolveSourcePaths returned error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("resolveSourcePaths returned %v, want empty result", got)
	}
}

func TestResolveTargetPath(t *testing.T) {
	tests := []struct {
		name          string
		sourcePattern string
		targetPattern string
		sourcePath    string
		want          string
		wantErr       bool
	}{
		{
			name:          "literal source returns literal target",
			sourcePattern: "messages/en/app.json",
			targetPattern: "messages/fr/app.json",
			sourcePath:    "messages/en/app.json",
			want:          "messages/fr/app.json",
		},
		{
			name:          "glob source requires glob target",
			sourcePattern: "messages/**/*.json",
			targetPattern: "messages/fr/app.json",
			sourcePath:    "messages/en/app.json",
			wantErr:       true,
		},
		{
			name:          "glob source maps to relative target path",
			sourcePattern: filepath.Join("messages", "**", "*.json"),
			targetPattern: filepath.Join("translations", "**", "*.json"),
			sourcePath:    filepath.Join("messages", "en", "app.json"),
			want:          filepath.Join("translations", "en", "app.json"),
		},
		{
			name:          "relative traversal from source base is preserved",
			sourcePattern: filepath.Join("messages", "*", "*.json"),
			targetPattern: filepath.Join("out", "*", "*.json"),
			sourcePath:    filepath.Join("messages", "en", "nested", "app.json"),
			want:          filepath.Join("out", "en", "nested", "app.json"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveTargetPath(tc.sourcePattern, tc.targetPattern, tc.sourcePath)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("resolveTargetPath(%q, %q, %q) expected error", tc.sourcePattern, tc.targetPattern, tc.sourcePath)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveTargetPath returned error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("resolveTargetPath returned %q, want %q", got, tc.want)
			}
		})
	}
}

func TestBaseDirForDoublestar(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		want    string
	}{
		{name: "simple subdir", pattern: filepath.Join("messages", "**", "*.json"), want: "messages"},
		{name: "no prefix", pattern: "**/*.json", want: "."},
		{name: "pattern without doublestar uses dir", pattern: filepath.Join("messages", "*.json"), want: "messages"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := baseDirForDoublestar(tc.pattern); got != tc.want {
				t.Fatalf("baseDirForDoublestar(%q) returned %q, want %q", tc.pattern, got, tc.want)
			}
		})
	}
}

func TestGlobBaseDir(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		want    string
	}{
		{name: "wildcard in child segment", pattern: filepath.Join("messages", "*", "app.json"), want: "messages"},
		{name: "wildcard at root", pattern: "*.json", want: "."},
		{name: "no wildcard returns directory", pattern: filepath.Join("messages", "app.json"), want: "messages"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := globBaseDir(tc.pattern); got != tc.want {
				t.Fatalf("globBaseDir(%q) returned %q, want %q", tc.pattern, got, tc.want)
			}
		})
	}
}

func TestGlobToRegex(t *testing.T) {
	re, err := globToRegex("messages/**/app?.json")
	if err != nil {
		t.Fatalf("globToRegex returned error: %v", err)
	}

	matching := []string{
		"messages/app1.json",
		"messages/en/app2.json",
		"messages/en/nested/appA.json",
	}
	for _, candidate := range matching {
		if !re.MatchString(candidate) {
			t.Fatalf("regex did not match expected candidate %q", candidate)
		}
	}

	nonMatching := []string{
		"messages/en/app10.json",
		"messages/app.json",
		"messages/en/nested/notapp1.json",
	}
	for _, candidate := range nonMatching {
		if re.MatchString(candidate) {
			t.Fatalf("regex unexpectedly matched candidate %q", candidate)
		}
	}
}

func mkdirAndWriteFile(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte("x"), 0o644)
}

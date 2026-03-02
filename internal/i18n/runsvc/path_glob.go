package runsvc

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

func shouldIgnoreSourcePath(sourcePath string, targetLocales []string) bool {
	normalized := filepath.ToSlash(sourcePath)
	segments := strings.Split(normalized, "/")
	if len(segments) < 2 {
		return false
	}

	targets := make(map[string]struct{}, len(targetLocales))
	for _, locale := range targetLocales {
		targets[locale] = struct{}{}
	}

	for i := 1; i < len(segments)-1; i++ {
		if _, ok := targets[segments[i]]; ok {
			return true
		}
	}
	return false
}

func resolveSourcePaths(sourcePattern string) ([]string, error) {
	if !strings.ContainsAny(sourcePattern, "*?[") {
		return []string{sourcePattern}, nil
	}
	if !strings.Contains(sourcePattern, "**") {
		matches, err := filepath.Glob(sourcePattern)
		if err != nil {
			return nil, err
		}
		slices.Sort(matches)
		return matches, nil
	}

	normalizedPattern := filepath.ToSlash(sourcePattern)
	re, err := globToRegex(normalizedPattern)
	if err != nil {
		return nil, err
	}

	baseDir := baseDirForDoublestar(sourcePattern)
	matches := make([]string, 0)
	err = filepath.WalkDir(baseDir, func(candidate string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if re.MatchString(filepath.ToSlash(candidate)) {
			matches = append(matches, candidate)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	slices.Sort(matches)
	return matches, nil
}

func resolveTargetPath(sourcePattern, targetPattern, sourcePath string) (string, error) {
	if !strings.ContainsAny(sourcePattern, "*?[") {
		return targetPattern, nil
	}
	if !strings.ContainsAny(targetPattern, "*?[") {
		return "", fmt.Errorf("target pattern %q must include glob tokens when source pattern %q includes globs", targetPattern, sourcePattern)
	}
	sourceBase := globBaseDir(sourcePattern)
	targetBase := globBaseDir(targetPattern)
	relative, err := filepath.Rel(sourceBase, sourcePath)
	if err != nil {
		return "", err
	}
	return filepath.Join(targetBase, relative), nil
}

func baseDirForDoublestar(pattern string) string {
	normalized := filepath.ToSlash(pattern)
	idx := strings.Index(normalized, "**")
	if idx == -1 {
		return filepath.Dir(pattern)
	}
	prefix := strings.TrimSuffix(normalized[:idx], "/")
	if prefix == "" {
		return "."
	}
	return filepath.FromSlash(prefix)
}

func globBaseDir(pattern string) string {
	idx := strings.IndexAny(filepath.ToSlash(pattern), "*?[")
	if idx == -1 {
		return filepath.Dir(pattern)
	}
	prefix := filepath.ToSlash(pattern)[:idx]
	prefix = strings.TrimSuffix(prefix, "/")
	if prefix == "" {
		return "."
	}
	return filepath.FromSlash(prefix)
}

func globToRegex(pattern string) (*regexp.Regexp, error) {
	var b strings.Builder
	b.WriteString("^")
	for i := 0; i < len(pattern); {
		switch pattern[i] {
		case '*':
			if i+1 < len(pattern) && pattern[i+1] == '*' {
				if i+2 < len(pattern) && pattern[i+2] == '/' {
					b.WriteString("(?:.*/)?")
					i += 3
					continue
				}
				b.WriteString(".*")
				i += 2
				continue
			}
			b.WriteString("[^/]*")
		case '?':
			b.WriteString("[^/]")
		default:
			b.WriteString(regexp.QuoteMeta(pattern[i : i+1]))
		}
		i++
	}
	b.WriteString("$")
	return regexp.Compile(b.String())
}

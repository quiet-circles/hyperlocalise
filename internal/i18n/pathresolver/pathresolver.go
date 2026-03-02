package pathresolver

import "strings"

const (
	tokenSource    = "{{source}}"
	tokenTarget    = "{{target}}"
	tokenLocaleDir = "{{localeDir}}"
	legacyLocale   = "[locale]"
)

func ResolveSourcePath(pattern, sourceLocale string) string {
	return resolve(pattern, sourceLocale, sourceLocale)
}

func ResolveTargetPath(pattern, sourceLocale, targetLocale string) string {
	return resolve(pattern, sourceLocale, targetLocale)
}

func resolve(pattern, sourceLocale, targetLocale string) string {
	localeDir := targetLocale
	if sourceLocale == targetLocale {
		localeDir = ""
	}

	path := strings.ReplaceAll(pattern, tokenSource, sourceLocale)
	path = strings.ReplaceAll(path, tokenTarget, targetLocale)
	path = strings.ReplaceAll(path, tokenLocaleDir, localeDir)
	path = strings.ReplaceAll(path, legacyLocale, targetLocale)

	for strings.Contains(path, "//") {
		path = strings.ReplaceAll(path, "//", "/")
	}

	return path
}

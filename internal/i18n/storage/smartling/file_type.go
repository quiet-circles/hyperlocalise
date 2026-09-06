package smartling

import (
	"fmt"
	"strings"
)

// FileTypeForExtension returns the Smartling fileType for a filename extension.
// ext must include the leading dot (e.g. ".json"); callers typically use
// strings.ToLower(filepath.Ext(path)). Unknown extensions return an empty string.
func FileTypeForExtension(ext string) string {
	switch strings.ToLower(ext) {
	case ".json":
		return "json"
	case ".yaml", ".yml":
		return "yaml"
	case ".xml":
		return "xml"
	case ".html", ".htm":
		return "html"
	case ".csv":
		return "csv"
	case ".strings":
		return "ios"
	case ".stringsdict":
		return "ios_stringsdict"
	case ".properties":
		return "javaProperties"
	case ".xliff", ".xlf":
		return "xliff"
	case ".md", ".markdown":
		return "markdown"
	default:
		return ""
	}
}

// CanonicalDirectiveField returns the Files API multipart field name for a parser directive.
func CanonicalDirectiveField(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	if strings.HasPrefix(key, "smartling.") {
		return key
	}
	return "smartling." + key
}

// NormalizeRetrievalType accepts pending, published, or pseudo. Empty means omit the query param.
func NormalizeRetrievalType(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", nil
	}
	switch strings.ToLower(trimmed) {
	case "pending", "published", "pseudo":
		return strings.ToLower(trimmed), nil
	default:
		return "", fmt.Errorf("smartling download: retrieval type must be pending, published, or pseudo")
	}
}

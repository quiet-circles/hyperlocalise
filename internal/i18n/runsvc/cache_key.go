package runsvc

import (
	"strings"
)

func normalizeSourceForCache(source string) string {
	normalized := strings.ReplaceAll(source, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	return strings.TrimSpace(normalized)
}

func contextMemoryFingerprint(task Task) string {
	return hashSourceText(strings.Join([]string{
		"context_key=" + strings.TrimSpace(task.ContextKey),
		"context_memory=" + normalizeSourceForCache(task.ContextMemory),
	}, "\n"))
}

func exactCacheKey(task Task) string {
	canonical := strings.Join([]string{
		"source_norm_hash=" + hashSourceText(normalizeSourceForCache(task.SourceText)),
		"source_locale=" + strings.TrimSpace(task.SourceLocale),
		"target_locale=" + strings.TrimSpace(task.TargetLocale),
		"provider=" + strings.TrimSpace(task.Provider),
		"model=" + strings.TrimSpace(task.Model),
		"profile=" + strings.TrimSpace(task.ProfileName),
		"prompt_version_hash=" + strings.TrimSpace(task.PromptVersion),
		"glossary_termbase_version_hash=" + strings.TrimSpace(task.GlossaryVersion),
		"parser_mode=" + strings.TrimSpace(task.ParserMode),
		"context_memory_fingerprint=" + contextMemoryFingerprint(task),
		"retrieval_corpus_snapshot_version=" + strings.TrimSpace(task.RAGSnapshot),
	}, "\n")
	return hashSourceText(canonical)
}

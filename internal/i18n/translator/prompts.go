package translator

import "strings"

func buildSystemPrompt(customPrompt string) string {
	base := strings.TrimSpace(customPrompt)
	if base == "" {
		base = "You are a translation assistant."
	}

	return base + " Return only the translated text with no explanations, labels, markdown, or quotes unless the translated content itself requires them."
}

func buildUserPrompt(req Request) string {
	b := strings.Builder{}
	b.WriteString("Translate the following source text into the requested target language. Preserve placeholders, variables, and formatting.\n\n")
	b.WriteString("Target language: ")
	b.WriteString(strings.TrimSpace(req.TargetLanguage))
	b.WriteString("\n")

	ctx := strings.TrimSpace(req.Context)
	if ctx != "" {
		b.WriteString("Context: ")
		b.WriteString(ctx)
		b.WriteString("\n")
	}

	b.WriteString("Source text:\n")
	b.WriteString(req.Source)
	return b.String()
}

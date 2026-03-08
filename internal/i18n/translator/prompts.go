package translator

import "strings"

func buildSystemPrompt(req Request) string {
	base := strings.TrimSpace(req.SystemPrompt)
	if base == "" {
		b := strings.Builder{}
		b.WriteString("You are a translation assistant. Translate the user-provided source text into the requested target language. Preserve meaning, placeholders, variables, and formatting. Do not translate programmatic identifiers inside placeholders or ICU message syntax. Keep ICU keywords and selectors such as plural, select, selectordinal, zero, one, two, few, many, other, explicit selectors like =0, and # unchanged. Return only the translated text with no explanations, labels, markdown, or quotes unless the translated content itself requires them.")

		target := strings.TrimSpace(req.TargetLanguage)
		if target != "" {
			b.WriteString("\nTarget language: ")
			b.WriteString(target)
		}

		base = b.String()
	}

	return appendRuntimeContextToSystemPrompt(base, req.RuntimeContext)
}

func buildUserPrompt(req Request) string {
	if custom := strings.TrimSpace(req.UserPrompt); custom != "" {
		return custom
	}

	b := strings.Builder{}
	b.WriteString("Translate the following source text into the requested target language. Preserve placeholders, variables, and formatting. Do not translate ICU keywords, selectors, or placeholder names.\n\n")
	b.WriteString("Target language: ")
	b.WriteString(strings.TrimSpace(req.TargetLanguage))
	b.WriteString("\n")

	b.WriteString("Source text:\n")
	b.WriteString(req.Source)
	return b.String()
}

func appendRuntimeContextToSystemPrompt(baseSystemPrompt, runtimeContext string) string {
	base := strings.TrimSpace(baseSystemPrompt)
	contextBlock := strings.TrimSpace(runtimeContext)
	if contextBlock == "" {
		return base
	}

	const header = "Runtime translation context (do not translate or repeat):"
	if base == "" {
		return header + "\n" + contextBlock
	}
	return base + "\n\n" + header + "\n" + contextBlock
}

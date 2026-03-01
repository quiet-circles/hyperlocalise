package translationfileparser

import (
	"strings"
	"testing"
)

func TestMarkdownParserParseKeepsFrontmatterAndCodeFencesOut(t *testing.T) {
	content := []byte("---\ntitle: Hello\n---\n\n# Heading\n\nParagraph with [link text](https://example.com).\n\n```go\nfmt.Println(\"hi\")\n```\n")

	got, err := (MarkdownParser{}).Parse(content)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if len(got) == 0 {
		t.Fatalf("expected extracted entries")
	}

	for _, value := range got {
		if strings.Contains(value, "title:") || strings.Contains(value, "fmt.Println") {
			t.Fatalf("unexpected extracted non-translatable value: %q", value)
		}
	}
}

func TestMarkdownParserParseMdxKeepsComponentsAndAttributesOut(t *testing.T) {
	template := []byte("---\ntitle: Mdx Page\n---\n\nimport Tabs from '@theme/Tabs'\n\n<Tabs defaultValue=\"first\">\n  <Tab value=\"first\" label=\"First tab\">\n    Here is a paragraph with <Badge text=\"New\" /> and [docs](https://mintlify.com).\n  </Tab>\n</Tabs>\n\n<CodeGroup>\n```bash\necho \"hi\"\n```\n</CodeGroup>\n\n> <Note icon=\"info\">Read me</Note>\n")

	entries, err := (MarkdownParser{}).Parse(template)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	combined := strings.Join(mapValues(entries), "\n")
	if strings.Contains(combined, "defaultValue") || strings.Contains(combined, "label=") || strings.Contains(combined, "import Tabs") {
		t.Fatalf("expected JSX/import literals to be excluded, got %q", combined)
	}
	if strings.Contains(combined, "echo \"hi\"") {
		t.Fatalf("expected fenced code to be excluded, got %q", combined)
	}
	if !strings.Contains(combined, "Here is a paragraph") || !strings.Contains(combined, "Read me") {
		t.Fatalf("expected prose to be extracted, got %q", combined)
	}
}

func TestMarshalMarkdownRoundTripReplacesOnlyExtractedSegments(t *testing.T) {
	template := []byte("# Heading\n\n- First item\n- Second item\n\nSee [docs](https://example.com).\n> Quote\n| Name | Value |\n| ---- | ----- |\n| Alpha | Beta |\n")

	entries, err := (MarkdownParser{}).Parse(template)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	updates := map[string]string{}
	for k, v := range entries {
		updates[k] = strings.ToUpper(v)
	}

	output := string(MarshalMarkdown(template, updates))
	if !strings.Contains(output, "https://example.com") {
		t.Fatalf("expected link destination preserved, got %q", output)
	}
	if strings.Contains(output, "First item") {
		t.Fatalf("expected translated text output, got %q", output)
	}
	if !strings.Contains(output, "```") && strings.Contains(string(template), "```") {
		t.Fatalf("expected markdown structure to be preserved")
	}
}

func TestMarshalMarkdownPreservesLinkDestinationsWithParentheses(t *testing.T) {
	template := []byte("See [URL docs](https://en.wikipedia.org/wiki/URL_(disambiguation)) now.\n")

	entries, err := (MarkdownParser{}).Parse(template)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	updates := map[string]string{}
	for k, v := range entries {
		updates[k] = strings.ToUpper(v)
	}

	output := string(MarshalMarkdown(template, updates))
	if !strings.Contains(output, "https://en.wikipedia.org/wiki/URL_(disambiguation)") {
		t.Fatalf("expected full link destination preserved, got %q", output)
	}
	if !strings.Contains(output, "[URL DOCS]") {
		t.Fatalf("expected link text translated, got %q", output)
	}
}

func TestMarshalMarkdownMdxRoundTripPreservesComponentSyntax(t *testing.T) {
	template := []byte("<Tabs defaultValue=\"cli\">\n<Tab value=\"cli\" label=\"CLI\">Run the command.</Tab>\n</Tabs>\n")
	entries, err := (MarkdownParser{}).Parse(template)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	updates := map[string]string{}
	for k, v := range entries {
		updates[k] = "FR(" + strings.TrimSpace(v) + ")"
	}

	output := string(MarshalMarkdown(template, updates))
	if !strings.Contains(output, "defaultValue=\"cli\"") || !strings.Contains(output, "label=\"CLI\"") {
		t.Fatalf("expected component attributes unchanged, got %q", output)
	}
	if !strings.Contains(output, "FR(Run the command.)") {
		t.Fatalf("expected prose translation in component body, got %q", output)
	}
}

func TestStrategyParsesMarkdown(t *testing.T) {
	s := NewDefaultStrategy()
	content := []byte("# Welcome\n\nHello world\n")

	got, err := s.Parse("fr.md", content)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(got) < 2 {
		t.Fatalf("expected markdown entries parsed, got %d", len(got))
	}
}

func TestStrategyParsesMDX(t *testing.T) {
	s := NewDefaultStrategy()
	content := []byte("<Callout>Use the docs.</Callout>\n")

	got, err := s.Parse("fr.mdx", content)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(got) == 0 {
		t.Fatalf("expected mdx entries parsed")
	}
}

func mapValues(values map[string]string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, value)
	}
	return out
}

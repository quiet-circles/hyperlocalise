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

func TestMarkdownParserParseMdxIncludesFrontmatterMetadataValues(t *testing.T) {
	content := []byte("---\ntitle: \"Conflict handling\"\ndescription: \"Understand pull/push conflicts and apply safe resolution strategies.\"\n---\n\nBody text.\n")

	got, err := (MarkdownParser{}).Parse(content)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	combined := strings.Join(mapValues(got), "\n")
	if !strings.Contains(combined, "Conflict handling") {
		t.Fatalf("expected title frontmatter value extracted, got %q", combined)
	}
	if !strings.Contains(combined, "Understand pull/push conflicts and apply safe resolution strategies.") {
		t.Fatalf("expected description frontmatter value extracted, got %q", combined)
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

func TestMarshalMarkdownPreservesBoundaryWhitespaceWithTrimmedTranslations(t *testing.T) {
	template := []byte("  Hello  \n\n- World \n")
	entries, err := (MarkdownParser{}).Parse(template)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	updates := map[string]string{}
	for k, v := range entries {
		updates[k] = strings.TrimSpace(strings.ToUpper(v))
	}

	output := string(MarshalMarkdown(template, updates))
	if output != "  HELLO  \n\n- WORLD \n" {
		t.Fatalf("expected boundary whitespace preserved, got %q", output)
	}
}

func TestMarshalMarkdownPreservesFrontmatterStructureWithTranslatedMetadataValues(t *testing.T) {
	template := []byte("---\ntitle: \"Conflict handling\"\ndescription: \"Understand pull/push conflicts and apply safe resolution strategies.\"\n---\n")
	entries, err := (MarkdownParser{}).Parse(template)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	updates := map[string]string{}
	for k, v := range entries {
		switch v {
		case "Conflict handling":
			updates[k] = "Gestion des conflits"
		case "Understand pull/push conflicts and apply safe resolution strategies.":
			updates[k] = "Comprendre les conflits pull/push et appliquer des strategies de resolution sures."
		default:
			updates[k] = v
		}
	}

	output := string(MarshalMarkdown(template, updates))
	if !strings.Contains(output, "title: \"Gestion des conflits\"") {
		t.Fatalf("expected translated title in frontmatter, got %q", output)
	}
	if !strings.Contains(output, "description: \"Comprendre les conflits pull/push et appliquer des strategies de resolution sures.\"") {
		t.Fatalf("expected translated description in frontmatter, got %q", output)
	}
	if !strings.HasPrefix(output, "---\n") || !strings.HasSuffix(output, "---\n") {
		t.Fatalf("expected frontmatter delimiters preserved, got %q", output)
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

func TestMarkdownParserKeysAreStableAcrossInsertedSegments(t *testing.T) {
	base := []byte("Alpha\nBeta\n")
	withInsert := []byte("Intro\nAlpha\nBeta\n")

	baseEntries, err := (MarkdownParser{}).Parse(base)
	if err != nil {
		t.Fatalf("parse base: %v", err)
	}
	insertedEntries, err := (MarkdownParser{}).Parse(withInsert)
	if err != nil {
		t.Fatalf("parse with insert: %v", err)
	}

	baseAlpha := findKeyByValue(baseEntries, "Alpha")
	baseBeta := findKeyByValue(baseEntries, "Beta")
	insertAlpha := findKeyByValue(insertedEntries, "Alpha")
	insertBeta := findKeyByValue(insertedEntries, "Beta")
	if baseAlpha == "" || baseBeta == "" || insertAlpha == "" || insertBeta == "" {
		t.Fatalf("expected keys for shared segments")
	}
	if baseAlpha != insertAlpha || baseBeta != insertBeta {
		t.Fatalf("expected stable hash keys, base=(%s,%s) inserted=(%s,%s)", baseAlpha, baseBeta, insertAlpha, insertBeta)
	}
}

func TestMarkdownParserKeysDisambiguateDuplicateSegments(t *testing.T) {
	content := []byte("Repeat\nRepeat\n")
	entries, err := (MarkdownParser{}).Parse(content)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected two distinct keys for duplicate segments, got %d", len(entries))
	}
}

func TestMarshalMarkdownWithTargetFallbackIncludesInsertedSourceSegments(t *testing.T) {
	source := []byte("# Guide\n\nExisting intro.\n\nNew section added.\n\nExisting outro.\n")
	target := []byte("# Guide\n\nIntro existant.\n\nConclusion existante.\n")

	sourceEntries, err := (MarkdownParser{}).Parse(source)
	if err != nil {
		t.Fatalf("parse source: %v", err)
	}

	var newKey string
	for key, value := range sourceEntries {
		if strings.TrimSpace(value) == "New section added." {
			newKey = key
			break
		}
	}
	if newKey == "" {
		t.Fatalf("expected key for inserted source segment")
	}

	output := string(MarshalMarkdownWithTargetFallback(source, target, map[string]string{
		newKey: "Nouvelle section ajoutee.",
	}))

	if !strings.Contains(output, "Intro existant.") {
		t.Fatalf("expected existing translated intro preserved, got %q", output)
	}
	if !strings.Contains(output, "Nouvelle section ajoutee.") {
		t.Fatalf("expected new inserted section translated, got %q", output)
	}
	if !strings.Contains(output, "Conclusion existante.") {
		t.Fatalf("expected existing translated outro preserved, got %q", output)
	}
}

func TestMarshalMarkdownWithTargetFallbackPreservesInlineCodeSentenceOrder(t *testing.T) {
	source := []byte("- MDX `import` and `export` lines\n- Next line\n")
	target := []byte("- MDX `import` va `export` dong\n- Dong tiep theo\n")
	output := string(MarshalMarkdownWithTargetFallback(source, target, map[string]string{}))

	if !strings.Contains(output, "- MDX `import` va `export` dong") {
		t.Fatalf("expected inline-code sentence order preserved, got %q", output)
	}
	if strings.Contains(output, "MDX `import` MDX `export`") {
		t.Fatalf("unexpected duplicated/reordered inline-code sentence, got %q", output)
	}
}

func TestMarshalMarkdownWithTargetFallbackDoesNotBleedAcrossHeadings(t *testing.T) {
	source := []byte("When you pass `--output`, report includes metadata.\n\n## Worker tuning guidance\n\nLower `--workers` in constrained CI.\n")
	target := []byte("Khi truyen `--output`, bao cao gom metadata.\n\n## Huong dan dieu chinh cong nhan\n\nGiam `--workers` trong CI gioi han.\n")

	out := string(MarshalMarkdownWithTargetFallback(source, target, map[string]string{}))
	if !strings.Contains(out, "## Huong dan dieu chinh cong nhan") {
		t.Fatalf("expected heading translation preserved, got %q", out)
	}
	if strings.Contains(out, "`--workers` Huong dan dieu chinh cong nhan") {
		t.Fatalf("unexpected heading text merged into sentence, got %q", out)
	}
}

func TestAlignMarkdownTargetToSourceMapsBySourceKeys(t *testing.T) {
	source := []byte("# Guide\n\nExisting intro.\n\nExisting outro.\n")
	target := []byte("# Guide\n\nIntro existant.\n\nConclusion existante.\n")

	aligned := AlignMarkdownTargetToSource(source, target)
	sourceEntries, err := (MarkdownParser{}).Parse(source)
	if err != nil {
		t.Fatalf("parse source: %v", err)
	}

	introKey := findKeyByValue(sourceEntries, "Existing intro.")
	outroKey := findKeyByValue(sourceEntries, "Existing outro.")
	if introKey == "" || outroKey == "" {
		t.Fatalf("expected source keys for intro/outro")
	}

	if got := strings.TrimSpace(aligned[introKey]); got != "Intro existant." {
		t.Fatalf("expected intro mapped to source key, got %q", got)
	}
	if got := strings.TrimSpace(aligned[outroKey]); got != "Conclusion existante." {
		t.Fatalf("expected outro mapped to source key, got %q", got)
	}
}

func TestAlignMarkdownTargetToSourceMapsInsertedSectionWhenPresent(t *testing.T) {
	source := []byte("# Guide\n\nExisting intro.\n\nNew section added.\n\nExisting outro.\n")
	target := []byte("# Guide\n\nIntro existant.\n\nNouvelle section ajoutee.\n\nConclusion existante.\n")

	aligned := AlignMarkdownTargetToSource(source, target)
	sourceEntries, err := (MarkdownParser{}).Parse(source)
	if err != nil {
		t.Fatalf("parse source: %v", err)
	}

	newKey := findKeyByValue(sourceEntries, "New section added.")
	if newKey == "" {
		t.Fatalf("expected source key for inserted section")
	}
	if got := strings.TrimSpace(aligned[newKey]); got != "Nouvelle section ajoutee." {
		t.Fatalf("expected inserted section mapped to source key, got %q", got)
	}
}

func mapValues(values map[string]string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, value)
	}
	return out
}

func findKeyByValue(values map[string]string, needle string) string {
	for key, value := range values {
		if strings.TrimSpace(value) == needle {
			return key
		}
	}
	return ""
}

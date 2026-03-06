package translationfileparser

import (
	"strings"
	"testing"
)

func TestXLIFFParserPreservesInlineMarkupInTarget(t *testing.T) {
	content := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<xliff version="1.2">
  <file source-language="en-US" target-language="fr">
    <body>
      <trans-unit id="hello">
        <source>Hello <ph id="1"/> world</source>
        <target>Bonjour <ph id="1"/> monde</target>
      </trans-unit>
    </body>
  </file>
</xliff>`)

	got, err := (XLIFFParser{}).Parse(content)
	if err != nil {
		t.Fatalf("parse xliff: %v", err)
	}
	if !strings.Contains(got["hello"], "Bonjour ") || !strings.Contains(got["hello"], `<ph id="1"></ph>`) || !strings.Contains(got["hello"], " monde") {
		t.Fatalf("expected inline target markup preserved, got %#v", got["hello"])
	}
}

func TestMarshalXLIFFReplacesTargetWhenPresent(t *testing.T) {
	template := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<xliff version="1.2">
  <file source-language="en-US" target-language="fr">
    <body>
      <trans-unit id="hello">
        <source>Hello</source>
        <target state="translated">Hello</target>
      </trans-unit>
    </body>
  </file>
</xliff>`)

	out, err := MarshalXLIFF(template, map[string]string{"hello": "Bonjour"}, "fr-FR")
	if err != nil {
		t.Fatalf("marshal xliff: %v", err)
	}

	content := string(out)
	if !strings.Contains(content, `source-language="en-US"`) {
		t.Fatalf("expected file source-language preserved, got %q", content)
	}
	if !strings.Contains(content, `target-language="fr-FR"`) {
		t.Fatalf("expected file target-language to match target locale, got %q", content)
	}
	if !strings.Contains(content, ">Hello</source>") {
		t.Fatalf("expected source text preserved when target exists, got %q", content)
	}
	if !strings.Contains(content, `<target state="translated">Bonjour</target>`) {
		t.Fatalf("expected target metadata preserved, got %q", content)
	}
}

func TestMarshalXLIFFPreservesInlineMarkupWhenReplacingTarget(t *testing.T) {
	template := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<xliff version="1.2">
  <file source-language="en-US" target-language="fr">
    <body>
      <trans-unit id="hello">
        <source>Hello <ph id="1"/> world</source>
        <target state="translated">Hello <ph id="1"/> world</target>
      </trans-unit>
    </body>
  </file>
</xliff>`)

	out, err := MarshalXLIFF(template, map[string]string{"hello": `Bonjour <ph id="1"></ph> monde`}, "fr-FR")
	if err != nil {
		t.Fatalf("marshal xliff: %v", err)
	}

	content := string(out)
	if !strings.Contains(content, `>Hello <ph id="1"></ph> world</source>`) {
		t.Fatalf("expected source inline markup preserved, got %q", content)
	}
	if !strings.Contains(content, `<target state="translated">Bonjour <ph id="1"></ph> monde</target>`) {
		t.Fatalf("expected target inline markup replaced without flattening, got %q", content)
	}
}

func TestMarshalXLIFF20PreservesSrcLangAndRewritesTrgLang(t *testing.T) {
	template := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<xliff version="2.0" srcLang="en-US" trgLang="de-DE" xmlns="urn:oasis:names:tc:xliff:document:2.0">
  <file id="f1">
    <unit id="hello">
      <segment>
        <source>Hello</source>
        <target state="initial">Hello</target>
      </segment>
    </unit>
  </file>
</xliff>`)

	out, err := MarshalXLIFF(template, map[string]string{"hello": "Bonjour"}, "fr-FR")
	if err != nil {
		t.Fatalf("marshal xliff: %v", err)
	}

	content := string(out)
	if !strings.Contains(content, `srcLang="en-US"`) {
		t.Fatalf("expected srcLang preserved, got %q", content)
	}
	if !strings.Contains(content, `trgLang="fr-FR"`) {
		t.Fatalf("expected trgLang rewritten to target locale, got %q", content)
	}
	if !strings.Contains(content, ">Hello</source>") {
		t.Fatalf("expected source text preserved for xliff 2.x when target exists, got %q", content)
	}
	if !strings.Contains(content, `state="initial">Bonjour</target>`) {
		t.Fatalf("expected target text updated without losing metadata, got %q", content)
	}
}

func TestMarshalXLIFFAddsMissingTargetLocaleAttrs(t *testing.T) {
	xliff12 := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<xliff version="1.2">
  <file source-language="en-US">
    <body>
      <trans-unit id="hello"><source>Hello</source></trans-unit>
    </body>
  </file>
</xliff>`)

	out12, err := MarshalXLIFF(xliff12, map[string]string{"hello": "Bonjour"}, "fr-FR")
	if err != nil {
		t.Fatalf("marshal xliff 1.2: %v", err)
	}
	if !strings.Contains(string(out12), `target-language="fr-FR"`) {
		t.Fatalf("expected target-language to be added when missing, got %q", string(out12))
	}

	xliff20 := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<xliff version="2.0" srcLang="en-US" xmlns="urn:oasis:names:tc:xliff:document:2.0">
  <file id="f1">
    <unit id="hello">
      <segment><source>Hello</source></segment>
    </unit>
  </file>
</xliff>`)

	out20, err := MarshalXLIFF(xliff20, map[string]string{"hello": "Bonjour"}, "fr-FR")
	if err != nil {
		t.Fatalf("marshal xliff 2.0: %v", err)
	}
	if !strings.Contains(string(out20), `trgLang="fr-FR"`) {
		t.Fatalf("expected trgLang to be added when missing, got %q", string(out20))
	}
}

func TestMarshalXLIFFReplacesSourceWhenTargetMissing(t *testing.T) {
	template := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<xliff version="1.2">
  <file source-language="en-US">
    <body>
      <trans-unit id="hello">
        <source>Hello</source>
      </trans-unit>
    </body>
  </file>
</xliff>`)

	out, err := MarshalXLIFF(template, map[string]string{"hello": "Bonjour"}, "fr-FR")
	if err != nil {
		t.Fatalf("marshal xliff: %v", err)
	}

	content := string(out)
	if !strings.Contains(content, "<source>Bonjour</source>") {
		t.Fatalf("expected source text to be replaced when target is missing, got %q", content)
	}
}

func TestMarshalXLIFFReplacesSourceWithInlineMarkupWhenTargetMissing(t *testing.T) {
	template := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<xliff version="1.2">
  <file source-language="en-US">
    <body>
      <trans-unit id="hello">
        <source>Hello <g id="1">world</g></source>
      </trans-unit>
    </body>
  </file>
</xliff>`)

	out, err := MarshalXLIFF(template, map[string]string{"hello": `Bonjour <g id="1">monde</g>`}, "fr-FR")
	if err != nil {
		t.Fatalf("marshal xliff: %v", err)
	}

	content := string(out)
	if !strings.Contains(content, `<source>Bonjour <g id="1">monde</g></source>`) {
		t.Fatalf("expected source inline markup replaced when target missing, got %q", content)
	}
}

func TestMarshalXLIFFEscapesPlainTextReplacementWhenFragmentInvalid(t *testing.T) {
	template := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<xliff version="1.2">
  <file source-language="en-US" target-language="fr">
    <body>
      <trans-unit id="math">
        <source>Math</source>
        <target>Math</target>
      </trans-unit>
    </body>
  </file>
</xliff>`)

	out, err := MarshalXLIFF(template, map[string]string{"math": "2 < 3 & 4"}, "fr-FR")
	if err != nil {
		t.Fatalf("marshal xliff: %v", err)
	}

	content := string(out)
	if !strings.Contains(content, `<target>2 &lt; 3 &amp; 4</target>`) {
		t.Fatalf("expected invalid xml fragment replacement to be escaped as text, got %q", content)
	}
}

func TestMarshalXLIFFClearsTargetWhenReplacementEmpty(t *testing.T) {
	template := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<xliff version="1.2">
  <file source-language="en-US" target-language="fr">
    <body>
      <trans-unit id="hello">
        <source>Hello</source>
        <target>Bonjour</target>
      </trans-unit>
    </body>
  </file>
</xliff>`)

	out, err := MarshalXLIFF(template, map[string]string{"hello": ""}, "fr-FR")
	if err != nil {
		t.Fatalf("marshal xliff: %v", err)
	}

	content := string(out)
	if !strings.Contains(content, "<target></target>") {
		t.Fatalf("expected empty replacement to clear target content, got %q", content)
	}
	if !strings.Contains(content, "<source>Hello</source>") {
		t.Fatalf("expected source preserved when clearing target, got %q", content)
	}
}

func TestXLIFFParserPreservesWhitespaceOnlyValues(t *testing.T) {
	content := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<xliff version="1.2">
  <file source-language="en-US">
    <body>
      <trans-unit id="space"><source>   </source></trans-unit>
    </body>
  </file>
</xliff>`)

	got, err := (XLIFFParser{}).Parse(content)
	if err != nil {
		t.Fatalf("parse xliff: %v", err)
	}
	if got["space"] != "   " {
		t.Fatalf("expected whitespace-only value preserved, got %#v", got["space"])
	}
}

func TestXLIFFParserConcatenatesMultipleSegmentsInOrder(t *testing.T) {
	content := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<xliff version="2.0" srcLang="en-US" trgLang="fr-FR" xmlns="urn:oasis:names:tc:xliff:document:2.0">
  <file id="f1">
    <unit id="hello">
      <segment>
        <source>Hello </source>
        <target>Bonjour </target>
      </segment>
      <segment>
        <source>world</source>
        <target>monde</target>
      </segment>
    </unit>
  </file>
</xliff>`)

	got, err := (XLIFFParser{}).Parse(content)
	if err != nil {
		t.Fatalf("parse xliff: %v", err)
	}
	if got["hello"] != "Bonjour monde" {
		t.Fatalf("expected multi-segment target content concatenated in order, got %#v", got["hello"])
	}
}

func TestXLIFFParserDuplicateUnitIDCollidesByID(t *testing.T) {
	content := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<xliff version="1.2">
  <file source-language="en-US" target-language="fr">
    <body>
      <trans-unit id="dup">
        <source>First source</source>
        <target>Premier</target>
      </trans-unit>
      <trans-unit id="dup">
        <source>Second source</source>
        <target>Deuxieme</target>
      </trans-unit>
    </body>
  </file>
</xliff>`)

	got, err := (XLIFFParser{}).Parse(content)
	if err != nil {
		t.Fatalf("parse xliff: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected duplicate ids to collapse to one key, got %+v", got)
	}
	if got["dup"] != "Deuxieme" {
		t.Fatalf("expected last duplicate id to win, got %+v", got)
	}
}

func TestXLIFFParserFallsBackToSourceWhenTargetIsEmpty(t *testing.T) {
	content := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<xliff version="1.2">
  <file source-language="en-US" target-language="fr">
    <body>
      <trans-unit id="hello">
        <source>Hello</source>
        <target></target>
      </trans-unit>
    </body>
  </file>
</xliff>`)

	got, err := (XLIFFParser{}).Parse(content)
	if err != nil {
		t.Fatalf("parse xliff: %v", err)
	}
	if got["hello"] != "Hello" {
		t.Fatalf("expected empty target to fall back to source, got %#v", got["hello"])
	}
}

func TestXLIFFParserFallsBackToSourceWhenTargetIsWhitespaceOnly(t *testing.T) {
	content := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<xliff version="1.2">
  <file source-language="en-US" target-language="fr">
    <body>
      <trans-unit id="hello">
        <source>Hello</source>
        <target>   </target>
      </trans-unit>
    </body>
  </file>
</xliff>`)

	got, err := (XLIFFParser{}).Parse(content)
	if err != nil {
		t.Fatalf("parse xliff: %v", err)
	}
	if got["hello"] != "Hello" {
		t.Fatalf("expected whitespace-only target to fall back to source, got %#v", got["hello"])
	}
}

func TestMarshalXLIFFSupportsNameAndResnameFallbackKeys(t *testing.T) {
	template := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<xliff version="1.2">
  <file source-language="en-US" target-language="fr">
    <body>
      <trans-unit name="welcome">
        <source>Welcome</source>
        <target>Welcome</target>
      </trans-unit>
      <trans-unit resname="goodbye">
        <source>Goodbye</source>
        <target>Goodbye</target>
      </trans-unit>
    </body>
  </file>
</xliff>`)

	out, err := MarshalXLIFF(template, map[string]string{
		"welcome": "Bienvenue",
		"goodbye": "Au revoir",
	}, "fr-FR")
	if err != nil {
		t.Fatalf("marshal xliff: %v", err)
	}

	content := string(out)
	if !strings.Contains(content, `<trans-unit name="welcome">`) || !strings.Contains(content, `<target>Bienvenue</target>`) {
		t.Fatalf("expected name key replacement, got %q", content)
	}
	if !strings.Contains(content, `<trans-unit resname="goodbye">`) || !strings.Contains(content, `<target>Au revoir</target>`) {
		t.Fatalf("expected resname key replacement, got %q", content)
	}
}

func TestXLIFFParserPreservesCommentAndProcInstInsideTarget(t *testing.T) {
	content := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<xliff version="1.2">
  <file source-language="en-US" target-language="fr">
    <body>
      <trans-unit id="hello">
        <source>Hello</source>
        <target>Bonjour<!-- tone --><?review keep?></target>
      </trans-unit>
    </body>
  </file>
</xliff>`)

	got, err := (XLIFFParser{}).Parse(content)
	if err != nil {
		t.Fatalf("parse xliff: %v", err)
	}
	if !strings.Contains(got["hello"], "Bonjour") || !strings.Contains(got["hello"], "<!-- tone -->") || !strings.Contains(got["hello"], "<?review keep?>") {
		t.Fatalf("expected comment and proc-inst preserved in target, got %#v", got["hello"])
	}
}

func TestMarshalXLIFFPreservesInlineMarkupWithNamespaceReplacement(t *testing.T) {
	template := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<xliff version="1.2">
  <file source-language="en-US" target-language="fr">
    <body>
      <trans-unit id="hello">
        <source>Hello</source>
        <target>Hello</target>
      </trans-unit>
    </body>
  </file>
</xliff>`)

	out, err := MarshalXLIFF(template, map[string]string{
		"hello": `Bonjour <ph xmlns="urn:test:inline" id="1"></ph> monde`,
	}, "fr-FR")
	if err != nil {
		t.Fatalf("marshal xliff: %v", err)
	}

	content := string(out)
	if !strings.Contains(content, `<target>Bonjour `) || !strings.Contains(content, `<ph xmlns="urn:test:inline"`) || !strings.Contains(content, `id="1"></ph> monde</target>`) {
		t.Fatalf("expected namespace-qualified inline replacement preserved, got %q", content)
	}
}

package translationfileparser

import (
	"strings"
	"testing"
)

func TestMarshalXLIFFReplacesTargetWhenPresent(t *testing.T) {
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
	if !strings.Contains(content, "<target>Bonjour</target>") {
		t.Fatalf("expected target text to be replaced, got %q", content)
	}
	if !strings.Contains(content, "<source>Bonjour</source>") {
		t.Fatalf("expected unit text to be updated, got %q", content)
	}
}

func TestMarshalXLIFF20PreservesSrcLangAndRewritesTrgLang(t *testing.T) {
	template := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<xliff version="2.0" srcLang="en-US" trgLang="de-DE" xmlns="urn:oasis:names:tc:xliff:document:2.0">
  <file id="f1">
    <unit id="hello">
      <segment>
        <source>Hello</source>
        <target>Hello</target>
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

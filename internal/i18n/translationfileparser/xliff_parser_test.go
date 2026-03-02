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
	if !strings.Contains(content, `source-language="fr-FR"`) {
		t.Fatalf("expected file source-language to match target locale, got %q", content)
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

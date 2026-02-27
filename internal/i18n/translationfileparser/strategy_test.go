package translationfileparser

import (
	"strings"
	"testing"
)

func TestStrategyParsesJSON(t *testing.T) {
	s := NewDefaultStrategy()

	got, err := s.Parse("fr.json", []byte(`{"hello":"bonjour","home":{"title":"Accueil"}}`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if got["hello"] != "bonjour" {
		t.Fatalf("unexpected hello translation: %q", got["hello"])
	}
	if got["home.title"] != "Accueil" {
		t.Fatalf("unexpected home.title translation: %q", got["home.title"])
	}
}

func TestStrategyParsesXLIFF12(t *testing.T) {
	s := NewDefaultStrategy()

	content := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<xliff version="1.2">
  <file source-language="en" target-language="fr" datatype="plaintext" original="messages">
    <body>
      <trans-unit id="hello">
        <source>Hello</source>
        <target>Bonjour</target>
      </trans-unit>
      <trans-unit id="welcome">
        <source>Welcome</source>
      </trans-unit>
    </body>
  </file>
</xliff>`)

	got, err := s.Parse("fr.xlf", content)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if got["hello"] != "Bonjour" {
		t.Fatalf("unexpected hello translation: %q", got["hello"])
	}
	if got["welcome"] != "Welcome" {
		t.Fatalf("unexpected welcome translation fallback: %q", got["welcome"])
	}
}

func TestStrategyParsesXLIFF2(t *testing.T) {
	s := NewDefaultStrategy()

	content := []byte(`<?xml version="1.0" encoding="UTF-8"?>
<xliff version="2.0" srcLang="en" trgLang="fr" xmlns="urn:oasis:names:tc:xliff:document:2.0">
  <file id="f1">
    <unit id="checkout.submit">
      <segment>
        <source>Submit</source>
        <target>Valider</target>
      </segment>
    </unit>
  </file>
</xliff>`)

	got, err := s.Parse("fr.xliff", content)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if got["checkout.submit"] != "Valider" {
		t.Fatalf("unexpected translation: %q", got["checkout.submit"])
	}
}

func TestStrategyUnsupportedExtension(t *testing.T) {
	s := NewDefaultStrategy()

	_, err := s.Parse("fr.yaml", []byte(""))
	if err == nil {
		t.Fatalf("expected unsupported extension error")
	}
	if !strings.Contains(err.Error(), "unsupported file extension") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStrategyParsesPO(t *testing.T) {
	s := NewDefaultStrategy()

	content := []byte(`msgid ""
msgstr ""
"Project-Id-Version: test\\n"

msgid "hello"
msgstr "bonjour"

msgid "home.title"
msgstr ""
"Accueil "
"Maison"

msgid "items"
msgid_plural "items"
msgstr[0] "article"
msgstr[1] "articles"
`)

	got, err := s.Parse("fr.po", content)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if got["hello"] != "bonjour" {
		t.Fatalf("unexpected hello translation: %q", got["hello"])
	}
	if got["home.title"] != "Accueil Maison" {
		t.Fatalf("unexpected home.title translation: %q", got["home.title"])
	}
	if got["items"] != "article" {
		t.Fatalf("unexpected plural base translation: %q", got["items"])
	}
	if len(got) != 3 {
		t.Fatalf("unexpected parsed entry count: got %d want 3", len(got))
	}
}

func TestStrategyParsesPOInvalidInputReturnsError(t *testing.T) {
	s := NewDefaultStrategy()

	content := []byte(`msgid hello
msgstr "bonjour"
`)

	_, err := s.Parse("fr.po", content)
	if err == nil {
		t.Fatalf("expected parse error for malformed po input")
	}
	if !strings.Contains(err.Error(), "parse msgid") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestJSONParserRejectsInvalidShape(t *testing.T) {
	_, err := (JSONParser{}).Parse([]byte(`{"count":1}`))
	if err == nil {
		t.Fatalf("expected invalid json translation error")
	}
}

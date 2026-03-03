package translationfileparser

import "testing"

func TestJSONParserParsesFormatJSDefaultMessageOnly(t *testing.T) {
	content := []byte(`{
  "checkout.submit": {
    "defaultMessage": "Submit order",
    "description": "Checkout submit button"
  },
  "home.title": {
    "defaultMessage": "Welcome",
    "description": "Home page title"
  }
}`)

	got, err := (JSONParser{}).Parse(content)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if got["checkout.submit"] != "Submit order" {
		t.Fatalf("unexpected checkout.submit translation: %q", got["checkout.submit"])
	}
	if got["home.title"] != "Welcome" {
		t.Fatalf("unexpected home.title translation: %q", got["home.title"])
	}
	if _, ok := got["checkout.submit.description"]; ok {
		t.Fatalf("description must not be parsed as translatable entry")
	}
}

func TestJSONParserRejectsFormatJSDefaultMessageNonString(t *testing.T) {
	_, err := (JSONParser{}).Parse([]byte(`{
  "checkout.submit": {
    "defaultMessage": 123,
    "description": "Checkout submit button"
  }
}`))
	if err == nil {
		t.Fatalf("expected invalid FormatJS defaultMessage error")
	}
}

func TestJSONParserMixedShapeFallsBackToStandardJSONFlattening(t *testing.T) {
	content := []byte(`{
  "meta": {
    "defaultMessage": "Do not treat as FormatJS root",
    "note": "still nested data"
  },
  "home": {
    "title": "Welcome"
  }
}`)

	got, err := (JSONParser{}).Parse(content)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if got["meta.defaultMessage"] != "Do not treat as FormatJS root" {
		t.Fatalf("unexpected meta.defaultMessage: %q", got["meta.defaultMessage"])
	}
	if got["meta.note"] != "still nested data" {
		t.Fatalf("unexpected meta.note: %q", got["meta.note"])
	}
	if got["home.title"] != "Welcome" {
		t.Fatalf("unexpected home.title: %q", got["home.title"])
	}
	if _, ok := got["meta"]; ok {
		t.Fatalf("unexpected top-level meta key in flattened output")
	}
}

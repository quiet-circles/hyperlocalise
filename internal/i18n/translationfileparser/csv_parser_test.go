package translationfileparser

import (
	"strings"
	"testing"
)

func TestCSVParserParsesKeyValueLayout(t *testing.T) {
	p := CSVParser{}
	got, err := p.Parse([]byte("key,value\nhello,Bonjour\n"))
	if err != nil {
		t.Fatalf("parse csv: %v", err)
	}
	if got["hello"] != "Bonjour" {
		t.Fatalf("unexpected value: %q", got["hello"])
	}
}

func TestCSVParserParsesPerLocaleColumnLayout(t *testing.T) {
	p := CSVParser{KeyColumn: "id", ValueColumn: "fr"}
	got, err := p.Parse([]byte("id,en,fr\nhello,Hello,Bonjour\n"))
	if err != nil {
		t.Fatalf("parse csv: %v", err)
	}
	if got["hello"] != "Bonjour" {
		t.Fatalf("unexpected locale column value: %q", got["hello"])
	}
}

func TestCSVParserDelimiterQuoteAndEscaping(t *testing.T) {
	p := CSVParser{Delimiter: ';'}
	got, err := p.Parse([]byte("key;value\nwelcome;\"He said \"\"Bonjour\"\"\\nAgain\"\n"))
	if err != nil {
		t.Fatalf("parse csv: %v", err)
	}
	if got["welcome"] != "He said \"Bonjour\"\\nAgain" {
		t.Fatalf("unexpected escaped value: %q", got["welcome"])
	}
}

func TestMarshalCSVPreservesColumnsAndAppendsDeterministically(t *testing.T) {
	template := []byte("key,en,fr\nhello,Hello,Salut\n")
	out, err := MarshalCSV(template, map[string]string{"hello": "Bonjour", "bye": "Au revoir"}, CSVParser{ValueColumn: "fr"})
	if err != nil {
		t.Fatalf("marshal csv: %v", err)
	}
	text := string(out)
	if !strings.Contains(text, "hello,Hello,Bonjour") {
		t.Fatalf("expected existing row update, got %q", text)
	}
	if !strings.Contains(text, "bye,,Au revoir") {
		t.Fatalf("expected deterministic append for new key, got %q", text)
	}
}

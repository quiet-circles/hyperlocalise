package translationfileparser

import (
	"strings"
	"testing"
)

func TestMarshalPOFileReplacesMsgstrByMsgid(t *testing.T) {
	template := []byte(`msgid ""
msgstr ""
"Language: en-US\n"

msgid "hello"
msgstr "Hello"

msgid "items"
msgid_plural "items"
msgstr[0] "item"
msgstr[1] "items"
`)

	out, err := MarshalPOFile(template, map[string]string{
		"hello": "Bonjour",
		"items": "article",
	})
	if err != nil {
		t.Fatalf("marshal po: %v", err)
	}

	content := string(out)
	if !strings.Contains(content, `msgstr "Bonjour"`) {
		t.Fatalf("expected msgstr replacement for hello, got %q", content)
	}
	if !strings.Contains(content, `msgstr[0] "article"`) {
		t.Fatalf("expected msgstr[0] replacement for items, got %q", content)
	}
	if !strings.Contains(content, `msgstr[1] "items"`) {
		t.Fatalf("expected higher plural forms unchanged, got %q", content)
	}
}

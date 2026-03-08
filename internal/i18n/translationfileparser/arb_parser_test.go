package translationfileparser

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestARBParserSkipsMetadataKeys(t *testing.T) {
	content := []byte(`{
  "@@locale": "en",
  "hello": "Hello",
  "@hello": {
    "description": "Greeting"
  }
}`)

	got, err := (ARBParser{}).Parse(content)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if got["hello"] != "Hello" {
		t.Fatalf("unexpected hello translation: %q", got["hello"])
	}
	if _, ok := got["@hello"]; ok {
		t.Fatalf("metadata key @hello must not be parsed as translatable entry")
	}
	if _, ok := got["@@locale"]; ok {
		t.Fatalf("metadata key @@locale must not be parsed as translatable entry")
	}
}

func TestARBParserRejectsNonStringMessageValues(t *testing.T) {
	_, err := (ARBParser{}).Parse([]byte(`{
  "count": 1
}`))
	if err == nil {
		t.Fatalf("expected parse error for non-string message value")
	}
}

func TestMarshalARBPreservesMetadataAndICUContent(t *testing.T) {
	template := []byte(`{
  "@@locale": "en",
  "inviteCount": "{count, plural, =0{No invites} one{1 invite} other{{count} invites}}",
  "@inviteCount": {
    "description": "Inbox invitation count",
    "placeholders": {
      "count": {
        "type": "int"
      }
    }
  },
  "hello": "Hello {name}",
  "@hello": {
    "description": "Greets the user",
    "placeholders": {
      "name": {}
    }
  }
}`)

	out, err := MarshalARB(template, template, map[string]string{
		"inviteCount": "{count, plural, =0{Aucune invitation} one{1 invitation} other{{count} invitations}}",
		"hello":       "Bonjour {name}",
	}, "fr")
	if err != nil {
		t.Fatalf("marshal arb: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("decode marshaled arb: %v", err)
	}

	if payload["hello"] != "Bonjour {name}" {
		t.Fatalf("unexpected hello translation: %#v", payload["hello"])
	}
	if payload["inviteCount"] != "{count, plural, =0{Aucune invitation} one{1 invitation} other{{count} invitations}}" {
		t.Fatalf("unexpected inviteCount translation: %#v", payload["inviteCount"])
	}
	if payload["@@locale"] != "fr" {
		t.Fatalf("expected @@locale metadata preserved, got %#v", payload["@@locale"])
	}

	metaHello, ok := payload["@hello"].(map[string]any)
	if !ok {
		t.Fatalf("expected @hello metadata object, got %#v", payload["@hello"])
	}
	if metaHello["description"] != "Greets the user" {
		t.Fatalf("expected @hello.description metadata preserved, got %#v", metaHello["description"])
	}
}

func TestARBParserParseWithContextIncludesDescriptions(t *testing.T) {
	content := []byte(`{
  "hello": "Hello",
  "@hello": {
    "description": "Greeting"
  },
  "bye": "Bye"
}`)

	messages, contextByKey, err := (ARBParser{}).ParseWithContext(content)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if messages["hello"] != "Hello" {
		t.Fatalf("unexpected hello translation: %q", messages["hello"])
	}
	if contextByKey["hello"] != "Greeting" {
		t.Fatalf("unexpected hello context: %q", contextByKey["hello"])
	}
	if _, ok := contextByKey["bye"]; ok {
		t.Fatalf("did not expect context for bye")
	}
}

func TestARBParserParseWithContextIgnoresInvalidDescriptions(t *testing.T) {
	testCases := []struct {
		name    string
		content []byte
	}{
		{
			name: "metadata is not object",
			content: []byte(`{
  "hello": "Hello",
  "@hello": "not-an-object"
}`),
		},
		{
			name: "description is non-string",
			content: []byte(`{
  "hello": "Hello",
  "@hello": {
    "description": 123
  }
}`),
		},
		{
			name: "description is whitespace only",
			content: []byte(`{
  "hello": "Hello",
  "@hello": {
    "description": "   "
  }
}`),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			messages, contextByKey, err := (ARBParser{}).ParseWithContext(tc.content)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if messages["hello"] != "Hello" {
				t.Fatalf("unexpected hello translation: %q", messages["hello"])
			}
			if contextByKey != nil {
				if _, ok := contextByKey["hello"]; ok {
					t.Fatalf("did not expect context for hello, got %q", contextByKey["hello"])
				}
			}
		})
	}
}

func TestARBParserParseWithContextNullPayload(t *testing.T) {
	messages, contextByKey, err := (ARBParser{}).ParseWithContext([]byte("null"))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(messages) != 0 {
		t.Fatalf("expected empty messages map, got %#v", messages)
	}
	if contextByKey != nil {
		t.Fatalf("expected nil context map, got %#v", contextByKey)
	}
}

func TestMarshalARBStructureFirstAndDeterministicAppend(t *testing.T) {
	template := []byte(`{
  "@@locale": "fr",
  "hello": "Bonjour",
  "@hello": {
    "description": "target-hello"
  },
  "bye": "Au revoir",
  "@bye": {
    "description": "farewell"
  },
  "@custom": {
    "owner": "mobile"
  }
}`)

	out, err := MarshalARB(template, template, map[string]string{
		"hello": "Salut",
		"new_b": "B",
		"new_a": "A",
	}, "fr")
	if err != nil {
		t.Fatalf("marshal arb: %v", err)
	}

	rendered := string(out)
	helloIdx := strings.Index(rendered, `"hello": "Salut"`)
	metaIdx := strings.Index(rendered, `"@hello": {`)
	customIdx := strings.Index(rendered, `"@custom": {`)
	newAIdx := strings.Index(rendered, `"new_a": "A"`)
	newBIdx := strings.Index(rendered, `"new_b": "B"`)
	if helloIdx == -1 || metaIdx == -1 || customIdx == -1 || newAIdx == -1 || newBIdx == -1 {
		t.Fatalf("expected structure and new keys in rendered output, got %q", rendered)
	}
	if metaIdx < helloIdx {
		t.Fatalf("expected existing metadata to remain after existing key, got %q", rendered)
	}
	if customIdx < metaIdx {
		t.Fatalf("expected unrelated metadata block retained in original position, got %q", rendered)
	}
	if newAIdx >= newBIdx || customIdx >= newAIdx {
		t.Fatalf("expected new keys appended in sorted order after template structure, got %q", rendered)
	}
	if strings.Contains(rendered, `"bye":`) {
		t.Fatalf("expected missing translatable key to be removed, got %q", rendered)
	}
	if strings.Contains(rendered, `"@bye":`) {
		t.Fatalf("expected metadata for removed key to be pruned, got %q", rendered)
	}
	if !strings.Contains(rendered, `"@custom": {`) {
		t.Fatalf("expected unrelated metadata blocks retained, got %q", rendered)
	}
}

func TestMarshalARBDoesNotTreatTemplateOrphanMetadataAsNewMessageMetadata(t *testing.T) {
	template := []byte(`{
  "@@locale": "fr",
  "@foo": {
    "description": "my new key"
  },
  "existing": "value"
}`)

	out, err := MarshalARB(template, template, map[string]string{
		"existing": "valeur",
		"foo":      "translated",
	}, "fr")
	if err != nil {
		t.Fatalf("marshal arb: %v", err)
	}

	rendered := string(out)
	existingIdx := strings.Index(rendered, `"existing": "valeur"`)
	metaIdx := strings.Index(rendered, `"@foo": {`)
	fooIdx := strings.Index(rendered, `"foo": "translated"`)
	if existingIdx == -1 || metaIdx == -1 || fooIdx == -1 {
		t.Fatalf("expected existing key, orphan metadata, and appended foo in rendered output, got %q", rendered)
	}
	if metaIdx >= existingIdx || existingIdx >= fooIdx {
		t.Fatalf("expected orphan metadata to stay in template position and new foo appended later, got %q", rendered)
	}
}

func TestMarshalARBCarriesSourceMetadataForAppendedKeys(t *testing.T) {
	targetTemplate := []byte(`{
  "@@locale": "fr",
  "existing": "value"
}`)

	sourceTemplate := []byte(`{
  "@@locale": "en",
  "existing": "value",
  "newKey": "New message",
  "@newKey": {
    "description": "Shown on the welcome screen",
    "placeholders": {}
  }
}`)

	out, err := MarshalARB(targetTemplate, sourceTemplate, map[string]string{
		"existing": "valeur",
		"newKey":   "Nouveau message",
	}, "fr")
	if err != nil {
		t.Fatalf("marshal arb: %v", err)
	}

	rendered := string(out)
	newKeyIdx := strings.Index(rendered, `"newKey": "Nouveau message"`)
	metaIdx := strings.Index(rendered, `"@newKey": {`)
	if newKeyIdx == -1 || metaIdx == -1 {
		t.Fatalf("expected appended newKey and carried source metadata in rendered output, got %q", rendered)
	}
	if metaIdx < newKeyIdx {
		t.Fatalf("expected carried source metadata to be written after appended newKey, got %q", rendered)
	}

	var payload map[string]any
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("decode marshaled arb: %v", err)
	}
	meta, ok := payload["@newKey"].(map[string]any)
	if !ok {
		t.Fatalf("expected @newKey metadata object, got %#v", payload["@newKey"])
	}
	if meta["description"] != "Shown on the welcome screen" {
		t.Fatalf("expected @newKey.description metadata preserved, got %#v", meta["description"])
	}
}

func TestMarshalARBNormalizesPreservedMetadataIndentation(t *testing.T) {
	template := []byte("{\n    \"@@locale\": \"fr\",\n    \"hello\": \"Bonjour\",\n    \"@hello\": {\n        \"description\": \"Greeting\"\n    }\n}")

	out, err := MarshalARB(template, template, map[string]string{
		"hello": "Salut",
	}, "fr")
	if err != nil {
		t.Fatalf("marshal arb: %v", err)
	}

	rendered := string(out)
	if !strings.Contains(rendered, "\"@hello\": {\n    \"description\": \"Greeting\"\n  }") {
		t.Fatalf("expected preserved metadata block to be reindented with two-space nesting, got %q", rendered)
	}
}

func TestMarshalARBSkipsSourceMetadataWhenTargetAlreadyHasOrphanedMetadataKey(t *testing.T) {
	targetTemplate := []byte(`{
  "@@locale": "fr",
  "@foo": {
    "description": "old orphan"
  },
  "existing": "value"
}`)

	sourceTemplate := []byte(`{
  "@@locale": "en",
  "existing": "value",
  "foo": "Foo",
  "@foo": {
    "description": "source meta"
  }
}`)

	out, err := MarshalARB(targetTemplate, sourceTemplate, map[string]string{
		"existing": "valeur",
		"foo":      "translated",
	}, "fr")
	if err != nil {
		t.Fatalf("marshal arb: %v", err)
	}

	rendered := string(out)
	if strings.Count(rendered, "\"@foo\": {") != 1 {
		t.Fatalf("expected exactly one @foo metadata block, got %q", rendered)
	}
	if !strings.Contains(rendered, "\"description\": \"old orphan\"") {
		t.Fatalf("expected orphaned target metadata to be retained, got %q", rendered)
	}
	if strings.Contains(rendered, "\"description\": \"source meta\"") {
		t.Fatalf("expected source metadata to be skipped when @foo already exists, got %q", rendered)
	}
}

func TestMarshalARBRemovesAllMessagesWhenValuesEmpty(t *testing.T) {
	template := []byte(`{
  "@@locale": "fr",
  "hello": "Bonjour",
  "@hello": {
    "description": "Greeting"
  },
  "@custom": {
    "owner": "mobile"
  }
}`)

	out, err := MarshalARB(template, template, map[string]string{}, "fr")
	if err != nil {
		t.Fatalf("marshal arb: %v", err)
	}

	rendered := string(out)
	if strings.Contains(rendered, `"hello":`) {
		t.Fatalf("expected hello key to be removed, got %q", rendered)
	}
	if strings.Contains(rendered, `"@hello":`) {
		t.Fatalf("expected @hello metadata to be removed with its message, got %q", rendered)
	}
	if !strings.Contains(rendered, `"@@locale": "fr"`) {
		t.Fatalf("expected locale metadata to match target locale, got %q", rendered)
	}
	if !strings.Contains(rendered, `"@custom": {`) {
		t.Fatalf("expected unrelated metadata to be preserved, got %q", rendered)
	}
}

func TestMarshalARBAppendsAllValuesWhenTargetTemplateEmpty(t *testing.T) {
	targetTemplate := []byte(`{}`)
	sourceTemplate := []byte(`{
  "newKey": "New message",
  "@newKey": {
    "description": "Shown on the welcome screen"
  }
}`)

	out, err := MarshalARB(targetTemplate, sourceTemplate, map[string]string{
		"newKey": "Nouveau message",
	}, "fr")
	if err != nil {
		t.Fatalf("marshal arb: %v", err)
	}

	rendered := string(out)
	newKeyIdx := strings.Index(rendered, `"newKey": "Nouveau message"`)
	metaIdx := strings.Index(rendered, `"@newKey": {`)
	if newKeyIdx == -1 || metaIdx == -1 {
		t.Fatalf("expected appended message and metadata, got %q", rendered)
	}
	if metaIdx < newKeyIdx {
		t.Fatalf("expected metadata to be written after appended key, got %q", rendered)
	}
}

func TestMarshalARBRejectsInvalidTemplates(t *testing.T) {
	validTemplate := []byte(`{"hello":"Hello"}`)

	testCases := []struct {
		name           string
		template       []byte
		sourceTemplate []byte
	}{
		{
			name:           "template must be object",
			template:       []byte(`[]`),
			sourceTemplate: validTemplate,
		},
		{
			name:           "template rejects trailing tokens",
			template:       []byte(`{"hello":"Hello"} {}`),
			sourceTemplate: validTemplate,
		},
		{
			name:           "source template rejects trailing tokens",
			template:       validTemplate,
			sourceTemplate: []byte(`{"hello":"Hello"} {}`),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := MarshalARB(tc.template, tc.sourceTemplate, map[string]string{
				"hello": "Bonjour",
			}, "fr")
			if err == nil {
				t.Fatalf("expected marshal error")
			}
			if !strings.Contains(err.Error(), "arb decode:") {
				t.Fatalf("expected arb decode error, got %v", err)
			}
		})
	}
}

package translationfileparser

import (
	"encoding/json"
	"testing"
)

func TestXCStringsParserParse(t *testing.T) {
	content := []byte(`{
  "sourceLanguage" : "en",
  "version" : "1.0",
  "strings" : {
    "greeting" : {
      "comment" : "Shown on home screen",
      "localizations" : {
        "en" : {
          "stringUnit" : {
            "state" : "translated",
            "value" : "Hello"
          }
        }
      }
    },
    "items_count" : {
      "localizations" : {
        "en" : {
          "variations" : {
            "plural" : {
              "one" : {
                "stringUnit" : {
                  "state" : "translated",
                  "value" : "%d item"
                }
              },
              "other" : {
                "stringUnit" : {
                  "state" : "translated",
                  "value" : "%d items"
                }
              }
            }
          }
        }
      }
    }
  }
}`)

	got, err := (XCStringsParser{}).Parse(content)
	if err != nil {
		t.Fatalf("parse xcstrings: %v", err)
	}

	if got["greeting"] != "Hello" {
		t.Fatalf("unexpected greeting translation: %q", got["greeting"])
	}
	if got["items_count.plural.one"] != "%d item" {
		t.Fatalf("unexpected plural one translation: %q", got["items_count.plural.one"])
	}
	if got["items_count.plural.other"] != "%d items" {
		t.Fatalf("unexpected plural other translation: %q", got["items_count.plural.other"])
	}
}

func TestMarshalXCStringsPreservesMetadataAndState(t *testing.T) {
	template := []byte(`{
  "sourceLanguage" : "en",
  "version" : "1.0",
  "strings" : {
    "greeting" : {
      "comment" : "Shown on home screen",
      "extractionState" : "manual",
      "localizations" : {
        "en" : {
          "stringUnit" : {
            "state" : "translated",
            "value" : "Hello"
          }
        }
      }
    },
    "items_count" : {
      "localizations" : {
        "en" : {
          "variations" : {
            "plural" : {
              "one" : {
                "stringUnit" : {
                  "state" : "translated",
                  "value" : "%d item"
                }
              },
              "other" : {
                "stringUnit" : {
                  "state" : "translated",
                  "value" : "%d items"
                }
              }
            }
          }
        }
      }
    }
  }
}`)

	out, err := MarshalXCStrings(template, map[string]string{
		"greeting":                 "Bonjour",
		"items_count.plural.one":   "%d article",
		"items_count.plural.other": "%d articles",
	})
	if err != nil {
		t.Fatalf("marshal xcstrings: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("decode marshaled output: %v", err)
	}

	stringsNode := payload["strings"].(map[string]any)
	greeting := stringsNode["greeting"].(map[string]any)
	if greeting["comment"] != "Shown on home screen" {
		t.Fatalf("expected comment metadata preserved, got %#v", greeting["comment"])
	}
	if greeting["extractionState"] != "manual" {
		t.Fatalf("expected extractionState metadata preserved, got %#v", greeting["extractionState"])
	}

	greetingLoc := greeting["localizations"].(map[string]any)["en"].(map[string]any)
	greetingUnit := greetingLoc["stringUnit"].(map[string]any)
	if greetingUnit["state"] != "translated" {
		t.Fatalf("expected state preserved, got %#v", greetingUnit["state"])
	}
	if greetingUnit["value"] != "Bonjour" {
		t.Fatalf("expected greeting value translated, got %#v", greetingUnit["value"])
	}

	itemLoc := stringsNode["items_count"].(map[string]any)["localizations"].(map[string]any)["en"].(map[string]any)
	plural := itemLoc["variations"].(map[string]any)["plural"].(map[string]any)
	oneUnit := plural["one"].(map[string]any)["stringUnit"].(map[string]any)
	otherUnit := plural["other"].(map[string]any)["stringUnit"].(map[string]any)
	if oneUnit["value"] != "%d article" {
		t.Fatalf("unexpected one value %#v", oneUnit["value"])
	}
	if otherUnit["value"] != "%d articles" {
		t.Fatalf("unexpected other value %#v", otherUnit["value"])
	}
}

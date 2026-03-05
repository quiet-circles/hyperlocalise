package translationfileparser

import (
	"encoding/json"
	"testing"
)

func TestXCStringsParserParsesValuesAndVariations(t *testing.T) {
	content := []byte(`{
  "sourceLanguage": "en",
  "version": "1.0",
  "strings": {
    "hello": {
      "comment": "Greeting",
      "localizations": {
        "en": {
          "stringUnit": {
            "state": "translated",
            "value": "Hello"
          }
        },
        "fr": {
          "stringUnit": {
            "state": "translated",
            "value": "Bonjour"
          }
        }
      }
    },
    "item_count": {
      "localizations": {
        "en": {
          "variations": {
            "plural": {
              "one": {
                "stringUnit": {
                  "state": "translated",
                  "value": "1 item"
                }
              },
              "other": {
                "stringUnit": {
                  "state": "translated",
                  "value": "%d items"
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

	if got["hello"] != "Hello" {
		t.Fatalf("unexpected hello translation: %q", got["hello"])
	}
	if got["item_count.plural.one"] != "1 item" {
		t.Fatalf("unexpected one translation: %q", got["item_count.plural.one"])
	}
	if got["item_count.plural.other"] != "%d items" {
		t.Fatalf("unexpected other translation: %q", got["item_count.plural.other"])
	}
}

func TestMarshalXCStringsPreservesStateAndUpdatesTargetLocale(t *testing.T) {
	template := []byte(`{
  "sourceLanguage": "en",
  "version": "1.0",
  "strings": {
    "hello": {
      "comment": "Greeting",
      "localizations": {
        "en": {
          "stringUnit": {
            "state": "translated",
            "value": "Hello"
          }
        },
        "fr": {
          "stringUnit": {
            "state": "needs_review",
            "value": "Bonjour"
          }
        }
      }
    },
    "item_count": {
      "localizations": {
        "fr": {
          "variations": {
            "plural": {
              "one": {
                "stringUnit": {
                  "state": "needs_review",
                  "value": "1 article"
                }
              },
              "other": {
                "stringUnit": {
                  "state": "needs_review",
                  "value": "%d articles"
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
		"hello":                   "Salut",
		"item_count.plural.one":   "1 element",
		"item_count.plural.other": "%d elements",
	}, "fr")
	if err != nil {
		t.Fatalf("marshal xcstrings: %v", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("decode marshaled xcstrings: %v", err)
	}

	stringsMap := payload["strings"].(map[string]any)
	hello := stringsMap["hello"].(map[string]any)
	helloLocs := hello["localizations"].(map[string]any)
	frUnit := helloLocs["fr"].(map[string]any)["stringUnit"].(map[string]any)
	enUnit := helloLocs["en"].(map[string]any)["stringUnit"].(map[string]any)

	if frUnit["value"] != "Salut" {
		t.Fatalf("expected fr hello updated, got %#v", frUnit["value"])
	}
	if frUnit["state"] != "needs_review" {
		t.Fatalf("expected fr state preserved, got %#v", frUnit["state"])
	}
	if enUnit["value"] != "Hello" {
		t.Fatalf("expected en locale to remain unchanged, got %#v", enUnit["value"])
	}

	itemCount := stringsMap["item_count"].(map[string]any)
	itemLocs := itemCount["localizations"].(map[string]any)
	plural := itemLocs["fr"].(map[string]any)["variations"].(map[string]any)["plural"].(map[string]any)

	one := plural["one"].(map[string]any)["stringUnit"].(map[string]any)
	other := plural["other"].(map[string]any)["stringUnit"].(map[string]any)
	if one["value"] != "1 element" {
		t.Fatalf("expected plural.one updated, got %#v", one["value"])
	}
	if other["value"] != "%d elements" {
		t.Fatalf("expected plural.other updated, got %#v", other["value"])
	}
	if one["state"] != "needs_review" || other["state"] != "needs_review" {
		t.Fatalf("expected plural state preserved, got one=%#v other=%#v", one["state"], other["state"])
	}
}

func TestMarshalXCStringsRejectsEmptyTargetLocale(t *testing.T) {
	template := []byte(`{
  "sourceLanguage": "en",
  "version": "1.0",
  "strings": {
    "hello": {
      "localizations": {
        "en": {
          "stringUnit": {
            "state": "translated",
            "value": "Hello"
          }
        }
      }
    }
  }
}`)

	_, err := MarshalXCStrings(template, map[string]string{"hello": "Bonjour"}, "")
	if err == nil {
		t.Fatalf("expected empty target locale error")
	}
}

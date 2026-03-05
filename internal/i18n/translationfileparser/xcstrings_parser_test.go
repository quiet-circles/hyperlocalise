package translationfileparser

import (
	"encoding/json"
	"strings"
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

func TestXCStringsParserRejectsMissingDeclaredSourceLocalization(t *testing.T) {
	content := []byte(`{
  "sourceLanguage": "en",
  "version": "1.0",
  "strings": {
    "hello": {
      "localizations": {
        "de": {
          "stringUnit": {
            "state": "translated",
            "value": "Hallo"
          }
        }
      }
    }
  }
}`)

	_, err := (XCStringsParser{}).Parse(content)
	if err == nil {
		t.Fatalf("expected missing source localization error")
	}
	if !strings.Contains(err.Error(), `source localization "en" not found`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestXCStringsParserFallsBackDeterministicallyWhenSourceLanguageMissing(t *testing.T) {
	content := []byte(`{
  "version": "1.0",
  "strings": {
    "hello": {
      "localizations": {
        "zh-Hans": {
          "stringUnit": {
            "state": "translated",
            "value": "你好"
          }
        },
        "de": {
          "stringUnit": {
            "state": "translated",
            "value": "Hallo"
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

	// sourceLanguage is absent, so parser falls back to first locale key (sorted).
	if got["hello"] != "Hallo" {
		t.Fatalf("unexpected fallback locale value: %q", got["hello"])
	}
}

func TestXCStringsParserParseForLocaleReadsRequestedLocale(t *testing.T) {
	content := []byte(`{
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
        },
        "fr": {
          "stringUnit": {
            "state": "translated",
            "value": "Bonjour"
          }
        }
      }
    }
  }
}`)

	got, err := (XCStringsParser{}).ParseForLocale(content, "fr")
	if err != nil {
		t.Fatalf("parse xcstrings for locale: %v", err)
	}
	if got["hello"] != "Bonjour" {
		t.Fatalf("expected requested locale value, got %q", got["hello"])
	}
}

func TestXCStringsParserParseForLocaleMissingLocaleDoesNotFallback(t *testing.T) {
	content := []byte(`{
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

	got, err := (XCStringsParser{}).ParseForLocale(content, "fr")
	if err != nil {
		t.Fatalf("parse xcstrings for locale: %v", err)
	}
	if _, ok := got["hello"]; ok {
		t.Fatalf("expected missing preferred locale to skip key instead of falling back to source")
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

func TestMarshalXCStringsCreatesMissingTargetLocaleFromSource(t *testing.T) {
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

	out, err := MarshalXCStrings(template, map[string]string{
		"hello":                   "Bonjour",
		"item_count.plural.one":   "1 article",
		"item_count.plural.other": "%d articles",
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
	enHello := helloLocs["en"].(map[string]any)["stringUnit"].(map[string]any)
	frHello := helloLocs["fr"].(map[string]any)["stringUnit"].(map[string]any)
	if enHello["value"] != "Hello" {
		t.Fatalf("expected source locale value unchanged, got %#v", enHello["value"])
	}
	if frHello["value"] != "Bonjour" {
		t.Fatalf("expected cloned target locale value updated, got %#v", frHello["value"])
	}
	if frHello["state"] != "needs_review" {
		t.Fatalf("expected cloned target locale state reset to needs_review, got %#v", frHello["state"])
	}

	itemCount := stringsMap["item_count"].(map[string]any)
	itemLocs := itemCount["localizations"].(map[string]any)
	enPlural := itemLocs["en"].(map[string]any)["variations"].(map[string]any)["plural"].(map[string]any)
	frPlural := itemLocs["fr"].(map[string]any)["variations"].(map[string]any)["plural"].(map[string]any)

	enOne := enPlural["one"].(map[string]any)["stringUnit"].(map[string]any)
	enOther := enPlural["other"].(map[string]any)["stringUnit"].(map[string]any)
	frOne := frPlural["one"].(map[string]any)["stringUnit"].(map[string]any)
	frOther := frPlural["other"].(map[string]any)["stringUnit"].(map[string]any)

	if enOne["value"] != "1 item" || enOther["value"] != "%d items" {
		t.Fatalf("expected source plural values unchanged, got one=%#v other=%#v", enOne["value"], enOther["value"])
	}
	if frOne["value"] != "1 article" || frOther["value"] != "%d articles" {
		t.Fatalf("expected cloned target plural values updated, got one=%#v other=%#v", frOne["value"], frOther["value"])
	}
	if frOne["state"] != "needs_review" || frOther["state"] != "needs_review" {
		t.Fatalf("expected cloned target plural states reset to needs_review, got one=%#v other=%#v", frOne["state"], frOther["state"])
	}
}

func TestMarshalXCStringsRejectsCloneWhenDeclaredSourceLocalizationMissing(t *testing.T) {
	template := []byte(`{
  "sourceLanguage": "en",
  "version": "1.0",
  "strings": {
    "hello": {
      "localizations": {
        "de": {
          "stringUnit": {
            "state": "translated",
            "value": "Hallo"
          }
        }
      }
    }
  }
}`)

	_, err := MarshalXCStrings(template, map[string]string{"hello": "Bonjour"}, "fr")
	if err == nil {
		t.Fatalf("expected missing source localization error during clone")
	}
	if !strings.Contains(err.Error(), `source localization "en" not found`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMarshalXCStringsResetsStateWhenExistingLocaleValueChanges(t *testing.T) {
	template := []byte(`{
  "sourceLanguage": "en",
  "version": "1.0",
  "strings": {
    "hello": {
      "localizations": {
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
        "fr": {
          "variations": {
            "plural": {
              "one": {
                "stringUnit": {
                  "state": "translated",
                  "value": "1 article"
                }
              },
              "other": {
                "stringUnit": {
                  "state": "translated",
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

	helloUnit := stringsMap["hello"].(map[string]any)["localizations"].(map[string]any)["fr"].(map[string]any)["stringUnit"].(map[string]any)
	if helloUnit["value"] != "Salut" {
		t.Fatalf("expected existing locale value updated, got %#v", helloUnit["value"])
	}
	if helloUnit["state"] != "needs_review" {
		t.Fatalf("expected existing locale state reset to needs_review, got %#v", helloUnit["state"])
	}

	plural := stringsMap["item_count"].(map[string]any)["localizations"].(map[string]any)["fr"].(map[string]any)["variations"].(map[string]any)["plural"].(map[string]any)
	one := plural["one"].(map[string]any)["stringUnit"].(map[string]any)
	other := plural["other"].(map[string]any)["stringUnit"].(map[string]any)
	if one["value"] != "1 element" || other["value"] != "%d elements" {
		t.Fatalf("expected existing plural values updated, got one=%#v other=%#v", one["value"], other["value"])
	}
	if one["state"] != "needs_review" || other["state"] != "needs_review" {
		t.Fatalf("expected existing plural states reset to needs_review, got one=%#v other=%#v", one["state"], other["state"])
	}
}

package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	testCases := []struct {
		name        string
		content     string
		errContains string
	}{
		{
			name: "valid minimal config",
			content: `{
			  "locale": {
			    "source": "en-US",
			    "targets": ["fr-FR"]
			  },
			  "buckets": {
			    "json": {"include": ["lang/[locale].json"]}
			  },
			  "llm": {
			    "default": {
			      "provider": "openai",
			      "model": "gpt-4.1-mini",
			      "prompt": "Translate from {source} to {target}."
			    }
			  }
			}`,
		},
		{
			name: "valid layered config",
			content: `{
			  "locale": {
			    "source": "en-US",
			    "targets": ["fr-FR", "fr-CA", "es-ES", "es-MX"],
			    "fallback": {
			      "fr-CA": ["fr-FR", "en-US"],
			      "es-MX": ["es-ES", "en-US"]
			    }
			  },
			  "buckets": {
			    "json": {"include": ["lang/[locale].json"]},
			    "markdown": {"include": ["_posts/[locale]/*.md"]}
			  },
			  "llm": {
			    "default": {
			      "provider": "openai",
			      "model": "gpt-4.1-mini",
			      "prompt": "Translate from {source} to {target}."
			    },
			    "groups": {
			      "fr": ["fr-FR", "fr-CA"]
			    },
			    "overrides": [
			      {
			        "match": {"group": "fr"},
			        "model": "gpt-4.1"
			      },
			      {
			        "match": {"targets": ["es-MX"]},
			        "prompt": "Use neutral Mexican Spanish."
			      }
			    ]
			  }
			}`,
		},
		{
			name: "invalid source inside targets",
			content: `{
			  "locale": {
			    "source": "en-US",
			    "targets": ["en-US", "fr-FR"]
			  },
			  "buckets": {
			    "json": {"include": ["lang/[locale].json"]}
			  },
			  "llm": {
			    "default": {
			      "provider": "openai",
			      "model": "gpt-4.1-mini",
			      "prompt": "Translate"
			    }
			  }
			}`,
			errContains: "source locale",
		},
		{
			name: "invalid fallback key not in targets",
			content: `{
			  "locale": {
			    "source": "en-US",
			    "targets": ["fr-FR"],
			    "fallback": {
			      "es-MX": ["en-US"]
			    }
			  },
			  "buckets": {
			    "json": {"include": ["lang/[locale].json"]}
			  },
			  "llm": {
			    "default": {
			      "provider": "openai",
			      "model": "gpt-4.1-mini",
			      "prompt": "Translate"
			    }
			  }
			}`,
			errContains: "fallback key",
		},
		{
			name: "invalid fallback self reference",
			content: `{
			  "locale": {
			    "source": "en-US",
			    "targets": ["fr-FR", "fr-CA"],
			    "fallback": {
			      "fr-CA": ["fr-CA", "en-US"]
			    }
			  },
			  "buckets": {
			    "json": {"include": ["lang/[locale].json"]}
			  },
			  "llm": {
			    "default": {
			      "provider": "openai",
			      "model": "gpt-4.1-mini",
			      "prompt": "Translate"
			    }
			  }
			}`,
			errContains: "self-reference",
		},
		{
			name: "invalid fallback cycle",
			content: `{
			  "locale": {
			    "source": "en-US",
			    "targets": ["fr-FR", "fr-CA", "es-ES"],
			    "fallback": {
			      "fr-FR": ["fr-CA"],
			      "fr-CA": ["fr-FR"],
			      "es-ES": ["en-US"]
			    }
			  },
			  "buckets": {
			    "json": {"include": ["lang/[locale].json"]}
			  },
			  "llm": {
			    "default": {
			      "provider": "openai",
			      "model": "gpt-4.1-mini",
			      "prompt": "Translate"
			    }
			  }
			}`,
			errContains: "cycle",
		},
		{
			name: "invalid override unknown group",
			content: `{
			  "locale": {
			    "source": "en-US",
			    "targets": ["fr-FR"]
			  },
			  "buckets": {
			    "json": {"include": ["lang/[locale].json"]}
			  },
			  "llm": {
			    "default": {
			      "provider": "openai",
			      "model": "gpt-4.1-mini",
			      "prompt": "Translate"
			    },
			    "overrides": [
			      {
			        "match": {"group": "fr"},
			        "model": "gpt-4.1"
			      }
			    ]
			  }
			}`,
			errContains: "unknown group",
		},
		{
			name: "invalid override unknown target locale",
			content: `{
			  "locale": {
			    "source": "en-US",
			    "targets": ["fr-FR"]
			  },
			  "buckets": {
			    "json": {"include": ["lang/[locale].json"]}
			  },
			  "llm": {
			    "default": {
			      "provider": "openai",
			      "model": "gpt-4.1-mini",
			      "prompt": "Translate"
			    },
			    "overrides": [
			      {
			        "match": {"targets": ["es-MX"]},
			        "prompt": "Use neutral Mexican Spanish."
			      }
			    ]
			  }
			}`,
			errContains: "must exist in locale.targets",
		},
		{
			name: "invalid override with neither group nor targets",
			content: `{
			  "locale": {
			    "source": "en-US",
			    "targets": ["fr-FR"]
			  },
			  "buckets": {
			    "json": {"include": ["lang/[locale].json"]}
			  },
			  "llm": {
			    "default": {
			      "provider": "openai",
			      "model": "gpt-4.1-mini",
			      "prompt": "Translate"
			    },
			    "overrides": [
			      {
			        "match": {},
			        "prompt": "Use neutral phrasing."
			      }
			    ]
			  }
			}`,
			errContains: "exactly one of group or targets",
		},
		{
			name: "invalid override with both group and targets",
			content: `{
			  "locale": {
			    "source": "en-US",
			    "targets": ["fr-FR"]
			  },
			  "buckets": {
			    "json": {"include": ["lang/[locale].json"]}
			  },
			  "llm": {
			    "default": {
			      "provider": "openai",
			      "model": "gpt-4.1-mini",
			      "prompt": "Translate"
			    },
			    "groups": {
			      "fr": ["fr-FR"]
			    },
			    "overrides": [
			      {
			        "match": {"group": "fr", "targets": ["fr-FR"]},
			        "model": "gpt-4.1"
			      }
			    ]
			  }
			}`,
			errContains: "exactly one of group or targets",
		},
		{
			name: "invalid no buckets",
			content: `{
			  "locale": {
			    "source": "en-US",
			    "targets": ["fr-FR"]
			  },
			  "buckets": {},
			  "llm": {
			    "default": {
			      "provider": "openai",
			      "model": "gpt-4.1-mini",
			      "prompt": "Translate"
			    }
			  }
			}`,
			errContains: "at least one of buckets.json or buckets.markdown",
		},
		{
			name: "invalid unknown json field rejected",
			content: `{
			  "locale": {
			    "source": "en-US",
			    "targets": ["fr-FR"],
			    "unknown": true
			  },
			  "buckets": {
			    "json": {"include": ["lang/[locale].json"]}
			  },
			  "llm": {
			    "default": {
			      "provider": "openai",
			      "model": "gpt-4.1-mini",
			      "prompt": "Translate"
			    }
			  }
			}`,
			errContains: "unknown field",
		},
	}

	for _, tc := range testCases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			path := writeConfigFile(t, tc.content)

			_, err := Load(path)
			if tc.errContains == "" {
				if err != nil {
					t.Fatalf("load config: %v", err)
				}

				return
			}

			if err == nil {
				t.Fatalf("expected error containing %q", tc.errContains)
			}

			if !strings.Contains(err.Error(), tc.errContains) {
				t.Fatalf("unexpected error: got %q want substring %q", err.Error(), tc.errContains)
			}
		})
	}
}

func TestLoadRejectsTrailingJSON(t *testing.T) {
	path := writeConfigFile(t, `{
	  "locale": {
	    "source": "en-US",
	    "targets": ["fr-FR"]
	  },
	  "buckets": {
	    "json": {"include": ["lang/[locale].json"]}
	  },
	  "llm": {
	    "default": {
	      "provider": "openai",
	      "model": "gpt-4.1-mini",
	      "prompt": "Translate from {source} to {target}."
	    }
	  }
	}
	{}`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected trailing JSON decode error")
	}

	if !strings.Contains(err.Error(), "unexpected trailing JSON value") {
		t.Fatalf("unexpected error: got %q", err.Error())
	}
}

func TestLoadUsesDefaultPathWhenEmpty(t *testing.T) {
	tempDir := t.TempDir()

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}

	t.Cleanup(func() {
		if chdirErr := os.Chdir(originalWD); chdirErr != nil {
			t.Fatalf("restore working directory: %v", chdirErr)
		}
	})

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("change working directory: %v", err)
	}

	defaultPath := filepath.Join(tempDir, ".i18n.json")
	content := `{
	  "locale": {
	    "source": "en-US",
	    "targets": ["fr-FR"]
	  },
	  "buckets": {
	    "json": {"include": ["lang/[locale].json"]}
	  },
	  "llm": {
	    "default": {
	      "provider": "openai",
	      "model": "gpt-4.1-mini",
	      "prompt": "Translate from {source} to {target}."
	    }
	  }
	}`

	if err := os.WriteFile(defaultPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write default config file: %v", err)
	}

	if _, err := Load(""); err != nil {
		t.Fatalf("load config from default path: %v", err)
	}
}

func TestLoadPrefersJSONCOverJSONWhenBothExist(t *testing.T) {
	tempDir := t.TempDir()

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}

	t.Cleanup(func() {
		if chdirErr := os.Chdir(originalWD); chdirErr != nil {
			t.Fatalf("restore working directory: %v", chdirErr)
		}
	})

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("change working directory: %v", err)
	}

	jsonPath := filepath.Join(tempDir, "i18n.json")
	jsonContent := `{
	  "locale": {
	    "source": "en-US",
	    "targets": ["fr-FR"]
	  },
	  "buckets": {
	    "json": {"include": ["lang/[locale].json"]}
	  },
	  "llm": {
	    "default": {
	      "provider": "openai",
	      "model": "json-model",
	      "prompt": "Translate"
	    }
	  }
	}`

	if err := os.WriteFile(jsonPath, []byte(jsonContent), 0o600); err != nil {
		t.Fatalf("write json config file: %v", err)
	}

	jsoncPath := filepath.Join(tempDir, "i18n.jsonc")
	jsoncContent := `{
	  // preferred file
	  "locale": {
	    "source": "en-US",
	    "targets": ["de-DE"],
	  },
	  "buckets": {
	    "json": {"include": ["lang/[locale].json"]},
	  },
	  "llm": {
	    "default": {
	      "provider": "openai",
	      "model": "jsonc-model",
	      "prompt": "Translate",
	    },
	  },
	}`

	if err := os.WriteFile(jsoncPath, []byte(jsoncContent), 0o600); err != nil {
		t.Fatalf("write jsonc config file: %v", err)
	}

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("load config from preferred jsonc path: %v", err)
	}

	if got, want := cfg.LLM.Default.Model, "jsonc-model"; got != want {
		t.Fatalf("unexpected model: got %q want %q", got, want)
	}
}

func TestLoadParsesJSONCFile(t *testing.T) {
	path := writeConfigFile(t, `{
	  // comment
	  "locale": {
	    "source": "en-US",
	    "targets": ["fr-FR"],
	  },
	  "buckets": {
	    "json": {"include": ["lang/[locale].json"]},
	  },
	  "llm": {
	    "default": {
	      "provider": "openai",
	      "model": "gpt-4.1-mini",
	      "prompt": "Translate from {source} to {target}.",
	    },
	  },
	}`)

	if _, err := Load(path); err != nil {
		t.Fatalf("load jsonc config: %v", err)
	}
}

func TestJSONSchema(t *testing.T) {
	schemaBytes, err := JSONSchema()
	if err != nil {
		t.Fatalf("generate schema: %v", err)
	}

	var schema map[string]any
	if err := json.Unmarshal(schemaBytes, &schema); err != nil {
		t.Fatalf("decode schema: %v", err)
	}

	if got, ok := schema["additionalProperties"].(bool); !ok || got {
		t.Fatalf("schema.additionalProperties should be false")
	}
}

func writeConfigFile(t *testing.T, content string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	return path
}

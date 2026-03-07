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
			name: "valid full config with jsonc comments",
			content: `{
			  "locales": {
			    "source": "en-US",
			    "targets": ["es-ES", "fr-FR", "de-DE"],
			    "fallbacks": {
			      // source is implicit fallback base
			      "fr-FR": ["es-ES"]
			    }
			  },
			  "buckets": {
			    "ui": {
			      "files": [
			        {
			          "from": "content/ui/{{source}}/strings.json",
			          "to": "dist/ui/{{target}}/strings.json"
			        },
			        {
			          "from": "content/ui/shared.json",
			          "to": "dist/ui/shared.{{target}}.json"
			        }
			      ]
			    },
			    "docs": {
			      "files": [
			        {
			          "from": "content/docs/{{target}}/guide.md",
			          "to": "dist/docs/{{target}}/guide.md"
			        }
			      ]
			    }
			  },
			  "groups": {
			    "modern": {
			      "targets": ["es-ES", "fr-FR"],
			      "buckets": ["ui"]
			    },
			    "docs-team": {
			      "targets": ["de-DE"],
			      "buckets": ["docs"]
			    }
			  },
			  "llm": {
			    "profiles": {
			      "default": {
			        "provider": "openai",
			        "model": "xmt-std",
			        "prompt": "Translate from {{source}} to {{target}}:\n\n{{input}}"
			      },
			      "verbose": {
			        "provider": "openai",
			        "model": "xmt-std",
			        "prompt": "Translate from {{source}} to {{target}} with richer context:\n\n{{input}}"
			      },
			      "precise": {
			        "provider": "openai",
			        "model": "xmt-std",
			        "prompt": "Translate from {{source}} to {{target}} precisely:\n\n{{input}}"
			      }
			    },
			    "rules": [
			      {"priority": 100, "group": "modern", "profile": "precise"},
			      {"priority": 50, "group": "docs-team", "profile": "verbose"}
			    ]
			  }
			}`,
		},
		{
			name: "valid group with empty targets",
			content: `{
			  "locales": {"source": "en-US", "targets": ["es-ES"]},
			  "buckets": {"ui": {"files": [{"from": "a", "to": "b"}]}},
			  "groups": {"g": {"targets": [], "buckets": ["ui"]}},
			  "llm": {"profiles": {"default": {"provider": "openai", "model": "x", "prompt": "p"}}}
			}`,
		},
		{
			name: "valid group with empty buckets",
			content: `{
			  "locales": {"source": "en-US", "targets": ["es-ES"]},
			  "buckets": {"ui": {"files": [{"from": "a", "to": "b"}]}},
			  "groups": {"g": {"targets": ["es-ES"], "buckets": []}},
			  "llm": {"profiles": {"default": {"provider": "openai", "model": "x", "prompt": "p"}}}
			}`,
		},
		{
			name: "valid llm profile without prompts",
			content: `{
			  "locales": {"source": "en-US", "targets": ["es-ES"]},
			  "buckets": {"ui": {"files": [{"from": "a", "to": "b"}]}},
			  "groups": {"g": {"targets": ["es-ES"], "buckets": ["ui"]}},
			  "llm": {"profiles": {"default": {"provider": "openai", "model": "x"}}}
			}`,
		},
		{
			name: "valid llm provider azure openai",
			content: `{
			  "locales": {"source": "en-US", "targets": ["es-ES"]},
			  "buckets": {"ui": {"files": [{"from": "a", "to": "b"}]}},
			  "groups": {"g": {"targets": ["es-ES"], "buckets": ["ui"]}},
			  "llm": {"profiles": {"default": {"provider": "azure_openai", "model": "x", "prompt": "p"}}}
			}`,
		},
		{
			name: "valid llm provider lmstudio",
			content: `{
			  "locales": {"source": "en-US", "targets": ["es-ES"]},
			  "buckets": {"ui": {"files": [{"from": "a", "to": "b"}]}},
			  "groups": {"g": {"targets": ["es-ES"], "buckets": ["ui"]}},
			  "llm": {"profiles": {"default": {"provider": "lmstudio", "model": "x", "prompt": "p"}}}
			}`,
		},
		{
			name: "valid llm provider groq",
			content: `{
			  "locales": {"source": "en-US", "targets": ["es-ES"]},
			  "buckets": {"ui": {"files": [{"from": "a", "to": "b"}]}},
			  "groups": {"g": {"targets": ["es-ES"], "buckets": ["ui"]}},
			  "llm": {"profiles": {"default": {"provider": "groq", "model": "x", "prompt": "p"}}}
			}`,
		},
		{
			name: "valid llm provider mistral",
			content: `{
			  "locales": {"source": "en-US", "targets": ["es-ES"]},
			  "buckets": {"ui": {"files": [{"from": "a", "to": "b"}]}},
			  "groups": {"g": {"targets": ["es-ES"], "buckets": ["ui"]}},
			  "llm": {"profiles": {"default": {"provider": "mistral", "model": "x", "prompt": "p"}}}
			}`,
		},
		{
			name: "valid llm provider ollama",
			content: `{
			  "locales": {"source": "en-US", "targets": ["es-ES"]},
			  "buckets": {"ui": {"files": [{"from": "a", "to": "b"}]}},
			  "groups": {"g": {"targets": ["es-ES"], "buckets": ["ui"]}},
			  "llm": {"profiles": {"default": {"provider": "ollama", "model": "x", "prompt": "p"}}}
			}`,
		},
		{
			name: "valid llm provider gemini",
			content: `{
			  "locales": {"source": "en-US", "targets": ["es-ES"]},
			  "buckets": {"ui": {"files": [{"from": "a", "to": "b"}]}},
			  "groups": {"g": {"targets": ["es-ES"], "buckets": ["ui"]}},
			  "llm": {"profiles": {"default": {"provider": "gemini", "model": "x", "prompt": "p"}}}
			}`,
		},
		{
			name: "valid llm provider anthropic",
			content: `{
			  "locales": {"source": "en-US", "targets": ["es-ES"]},
			  "buckets": {"ui": {"files": [{"from": "a", "to": "b"}]}},
			  "groups": {"g": {"targets": ["es-ES"], "buckets": ["ui"]}},
			  "llm": {"profiles": {"default": {"provider": "anthropic", "model": "x", "prompt": "p"}}}
			}`,
		},
		{
			name: "valid llm provider bedrock",
			content: `{
			  "locales": {"source": "en-US", "targets": ["es-ES"]},
			  "buckets": {"ui": {"files": [{"from": "a", "to": "b"}]}},
			  "groups": {"g": {"targets": ["es-ES"], "buckets": ["ui"]}},
			  "llm": {"profiles": {"default": {"provider": "bedrock", "model": "x", "prompt": "p"}}}
			}`,
		},
		{
			name: "valid llm context memory profile",
			content: `{
			  "locales": {"source": "en-US", "targets": ["es-ES"]},
			  "buckets": {"ui": {"files": [{"from": "a", "to": "b"}]}},
			  "groups": {"g": {"targets": ["es-ES"], "buckets": ["ui"]}},
			  "llm": {
			    "profiles": {"default": {"provider": "openai", "model": "x", "prompt": "p"}},
			    "context_memory": {"provider": "ollama", "model": "qwen2.5:7b"}
			  }
			}`,
		},
		{
			name: "invalid unknown field rejected",
			content: `{
			  "locales": {"source": "en-US", "targets": ["es-ES"]},
			  "buckets": {"ui": {"files": [{"from": "a", "to": "b"}], "unknown": true}},
			  "groups": {"g": {"targets": ["es-ES"], "buckets": ["ui"]}},
			  "llm": {"profiles": {"default": {"provider": "openai", "model": "x", "prompt": "p"}}}
			}`,
			errContains: "unknown field",
		},
		{
			name: "invalid source in targets",
			content: `{
			  "locales": {"source": "en-US", "targets": ["en-US", "es-ES"]},
			  "buckets": {"ui": {"files": [{"from": "a", "to": "b"}]}},
			  "groups": {"g": {"targets": ["es-ES"], "buckets": ["ui"]}},
			  "llm": {"profiles": {"default": {"provider": "openai", "model": "x", "prompt": "p"}}}
			}`,
			errContains: "source locale",
		},
		{
			name: "invalid fallback key not in targets",
			content: `{
			  "locales": {
			    "source": "en-US",
			    "targets": ["es-ES"],
			    "fallbacks": {"fr-FR": ["en-US"]}
			  },
			  "buckets": {"ui": {"files": [{"from": "a", "to": "b"}]}},
			  "groups": {"g": {"targets": ["es-ES"], "buckets": ["ui"]}},
			  "llm": {"profiles": {"default": {"provider": "openai", "model": "x", "prompt": "p"}}}
			}`,
			errContains: "fallback key",
		},
		{
			name: "invalid fallback candidate not in targets or source",
			content: `{
			  "locales": {
			    "source": "en-US",
			    "targets": ["es-ES", "fr-FR"],
			    "fallbacks": {"fr-FR": ["it-IT"]}
			  },
			  "buckets": {"ui": {"files": [{"from": "a", "to": "b"}]}},
			  "groups": {"g": {"targets": ["es-ES"], "buckets": ["ui"]}},
			  "llm": {"profiles": {"default": {"provider": "openai", "model": "x", "prompt": "p"}}}
			}`,
			errContains: "must be in locales.targets or locales.source",
		},
		{
			name: "invalid fallback cycle",
			content: `{
			  "locales": {
			    "source": "en-US",
			    "targets": ["es-ES", "fr-FR"],
			    "fallbacks": {"es-ES": ["fr-FR"], "fr-FR": ["es-ES"]}
			  },
			  "buckets": {"ui": {"files": [{"from": "a", "to": "b"}]}},
			  "groups": {"g": {"targets": ["es-ES"], "buckets": ["ui"]}},
			  "llm": {"profiles": {"default": {"provider": "openai", "model": "x", "prompt": "p"}}}
			}`,
			errContains: "cycle detected",
		},
		{
			name: "invalid empty buckets",
			content: `{
			  "locales": {"source": "en-US", "targets": ["es-ES"]},
			  "buckets": {},
			  "groups": {"g": {"targets": ["es-ES"], "buckets": ["ui"]}},
			  "llm": {"profiles": {"default": {"provider": "openai", "model": "x", "prompt": "p"}}}
			}`,
			errContains: "buckets: must not be empty",
		},
		{
			name: "invalid bucket files empty",
			content: `{
			  "locales": {"source": "en-US", "targets": ["es-ES"]},
			  "buckets": {"ui": {"files": []}},
			  "groups": {"g": {"targets": ["es-ES"], "buckets": ["ui"]}},
			  "llm": {"profiles": {"default": {"provider": "openai", "model": "x", "prompt": "p"}}}
			}`,
			errContains: "files: must not be empty",
		},
		{
			name: "invalid bucket file mapping missing from",
			content: `{
			  "locales": {"source": "en-US", "targets": ["es-ES"]},
			  "buckets": {"ui": {"files": [{"from": "", "to": "b"}]}},
			  "groups": {"g": {"targets": ["es-ES"], "buckets": ["ui"]}},
			  "llm": {"profiles": {"default": {"provider": "openai", "model": "x", "prompt": "p"}}}
			}`,
			errContains: ".from: must not be empty",
		},
		{
			name: "valid bucket file mapping same suffix md",
			content: `{
			  "locales": {"source": "en-US", "targets": ["es-ES"]},
			  "buckets": {"ui": {"files": [{"from": "content/about.md", "to": "content/about.es.md"}]}},
			  "groups": {"g": {"targets": ["es-ES"], "buckets": ["ui"]}},
			  "llm": {"profiles": {"default": {"provider": "openai", "model": "x", "prompt": "p"}}}
			}`,
		},
		{
			name: "valid bucket file mapping same suffix json",
			content: `{
			  "locales": {"source": "en-US", "targets": ["es-ES"]},
			  "buckets": {"ui": {"files": [{"from": "lang/en.json", "to": "lang/es.json"}]}},
			  "groups": {"g": {"targets": ["es-ES"], "buckets": ["ui"]}},
			  "llm": {"profiles": {"default": {"provider": "openai", "model": "x", "prompt": "p"}}}
			}`,
		},
		{
			name: "valid bucket file mapping same suffix no extension",
			content: `{
			  "locales": {"source": "en-US", "targets": ["es-ES"]},
			  "buckets": {"ui": {"files": [{"from": "Dockerfile", "to": "Dockerfile"}]}},
			  "groups": {"g": {"targets": ["es-ES"], "buckets": ["ui"]}},
			  "llm": {"profiles": {"default": {"provider": "openai", "model": "x", "prompt": "p"}}}
			}`,
		},
		{
			name: "invalid bucket file mapping suffix mismatch md to txt",
			content: `{
			  "locales": {"source": "en-US", "targets": ["es-ES"]},
			  "buckets": {"ui": {"files": [{"from": "content/about.md", "to": "content/about.txt"}]}},
			  "groups": {"g": {"targets": ["es-ES"], "buckets": ["ui"]}},
			  "llm": {"profiles": {"default": {"provider": "openai", "model": "x", "prompt": "p"}}}
			}`,
			errContains: "file suffix mismatch",
		},
		{
			name: "invalid bucket file mapping suffix mismatch json to md",
			content: `{
			  "locales": {"source": "en-US", "targets": ["es-ES"]},
			  "buckets": {"ui": {"files": [{"from": "lang/en.json", "to": "lang/en.md"}]}},
			  "groups": {"g": {"targets": ["es-ES"], "buckets": ["ui"]}},
			  "llm": {"profiles": {"default": {"provider": "openai", "model": "x", "prompt": "p"}}}
			}`,
			errContains: "file suffix mismatch",
		},
		{
			name: "invalid bucket file mapping suffix mismatch no ext to ext",
			content: `{
			  "locales": {"source": "en-US", "targets": ["es-ES"]},
			  "buckets": {"ui": {"files": [{"from": "Makefile", "to": "Makefile.md"}]}},
			  "groups": {"g": {"targets": ["es-ES"], "buckets": ["ui"]}},
			  "llm": {"profiles": {"default": {"provider": "openai", "model": "x", "prompt": "p"}}}
			}`,
			errContains: "file suffix mismatch",
		},
		{
			name: "invalid bucket file mapping suffix mismatch ext to no ext",
			content: `{
			  "locales": {"source": "en-US", "targets": ["es-ES"]},
			  "buckets": {"ui": {"files": [{"from": "about.md", "to": "about"}]}},
			  "groups": {"g": {"targets": ["es-ES"], "buckets": ["ui"]}},
			  "llm": {"profiles": {"default": {"provider": "openai", "model": "x", "prompt": "p"}}}
			}`,
			errContains: "file suffix mismatch",
		},
		{
			name: "valid bucket file mapping no extension in dotted directory",
			content: `{
			  "locales": {"source": "en-US", "targets": ["es-ES"]},
			  "buckets": {"ui": {"files": [{"from": "v1.5/Makefile", "to": "v2.0/Makefile"}]}},
			  "groups": {"g": {"targets": ["es-ES"], "buckets": ["ui"]}},
			  "llm": {"profiles": {"default": {"provider": "openai", "model": "x", "prompt": "p"}}}
			}`,
		},
		{
			name: "valid bucket file mapping same extension in dotted directory",
			content: `{
			  "locales": {"source": "en-US", "targets": ["es-ES"]},
			  "buckets": {"ui": {"files": [{"from": "my.project/config.yaml", "to": "my.project/config.es.yaml"}]}},
			  "groups": {"g": {"targets": ["es-ES"], "buckets": ["ui"]}},
			  "llm": {"profiles": {"default": {"provider": "openai", "model": "x", "prompt": "p"}}}
			}`,
		},
		{
			name: "invalid group targets and buckets both empty",
			content: `{
			  "locales": {"source": "en-US", "targets": ["es-ES"]},
			  "buckets": {"ui": {"files": [{"from": "a", "to": "b"}]}},
			  "groups": {"g": {"targets": [], "buckets": []}},
			  "llm": {"profiles": {"default": {"provider": "openai", "model": "x", "prompt": "p"}}}
			}`,
			errContains: "cannot both be empty",
		},
		{
			name: "invalid group unknown bucket",
			content: `{
			  "locales": {"source": "en-US", "targets": ["es-ES"]},
			  "buckets": {"ui": {"files": [{"from": "a", "to": "b"}]}},
			  "groups": {"g": {"targets": ["es-ES"], "buckets": ["docs"]}},
			  "llm": {"profiles": {"default": {"provider": "openai", "model": "x", "prompt": "p"}}}
			}`,
			errContains: "must exist in buckets",
		},
		{
			name: "invalid group unknown locale target",
			content: `{
			  "locales": {"source": "en-US", "targets": ["es-ES"]},
			  "buckets": {"ui": {"files": [{"from": "a", "to": "b"}]}},
			  "groups": {"g": {"targets": ["de-DE"], "buckets": ["ui"]}},
			  "llm": {"profiles": {"default": {"provider": "openai", "model": "x", "prompt": "p"}}}
			}`,
			errContains: "must exist in locales.targets",
		},
		{
			name: "invalid llm missing default profile",
			content: `{
			  "locales": {"source": "en-US", "targets": ["es-ES"]},
			  "buckets": {"ui": {"files": [{"from": "a", "to": "b"}]}},
			  "groups": {"g": {"targets": ["es-ES"], "buckets": ["ui"]}},
			  "llm": {"profiles": {"verbose": {"provider": "openai", "model": "x", "prompt": "p"}}}
			}`,
			errContains: "llm.profiles.default: is required",
		},
		{
			name: "invalid llm unsupported provider",
			content: `{
			  "locales": {"source": "en-US", "targets": ["es-ES"]},
			  "buckets": {"ui": {"files": [{"from": "a", "to": "b"}]}},
			  "groups": {"g": {"targets": ["es-ES"], "buckets": ["ui"]}},
			  "llm": {"profiles": {"default": {"provider": "unsupported", "model": "x", "prompt": "p"}}}
			}`,
			errContains: "unsupported provider",
		},
		{
			name: "invalid llm context memory missing model",
			content: `{
			  "locales": {"source": "en-US", "targets": ["es-ES"]},
			  "buckets": {"ui": {"files": [{"from": "a", "to": "b"}]}},
			  "groups": {"g": {"targets": ["es-ES"], "buckets": ["ui"]}},
			  "llm": {
			    "profiles": {"default": {"provider": "openai", "model": "x", "prompt": "p"}},
			    "context_memory": {"provider": "ollama"}
			  }
			}`,
			errContains: "llm.context_memory.model: must not be empty",
		},
		{
			name: "invalid llm context memory unsupported provider",
			content: `{
			  "locales": {"source": "en-US", "targets": ["es-ES"]},
			  "buckets": {"ui": {"files": [{"from": "a", "to": "b"}]}},
			  "groups": {"g": {"targets": ["es-ES"], "buckets": ["ui"]}},
			  "llm": {
			    "profiles": {"default": {"provider": "openai", "model": "x", "prompt": "p"}},
			    "context_memory": {"provider": "unsupported", "model": "m"}
			  }
			}`,
			errContains: "llm.context_memory.provider: unsupported provider",
		},
		{
			name: "invalid llm context memory empty provider",
			content: `{
			  "locales": {"source": "en-US", "targets": ["es-ES"]},
			  "buckets": {"ui": {"files": [{"from": "a", "to": "b"}]}},
			  "groups": {"g": {"targets": ["es-ES"], "buckets": ["ui"]}},
			  "llm": {
			    "profiles": {"default": {"provider": "openai", "model": "x", "prompt": "p"}},
			    "context_memory": {"provider": "   ", "model": "m"}
			  }
			}`,
			errContains: "llm.context_memory.provider: must not be empty",
		},
		{
			name: "invalid llm rule unknown group",
			content: `{
			  "locales": {"source": "en-US", "targets": ["es-ES"]},
			  "buckets": {"ui": {"files": [{"from": "a", "to": "b"}]}},
			  "groups": {"g": {"targets": ["es-ES"], "buckets": ["ui"]}},
			  "llm": {
			    "profiles": {"default": {"provider": "openai", "model": "x", "prompt": "p"}},
			    "rules": [{"priority": 1, "group": "missing", "profile": "default"}]
			  }
			}`,
			errContains: "unknown group",
		},
		{
			name: "invalid llm rule unknown profile",
			content: `{
			  "locales": {"source": "en-US", "targets": ["es-ES"]},
			  "buckets": {"ui": {"files": [{"from": "a", "to": "b"}]}},
			  "groups": {"g": {"targets": ["es-ES"], "buckets": ["ui"]}},
			  "llm": {
			    "profiles": {"default": {"provider": "openai", "model": "x", "prompt": "p"}},
			    "rules": [{"priority": 1, "group": "g", "profile": "missing"}]
			  }
			}`,
			errContains: "unknown profile",
		},
		{
			name: "invalid llm rule negative priority",
			content: `{
			  "locales": {"source": "en-US", "targets": ["es-ES"]},
			  "buckets": {"ui": {"files": [{"from": "a", "to": "b"}]}},
			  "groups": {"g": {"targets": ["es-ES"], "buckets": ["ui"]}},
			  "llm": {
			    "profiles": {"default": {"provider": "openai", "model": "x", "prompt": "p"}},
			    "rules": [{"priority": -1, "group": "g", "profile": "default"}]
			  }
			}`,
			errContains: "must be >= 0",
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
	  "locales": {"source": "en-US", "targets": ["fr-FR"]},
	  "buckets": {"ui": {"files": [{"from": "a", "to": "b"}]}},
	  "groups": {"g": {"targets": ["fr-FR"], "buckets": ["ui"]}},
	  "llm": {"profiles": {"default": {"provider": "openai", "model": "x", "prompt": "p"}}}
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

func TestLoadAcceptsBlockComments(t *testing.T) {
	path := writeConfigFile(t, `{
	  "locales": {
	    "source": "en-US",
	    "targets": ["fr-FR"]
	  },
	  "buckets": {
	    "ui": {
	      "files": [
	        {"from": "a", "to": "b"}
	      ]
	    }
	  },
	  /* this block comment should be stripped */
	  "groups": {
	    "g": {
	      "targets": ["fr-FR"],
	      "buckets": ["ui"]
	    }
	  },
	  "llm": {
	    "profiles": {
	      "default": {
	        "provider": "openai",
	        "model": "x",
	        "prompt": "p"
	      }
	    }
	  }
	}`)

	if _, err := Load(path); err != nil {
		t.Fatalf("load config with block comments: %v", err)
	}
}

func TestLoadAcceptsOptionalSystemAndUserPrompt(t *testing.T) {
	path := writeConfigFile(t, `{
	  "locales": {"source": "en-US", "targets": ["fr-FR"]},
	  "buckets": {"ui": {"files": [{"from": "a", "to": "b"}]}},
	  "groups": {"g": {"targets": ["fr-FR"], "buckets": ["ui"]}},
	  "llm": {
	    "profiles": {
	      "default": {
	        "provider": "openai",
	        "model": "x",
	        "prompt": "Translate {{input}}",
	        "system_prompt": "System {{source}} -> {{target}}",
	        "user_prompt": "User: {{input}}"
	      }
	    }
	  }
	}`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load config with optional prompt fields: %v", err)
	}

	profile := cfg.LLM.Profiles["default"]
	if profile.SystemPrompt == "" {
		t.Fatalf("expected system_prompt to be loaded")
	}
	if profile.UserPrompt == "" {
		t.Fatalf("expected user_prompt to be loaded")
	}
}

func TestLoadRejectsUnterminatedBlockComment(t *testing.T) {
	path := writeConfigFile(t, `{
	  "locales": {"source": "en-US", "targets": ["fr-FR"]},
	  /* broken
	  "buckets": {"ui": {"files": [{"from": "a", "to": "b"}]}},
	  "groups": {"g": {"targets": ["fr-FR"], "buckets": ["ui"]}},
	  "llm": {"profiles": {"default": {"provider": "openai", "model": "x", "prompt": "p"}}}
	}`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected comment parsing error")
	}

	if !strings.Contains(err.Error(), "unexpected EOF") {
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

	defaultPath := filepath.Join(tempDir, "i18n.jsonc")
	content := `{
	  "locales": {"source": "en-US", "targets": ["fr-FR"]},
	  "buckets": {"ui": {"files": [{"from": "a", "to": "b"}]}},
	  "groups": {"g": {"targets": ["fr-FR"], "buckets": ["ui"]}},
	  "llm": {"profiles": {"default": {"provider": "openai", "model": "x", "prompt": "p"}}}
	}`

	if err := os.WriteFile(defaultPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write default config file: %v", err)
	}

	if _, err := Load(""); err != nil {
		t.Fatalf("load config from default path: %v", err)
	}
}

func TestLoadIgnoresHiddenJSONCPathWhenDefaultMissing(t *testing.T) {
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

	jsoncPath := filepath.Join(tempDir, ".i18n.jsonc")
	jsoncContent := `{
	  // hidden default file
	  "locales": {
	    "source": "en-US",
	    "targets": ["de-DE"]
	  },
	  "buckets": {
	    "json": {
	      "files": [
	        {"from": "lang/{{source}}.json", "to": "lang/{{target}}.json"}
	      ]
	    }
	  },
	  "groups": {
	    "all": {
	      "targets": ["de-DE"],
	      "buckets": ["json"]
	    }
	  },
	  "llm": {
	    "profiles": {
	      "default": {
	        "provider": "openai",
	        "model": "jsonc-model",
	        "prompt": "Translate"
	      }
	    }
	  }
	}`

	if err := os.WriteFile(jsoncPath, []byte(jsoncContent), 0o600); err != nil {
		t.Fatalf("write jsonc config file: %v", err)
	}

	_, err = Load("")
	if err == nil {
		t.Fatal("expected load to fail when only .i18n.jsonc exists")
	}

	if !strings.Contains(err.Error(), "open i18n config") {
		t.Fatalf("unexpected error: got %q", err.Error())
	}
}

func TestLoadParsesJSONCFile(t *testing.T) {
	path := writeConfigFile(t, `{
	  // comment
	  "locales": {
	    "source": "en-US",
	    "targets": ["fr-FR"]
	  },
	  "buckets": {
	    "json": {
	      "files": [
	        {"from": "lang/{{source}}.json", "to": "lang/{{target}}.json"}
	      ]
	    }
	  },
	  "groups": {
	    "all": {
	      "targets": ["fr-FR"],
	      "buckets": ["json"]
	    }
	  },
	  "llm": {
	    "profiles": {
	      "default": {
	        "provider": "openai",
	        "model": "gpt-4.1-mini",
	        "prompt": "Translate from {source} to {target}."
	      }
	    }
	  }
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

func TestLoadAllowsOptionalStorageConfig(t *testing.T) {
	path := writeConfigFile(t, `{
	  "locales": {
	    "source": "en-US",
	    "targets": ["fr-FR"]
	  },
	  "buckets": {
	    "json": {
	      "files": [
	        {"from": "lang/{{source}}.json", "to": "lang/{{target}}.json"}
	      ]
	    }
	  },
	  "groups": {
	    "default": {
	      "targets": ["fr-FR"],
	      "buckets": ["json"]
	    }
	  },
	  "llm": {
	    "profiles": {
	      "default": {
	        "provider": "openai",
	        "model": "gpt-4.1-mini",
	        "prompt": "Translate from {source} to {target}."
	      }
	    }
	  },
	  "storage": {
	    "adapter": "poeditor",
	    "config": {"projectID":"123"}
	  }
	}`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load config with storage: %v", err)
	}
	if cfg.Storage == nil || cfg.Storage.Adapter != "poeditor" {
		t.Fatalf("unexpected storage config: %+v", cfg.Storage)
	}
}

func TestLoadRejectsEmptyStorageAdapter(t *testing.T) {
	path := writeConfigFile(t, `{
	  "locales": {
	    "source": "en-US",
	    "targets": ["fr-FR"]
	  },
	  "buckets": {
	    "json": {
	      "files": [
	        {"from": "lang/{{source}}.json", "to": "lang/{{target}}.json"}
	      ]
	    }
	  },
	  "groups": {
	    "default": {
	      "targets": ["fr-FR"],
	      "buckets": ["json"]
	    }
	  },
	  "llm": {
	    "profiles": {
	      "default": {
	        "provider": "openai",
	        "model": "gpt-4.1-mini",
	        "prompt": "Translate from {source} to {target}."
	      }
	    }
	  },
	  "storage": {
	    "adapter": "  ",
	    "config": {"projectID":"123"}
	  }
	}`)

	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "storage.adapter") {
		t.Fatalf("expected storage adapter validation error, got %v", err)
	}
}

func TestLoadAppliesCacheDefaultsWhenOmitted(t *testing.T) {
	path := writeConfigFile(t, `{
	  "locales": {
	    "source": "en-US",
	    "targets": ["fr-FR"]
	  },
	  "buckets": {
	    "json": {
	      "files": [
	        {"from": "lang/{{source}}.json", "to": "lang/{{target}}.json"}
	      ]
	    }
	  },
	  "groups": {
	    "default": {
	      "targets": ["fr-FR"],
	      "buckets": ["json"]
	    }
	  },
	  "llm": {
	    "profiles": {
	      "default": {
	        "provider": "openai",
	        "model": "gpt-4.1-mini",
	        "prompt": "Translate"
	      }
	    }
	  }
	}`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Cache.Enabled {
		t.Fatal("cache should be disabled by default")
	}
	if cfg.Cache.DBPath != DefaultCacheDBPath {
		t.Fatalf("unexpected cache.db_path default: %q", cfg.Cache.DBPath)
	}
	if cfg.Cache.SQLite.MaxOpenConns != DefaultCacheMaxOpen {
		t.Fatalf("unexpected cache.sqlite.max_open_conns default: %d", cfg.Cache.SQLite.MaxOpenConns)
	}
	if cfg.Cache.SQLite.MaxIdleConns != DefaultCacheMaxIdle {
		t.Fatalf("unexpected cache.sqlite.max_idle_conns default: %d", cfg.Cache.SQLite.MaxIdleConns)
	}
	if cfg.Cache.SQLite.ConnMaxLifetime != DefaultCacheMaxLifeSec {
		t.Fatalf("unexpected cache.sqlite.conn_max_lifetime_seconds default: %d", cfg.Cache.SQLite.ConnMaxLifetime)
	}
	if cfg.Cache.L1.MaxItems != DefaultCacheL1MaxItems {
		t.Fatalf("unexpected cache.l1.max_items default: %d", cfg.Cache.L1.MaxItems)
	}
	if cfg.Cache.L2.AutoAcceptThreshold != DefaultCacheL2AutoAcceptThreshold {
		t.Fatalf("unexpected cache.l2.auto_accept_threshold default: %f", cfg.Cache.L2.AutoAcceptThreshold)
	}
	if cfg.Cache.RAG.TopK != DefaultCacheRAGTopK {
		t.Fatalf("unexpected cache.rag.top_k default: %d", cfg.Cache.RAG.TopK)
	}
}

func TestLoadRejectsInvalidCacheConfig(t *testing.T) {
	path := writeConfigFile(t, `{
	  "locales": {
	    "source": "en-US",
	    "targets": ["fr-FR"]
	  },
	  "buckets": {
	    "json": {
	      "files": [
	        {"from": "lang/{{source}}.json", "to": "lang/{{target}}.json"}
	      ]
	    }
	  },
	  "groups": {
	    "default": {
	      "targets": ["fr-FR"],
	      "buckets": ["json"]
	    }
	  },
	  "llm": {
	    "profiles": {
	      "default": {
	        "provider": "openai",
	        "model": "gpt-4.1-mini",
	        "prompt": "Translate"
	      }
	    }
	  },
	  "cache": {
	    "enabled": true,
	    "sqlite": {
	      "max_open_conns": -1
	    }
	  }
	}`)

	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "cache.sqlite.max_open_conns") {
		t.Fatalf("expected cache sqlite validation error, got %v", err)
	}
}

func TestLoadRejectsInvalidCacheL2AutoAcceptThreshold(t *testing.T) {
	path := writeConfigFile(t, `{
	  "locales": {
	    "source": "en-US",
	    "targets": ["fr-FR"]
	  },
	  "buckets": {
	    "json": {
	      "files": [
	        {"from": "lang/{{source}}.json", "to": "lang/{{target}}.json"}
	      ]
	    }
	  },
	  "groups": {
	    "default": {
	      "targets": ["fr-FR"],
	      "buckets": ["json"]
	    }
	  },
	  "llm": {
	    "profiles": {
	      "default": {
	        "provider": "openai",
	        "model": "gpt-4.1-mini",
	        "prompt": "Translate"
	      }
	    }
	  },
	  "cache": {
	    "enabled": true,
	    "l2": {
	      "enabled": true,
	      "auto_accept_threshold": 1.5
	    }
	  }
	}`)

	_, err := Load(path)
	if err == nil || !strings.Contains(err.Error(), "cache.l2.auto_accept_threshold") {
		t.Fatalf("expected cache l2 threshold validation error, got %v", err)
	}
}

func TestLoadPreservesExplicitZeroCacheL2AutoAcceptThreshold(t *testing.T) {
	path := writeConfigFile(t, `{
	  "locales": {
	    "source": "en-US",
	    "targets": ["fr-FR"]
	  },
	  "buckets": {
	    "json": {
	      "files": [
	        {"from": "lang/{{source}}.json", "to": "lang/{{target}}.json"}
	      ]
	    }
	  },
	  "groups": {
	    "default": {
	      "targets": ["fr-FR"],
	      "buckets": ["json"]
	    }
	  },
	  "llm": {
	    "profiles": {
	      "default": {
	        "provider": "openai",
	        "model": "gpt-4.1-mini",
	        "prompt": "Translate"
	      }
	    }
	  },
	  "cache": {
	    "enabled": true,
	    "l2": {
	      "enabled": true,
	      "auto_accept_threshold": 0
	    }
	  }
	}`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Cache.L2.AutoAcceptThreshold != 0 {
		t.Fatalf("cache.l2.auto_accept_threshold=%f, want 0", cfg.Cache.L2.AutoAcceptThreshold)
	}
}

func writeConfigFile(t *testing.T, content string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "config.jsonc")

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	return path
}

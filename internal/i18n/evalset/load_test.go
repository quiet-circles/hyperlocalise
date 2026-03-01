package evalset

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadJSONAndJSONC(t *testing.T) {
	testCases := []struct {
		name        string
		filename    string
		content     string
		errContains string
	}{
		{
			name:     "decode valid jsonc dataset",
			filename: "evalset.jsonc",
			content: `{
			  // schema metadata for tooling
			  "version": "v1",
			  "metadata": {
			    "owner": "l10n",
			    "domain": "checkout"
			  },
			  "cases": [
			    {
			      "id": "ui.pay.cta",
			      "source": "Pay now",
			      "targetLocale": "fr-FR",
			      "context": "Primary CTA on checkout page",
			      "reference": "Payer maintenant",
			      "tags": ["ui", "short"],
			      "bucket": "checkout",
			      "group": "critical"
			    }
			  ]
			}`,
		},
		{
			name:     "reject unknown fields",
			filename: "evalset.json",
			content: `{
			  "cases": [
			    {
			      "id": "a",
			      "source": "Hello",
			      "targetLocale": "es-ES",
			      "unknown": true
			    }
			  ]
			}`,
			errContains: "unknown field",
		},
		{
			name:     "validate non-empty cases",
			filename: "evalset.json",
			content: `{
			  "cases": []
			}`,
			errContains: "cases: must not be empty",
		},
		{
			name:     "validate required source",
			filename: "evalset.json",
			content: `{
			  "cases": [
			    {
			      "id": "a",
			      "source": "",
			      "targetLocale": "de-DE"
			    }
			  ]
			}`,
			errContains: "source: must not be empty",
		},
		{
			name:     "validate required target locale",
			filename: "evalset.json",
			content: `{
			  "cases": [
			    {
			      "id": "a",
			      "source": "Settings",
			      "targetLocale": ""
			    }
			  ]
			}`,
			errContains: "targetLocale: must not be empty",
		},
		{
			name:     "validate empty id",
			filename: "evalset.json",
			content: `{
			  "cases": [
			    {
			      "id": "   ",
			      "source": "Settings",
			      "targetLocale": "fr-FR"
			    }
			  ]
			}`,
			errContains: "id: must not be empty",
		},
		{
			name:     "validate duplicate id with whitespace",
			filename: "evalset.json",
			content: `{
			  "cases": [
			    {
			      "id": "dup",
			      "source": "One",
			      "targetLocale": "ja-JP"
			    },
			    {
			      "id": "dup ",
			      "source": "Two",
			      "targetLocale": "ja-JP"
			    }
			  ]
			}`,
			errContains: "duplicate id",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), tc.filename)
			if err := os.WriteFile(path, []byte(tc.content), 0o644); err != nil {
				t.Fatalf("write evalset: %v", err)
			}

			dataset, err := Load(path)
			if tc.errContains != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.errContains)
				}

				if !strings.Contains(err.Error(), tc.errContains) {
					t.Fatalf("expected error containing %q, got %q", tc.errContains, err.Error())
				}

				return
			}

			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}

			if dataset == nil {
				t.Fatalf("Load() dataset is nil")
			}

			if len(dataset.Cases) != 1 {
				t.Fatalf("expected 1 case, got %d", len(dataset.Cases))
			}

			if dataset.Cases[0].ID != "ui.pay.cta" {
				t.Fatalf("expected case id ui.pay.cta, got %q", dataset.Cases[0].ID)
			}
		})
	}
}

func TestLoadCSV(t *testing.T) {
	testCases := []struct {
		name        string
		content     string
		errContains string
	}{
		{
			name: "decode valid csv dataset",
			content: strings.Join([]string{
				"id,source,targetLocale,context,reference,tags,bucket,group",
				`ui.pay.cta,Pay now,fr-FR,Primary CTA,Payer maintenant,"ui;short",checkout,critical`,
			}, "\n"),
		},
		{
			name: "reject unknown csv header",
			content: strings.Join([]string{
				"id,source,targetLocale,unknown",
				"a,Hello,es-ES,x",
			}, "\n"),
			errContains: "unknown header",
		},
		{
			name: "reject missing required csv header",
			content: strings.Join([]string{
				"id,source",
				"a,Hello",
			}, "\n"),
			errContains: "missing required header \"targetLocale\"",
		},
		{
			name: "validate required source from csv",
			content: strings.Join([]string{
				"id,source,targetLocale",
				"a,,de-DE",
			}, "\n"),
			errContains: "source: must not be empty",
		},
		{
			name: "validate required target locale from csv",
			content: strings.Join([]string{
				"id,source,targetLocale",
				"a,Settings,",
			}, "\n"),
			errContains: "targetLocale: must not be empty",
		},
		{
			name: "validate unique id from csv",
			content: strings.Join([]string{
				"id,source,targetLocale",
				"dup,One,ja-JP",
				"dup,Two,ja-JP",
			}, "\n"),
			errContains: "duplicate id",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "evalset.csv")
			if err := os.WriteFile(path, []byte(tc.content), 0o644); err != nil {
				t.Fatalf("write evalset: %v", err)
			}

			dataset, err := Load(path)
			if tc.errContains != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.errContains)
				}

				if !strings.Contains(err.Error(), tc.errContains) {
					t.Fatalf("expected error containing %q, got %q", tc.errContains, err.Error())
				}

				return
			}

			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}

			if dataset == nil {
				t.Fatalf("Load() dataset is nil")
			}

			if len(dataset.Cases) != 1 {
				t.Fatalf("expected 1 case, got %d", len(dataset.Cases))
			}

			if dataset.Cases[0].ID != "ui.pay.cta" {
				t.Fatalf("expected case id ui.pay.cta, got %q", dataset.Cases[0].ID)
			}

			tags := dataset.Cases[0].Tags
			if len(tags) != 2 || tags[0] != "ui" || tags[1] != "short" {
				t.Fatalf("expected tags [ui short], got %#v", tags)
			}
		})
	}
}

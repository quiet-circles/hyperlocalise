package runsvc

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/quiet-circles/hyperlocalise/internal/i18n/translationfileparser"
)

func TestMarshalTargetFileUnsupportedExtension(t *testing.T) {
	svc := newTestService()
	_, _, err := svc.marshalTargetFile("/tmp/out.txt", "/tmp/src.txt", "en", "fr", map[string]string{"hello": "Bonjour"}, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "unsupported target file extension") {
		t.Fatalf("expected unsupported extension error, got %v", err)
	}
}

func TestMarshalTemplateBasedTargetUnsupportedExtension(t *testing.T) {
	svc := newTestService()
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case "/tmp/target.bin":
			return []byte("template"), nil
		case "/tmp/source.bin":
			return []byte("source"), nil
		default:
			return nil, os.ErrNotExist
		}
	}
	_, _, err := svc.marshalTemplateBasedTarget(".bin", "/tmp/target.bin", "/tmp/source.bin", "en", "fr", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "unsupported target file extension") {
		t.Fatalf("expected unsupported extension error, got %v", err)
	}
}

func TestLoadTemplateFallback(t *testing.T) {
	svc := newTestService()
	t.Run("prefer target", func(t *testing.T) {
		svc.readFile = func(path string) ([]byte, error) {
			if path == "/tmp/target.json" {
				return []byte("target"), nil
			}
			return nil, errors.New("unexpected path")
		}
		content, err := svc.loadTemplateFallback("/tmp/target.json", "/tmp/source.json")
		if err != nil {
			t.Fatalf("load target template: %v", err)
		}
		if string(content) != "target" {
			t.Fatalf("expected target content, got %q", content)
		}
	})

	t.Run("fallback source", func(t *testing.T) {
		svc.readFile = func(path string) ([]byte, error) {
			switch path {
			case "/tmp/target.json":
				return nil, os.ErrNotExist
			case "/tmp/source.json":
				return []byte("source"), nil
			default:
				return nil, errors.New("unexpected path")
			}
		}
		content, err := svc.loadTemplateFallback("/tmp/target.json", "/tmp/source.json")
		if err != nil {
			t.Fatalf("load source fallback: %v", err)
		}
		if string(content) != "source" {
			t.Fatalf("expected source content, got %q", content)
		}
	})

	t.Run("target read error", func(t *testing.T) {
		svc.readFile = func(path string) ([]byte, error) {
			if path == "/tmp/target.json" {
				return nil, errors.New("boom")
			}
			return nil, errors.New("unexpected path")
		}
		_, err := svc.loadTemplateFallback("/tmp/target.json", "/tmp/source.json")
		if err == nil || !strings.Contains(err.Error(), "read target file") {
			t.Fatalf("expected target read error, got %v", err)
		}
	})
}

func TestHasExactKeySet(t *testing.T) {
	if !hasExactKeySet(map[string]string{"a": "1", "b": "2"}, map[string]string{"a": "x", "b": "y"}) {
		t.Fatalf("expected exact key set")
	}
	if hasExactKeySet(map[string]string{"a": "1"}, map[string]string{"a": "x", "b": "y"}) {
		t.Fatalf("did not expect exact key set")
	}
	if hasExactKeySet(map[string]string{"a": "1"}, map[string]string{"b": "y"}) {
		t.Fatalf("did not expect exact key set with different keys")
	}
}

func TestMarkdownRenderWarnings(t *testing.T) {
	diags := translationfileparser.MarkdownRenderDiagnostics{SourceFallbackKeys: []string{"k3", "k1", "k2"}}
	warnings := markdownRenderWarnings("/tmp/doc.md", diags)
	if len(warnings) != 1 {
		t.Fatalf("expected warning")
	}
	if !strings.Contains(warnings[0], "keys: k1, k2, k3") {
		t.Fatalf("unexpected warning order/message: %q", warnings[0])
	}

	diags = translationfileparser.MarkdownRenderDiagnostics{SourceFallbackKeys: []string{"k4", "k2", "k1", "k3"}}
	warnings = markdownRenderWarnings("/tmp/doc.md", diags)
	if !strings.Contains(warnings[0], "first keys: k1, k2, k3") {
		t.Fatalf("unexpected truncated warning: %q", warnings[0])
	}

	if got := markdownRenderWarnings("/tmp/doc.md", translationfileparser.MarkdownRenderDiagnostics{}); got != nil {
		t.Fatalf("expected nil warnings when no diagnostics, got %#v", got)
	}
}

func TestMarshalMarkdownTargetReadErrors(t *testing.T) {
	svc := newTestService()
	sourcePath := filepath.Join(t.TempDir(), "source.md")
	if err := os.WriteFile(sourcePath, []byte("# Title"), 0o644); err != nil {
		t.Fatalf("write source markdown: %v", err)
	}
	targetPath := filepath.Join(t.TempDir(), "target.md")

	svc.readFile = os.ReadFile
	content, warnings, err := svc.marshalMarkdownTarget(targetPath, sourcePath, map[string]string{"title": "Titre"})
	if err != nil {
		t.Fatalf("marshal markdown missing target: %v", err)
	}
	if len(content) == 0 {
		t.Fatalf("expected markdown content")
	}
	if warnings != nil {
		t.Fatalf("expected no warnings for simple markdown, got %#v", warnings)
	}

	svc.readFile = func(path string) ([]byte, error) {
		if path == sourcePath {
			return nil, os.ErrNotExist
		}
		return nil, errors.New("unexpected")
	}
	_, _, err = svc.marshalMarkdownTarget(targetPath, sourcePath, map[string]string{"title": "Titre"})
	if err == nil || !strings.Contains(err.Error(), "read template source") {
		t.Fatalf("expected source read error, got %v", err)
	}
}

func TestMarshalSourceTemplateTargetFallsBackToSourceOnKeyMismatch(t *testing.T) {
	sourcePath := filepath.Join(t.TempDir(), "source.po")
	targetPath := filepath.Join(t.TempDir(), "target.po")
	source := "msgid \"hello\"\nmsgstr \"Hello\"\n"
	target := "msgid \"other\"\nmsgstr \"Autre\"\n"
	if err := os.WriteFile(sourcePath, []byte(source), 0o644); err != nil {
		t.Fatalf("write source po: %v", err)
	}
	if err := os.WriteFile(targetPath, []byte(target), 0o644); err != nil {
		t.Fatalf("write target po: %v", err)
	}

	svc := newTestService()
	svc.readFile = os.ReadFile
	content, err := svc.marshalSourceTemplateTarget(".po", targetPath, sourcePath, "en", "fr", map[string]string{"hello": "Bonjour"})
	if err != nil {
		t.Fatalf("marshal source template target: %v", err)
	}
	text := string(content)
	if !strings.Contains(text, "msgid \"hello\"") || !strings.Contains(text, "msgstr \"Bonjour\"") {
		t.Fatalf("expected source-template fallback content, got %q", text)
	}
}

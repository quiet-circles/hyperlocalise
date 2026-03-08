package runsvc

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestFlushOutputForTargetPrunesAndMerges(t *testing.T) {
	targetPath := filepath.Join(t.TempDir(), "fr.json")
	sourcePath := filepath.Join(t.TempDir(), "en.json")
	if err := os.WriteFile(targetPath, []byte(`{"a":"A","b":"B"}`), 0o644); err != nil {
		t.Fatalf("write target file: %v", err)
	}
	if err := os.WriteFile(sourcePath, []byte(`{"a":"A","b":"B","c":"C"}`), 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	svc := newTestService()
	svc.readFile = os.ReadFile

	var written []byte
	svc.writeFile = func(path string, content []byte) error {
		if path != targetPath {
			t.Fatalf("unexpected write path: %s", path)
		}
		written = append([]byte(nil), content...)
		return nil
	}

	warnings, err := svc.flushOutputForTarget(targetPath, stagedOutput{
		entries:      map[string]string{"a": "AA", "c": "CC"},
		sourcePath:   sourcePath,
		targetLocale: "fr",
	}, map[string]struct{}{"a": {}, "c": {}})
	if err != nil {
		t.Fatalf("flush output target: %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("unexpected warnings: %#v", warnings)
	}

	var payload map[string]string
	if err := json.Unmarshal(written, &payload); err != nil {
		t.Fatalf("decode written payload: %v", err)
	}
	want := map[string]string{"a": "AA", "c": "CC"}
	if !reflect.DeepEqual(payload, want) {
		t.Fatalf("written payload mismatch\nwant: %#v\n got: %#v", want, payload)
	}
}

func TestFlushOutputsSortedUniqueTargets(t *testing.T) {
	dir := t.TempDir()
	paths := []string{
		filepath.Join(dir, "a.json"),
		filepath.Join(dir, "b.json"),
		filepath.Join(dir, "c.json"),
	}
	for _, path := range paths {
		if err := os.WriteFile(path, []byte(`{"k":"v"}`), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}

	svc := newTestService()
	svc.readFile = os.ReadFile
	order := make([]string, 0, 3)
	svc.writeFile = func(path string, _ []byte) error {
		order = append(order, path)
		return nil
	}

	staged := map[string]stagedOutput{
		paths[1]: {entries: map[string]string{"k": "vb"}, targetLocale: "fr"},
		paths[0]: {entries: map[string]string{"k": "va"}, targetLocale: "fr"},
	}
	prune := map[string]map[string]struct{}{
		paths[2]: {"k": {}},
		paths[1]: {"k": {}},
	}

	_, err := svc.flushOutputs(staged, prune, nil)
	if err != nil {
		t.Fatalf("flush outputs: %v", err)
	}
	wantOrder := []string{paths[0], paths[1], paths[2]}
	if !reflect.DeepEqual(order, wantOrder) {
		t.Fatalf("flush order mismatch\nwant: %#v\n got: %#v", wantOrder, order)
	}
}

func TestLoadExistingTargetWithWarnings(t *testing.T) {
	svc := newTestService()

	t.Run("missing file", func(t *testing.T) {
		svc.readFile = func(_ string) ([]byte, error) { return nil, os.ErrNotExist }
		entries, warnings, err := svc.loadExistingTargetWithWarnings("/tmp/missing.json", "fr")
		if err != nil {
			t.Fatalf("missing file should not error: %v", err)
		}
		if len(entries) != 0 || warnings != nil {
			t.Fatalf("unexpected outputs for missing file: entries=%#v warnings=%#v", entries, warnings)
		}
	})

	t.Run("malformed json warning", func(t *testing.T) {
		svc.readFile = func(_ string) ([]byte, error) { return []byte("{"), nil }
		entries, warnings, err := svc.loadExistingTargetWithWarnings("/tmp/target.json", "fr")
		if err != nil {
			t.Fatalf("malformed json should warn not fail: %v", err)
		}
		if len(entries) != 0 {
			t.Fatalf("expected empty recovered entries, got %#v", entries)
		}
		if len(warnings) != 1 {
			t.Fatalf("expected one warning, got %#v", warnings)
		}
	})

	t.Run("non-json parse error", func(t *testing.T) {
		svc.readFile = func(_ string) ([]byte, error) { return []byte("not-supported"), nil }
		_, _, err := svc.loadExistingTargetWithWarnings("/tmp/target.txt", "fr")
		if err == nil {
			t.Fatalf("expected parse error for non-json unsupported format")
		}
	})

	t.Run("read error", func(t *testing.T) {
		svc.readFile = func(_ string) ([]byte, error) { return nil, errors.New("boom") }
		_, _, err := svc.loadExistingTargetWithWarnings("/tmp/target.json", "fr")
		if err == nil {
			t.Fatalf("expected read error")
		}
	})
}

func TestParseExistingTargetEntriesCSV(t *testing.T) {
	parser := newTestService().newParser()
	entries, err := parseExistingTargetEntries("/tmp/target.csv", []byte("id,en,fr\nhello,Hello,Bonjour\n"), "fr", parser)
	if err != nil {
		t.Fatalf("parse existing csv: %v", err)
	}
	if got := entries["hello"]; got != "Bonjour" {
		t.Fatalf("csv locale parse mismatch: %q", got)
	}
}

func TestFlushOutputForTargetWrapsWriteError(t *testing.T) {
	targetPath := filepath.Join(t.TempDir(), "fr.json")
	sourcePath := filepath.Join(t.TempDir(), "en.json")
	if err := os.WriteFile(targetPath, []byte(`{"hello":"Bonjour"}`), 0o644); err != nil {
		t.Fatalf("write target file: %v", err)
	}
	if err := os.WriteFile(sourcePath, []byte(`{"hello":"Hello"}`), 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	svc := newTestService()
	svc.readFile = os.ReadFile
	svc.writeFile = func(_ string, _ []byte) error { return errors.New("disk full") }

	_, err := svc.flushOutputForTarget(targetPath, stagedOutput{
		entries:      map[string]string{"hello": "Salut"},
		sourcePath:   sourcePath,
		targetLocale: "fr",
	}, nil)
	if err == nil {
		t.Fatalf("expected write error")
	}
	if !strings.Contains(err.Error(), "flush outputs: write") || !strings.Contains(err.Error(), "disk full") {
		t.Fatalf("expected wrapped write error, got %v", err)
	}
}

func TestFlushOutputsPruneOnlyMissingTargetAfterScan(t *testing.T) {
	targetPath := filepath.Join(t.TempDir(), "target.json")
	svc := newTestService()
	svc.readFile = func(path string) ([]byte, error) {
		// Simulate target disappearing between prune scan and flush.
		if path == targetPath {
			return nil, os.ErrNotExist
		}
		if path == "" {
			return nil, os.ErrNotExist
		}
		return nil, errors.New("unexpected path")
	}
	svc.writeFile = func(_ string, _ []byte) error { return nil }

	_, err := svc.flushOutputs(nil, map[string]map[string]struct{}{
		targetPath: {"hello": {}},
	}, nil)
	if err == nil {
		t.Fatalf("expected error for missing target with no source template")
	}
	if !strings.Contains(err.Error(), "read template source") {
		t.Fatalf("expected source template read error, got %v", err)
	}
}

func TestFlushOutputsPruneOnlyUsesPlannedMetadata(t *testing.T) {
	targetPath := filepath.Join(t.TempDir(), "fr.json")
	sourcePath := filepath.Join(t.TempDir(), "en.json")
	if err := os.WriteFile(targetPath, []byte(`{"hello":"Bonjour","legacy":"Ancien"}`), 0o644); err != nil {
		t.Fatalf("write target file: %v", err)
	}
	if err := os.WriteFile(sourcePath, []byte(`{"hello":"Hello"}`), 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	svc := newTestService()
	svc.readFile = os.ReadFile

	var written []byte
	svc.writeFile = func(path string, content []byte) error {
		if path != targetPath {
			t.Fatalf("unexpected write path: %s", path)
		}
		written = append([]byte(nil), content...)
		return nil
	}

	_, err := svc.flushOutputs(nil, map[string]map[string]struct{}{
		targetPath: {"hello": {}},
	}, map[string]stagedOutput{
		targetPath: {entries: map[string]string{}, sourcePath: sourcePath, sourceLocale: "en", targetLocale: "fr"},
	})
	if err != nil {
		t.Fatalf("flush outputs prune-only with metadata: %v", err)
	}
	if strings.Contains(string(written), "legacy") {
		t.Fatalf("expected legacy key pruned, got %s", written)
	}
}

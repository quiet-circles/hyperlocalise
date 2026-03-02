package runsvc

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/quiet-circles/hyperlocalise/internal/config"
	"github.com/quiet-circles/hyperlocalise/internal/i18n/lockfile"
	"github.com/quiet-circles/hyperlocalise/internal/i18n/translationfileparser"
	"github.com/quiet-circles/hyperlocalise/internal/i18n/translator"
)

func TestRunUsesConfiguredWorkersWhenProvided(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.json"
	targetPath := "/tmp/out.json"
	svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
		cfg := testConfig(sourcePath, targetPath)
		return &cfg, nil
	}
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return []byte(`{"a":"A","b":"B","c":"C"}`), nil
		case targetPath:
			return []byte(`{}`), nil
		default:
			return nil, filepath.ErrBadPattern
		}
	}
	svc.numCPU = func() int { return 1 }

	var mu sync.Mutex
	active := 0
	maxActive := 0
	svc.translate = func(_ context.Context, req translator.Request) (string, error) {
		mu.Lock()
		active++
		if active > maxActive {
			maxActive = active
		}
		mu.Unlock()

		time.Sleep(20 * time.Millisecond)

		mu.Lock()
		active--
		mu.Unlock()
		return strings.ToUpper(req.Source), nil
	}

	_, err := svc.Run(context.Background(), Input{Workers: 3})
	if err != nil {
		t.Fatalf("run execution: %v", err)
	}
	if maxActive < 2 {
		t.Fatalf("expected parallel execution with explicit workers, max active=%d", maxActive)
	}
}

func TestRunDefaultsWorkersToNumCPUWhenUnset(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.json"
	targetPath := "/tmp/out.json"
	svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
		cfg := testConfig(sourcePath, targetPath)
		return &cfg, nil
	}
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return []byte(`{"a":"A","b":"B","c":"C"}`), nil
		case targetPath:
			return []byte(`{}`), nil
		default:
			return nil, filepath.ErrBadPattern
		}
	}
	svc.numCPU = func() int { return 1 }

	var mu sync.Mutex
	active := 0
	maxActive := 0
	svc.translate = func(_ context.Context, req translator.Request) (string, error) {
		mu.Lock()
		active++
		if active > maxActive {
			maxActive = active
		}
		mu.Unlock()

		time.Sleep(20 * time.Millisecond)

		mu.Lock()
		active--
		mu.Unlock()
		return strings.ToUpper(req.Source), nil
	}

	_, err := svc.Run(context.Background(), Input{})
	if err != nil {
		t.Fatalf("run execution: %v", err)
	}
	if maxActive != 1 {
		t.Fatalf("expected single worker from numCPU default, max active=%d", maxActive)
	}
}

func TestRunFailsWhenSourceFileMissing(t *testing.T) {
	svc := newTestService()
	svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
		cfg := testConfig("/tmp/missing.json", "/tmp/out.json")
		return &cfg, nil
	}
	svc.readFile = func(_ string) ([]byte, error) {
		return nil, os.ErrNotExist
	}

	_, err := svc.Run(context.Background(), Input{})
	if err == nil {
		t.Fatalf("expected planning error")
	}
}

func TestRunFailsOnUnsupportedSourceFormat(t *testing.T) {
	svc := newTestService()
	svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
		cfg := testConfig("/tmp/source.yaml", "/tmp/out.json")
		return &cfg, nil
	}
	svc.readFile = func(_ string) ([]byte, error) {
		return []byte("hello: world"), nil
	}

	_, err := svc.Run(context.Background(), Input{})
	if err == nil || !strings.Contains(err.Error(), "unsupported file extension") {
		t.Fatalf("expected unsupported extension error, got %v", err)
	}
}

func TestResolveProfileRulePriorityWithDefaultFallback(t *testing.T) {
	cfg := testConfig("/tmp/source.json", "/tmp/out.json")
	cfg.LLM.Profiles["fast"] = config.LLMProfile{Provider: "openai", Model: "fast-model", Prompt: "fast {{input}}"}
	cfg.LLM.Rules = []config.LLMRule{
		{Priority: 1, Group: "default", Profile: "default"},
		{Priority: 100, Group: "default", Profile: "fast"},
	}

	profileName, profile, err := resolveProfile(&cfg, "default")
	if err != nil {
		t.Fatalf("resolve profile: %v", err)
	}
	if profileName != "fast" || profile.Model != "fast-model" {
		t.Fatalf("unexpected profile resolved: name=%s model=%s", profileName, profile.Model)
	}

	profileName, profile, err = resolveProfile(&cfg, "unknown")
	if err != nil {
		t.Fatalf("resolve fallback profile: %v", err)
	}
	if profileName != "default" || profile.Model != "gpt-4.1-mini" {
		t.Fatalf("expected default fallback profile, got name=%s model=%s", profileName, profile.Model)
	}
}

func TestRunAppliesLockFilterByTargetAndEntry(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.json"
	targetPath := "/tmp/out.json"
	svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
		cfg := testConfig(sourcePath, targetPath)
		return &cfg, nil
	}
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return []byte(`{"a":"A","b":"B"}`), nil
		default:
			return nil, filepath.ErrBadPattern
		}
	}
	svc.loadLock = func(_ string) (*lockfile.File, error) {
		return &lockfile.File{RunCompleted: map[string]lockfile.RunCompletion{taskIdentity(targetPath, "a"): {CompletedAt: time.Now(), SourceHash: hashSourceText("A")}}}, nil
	}

	report, err := svc.Run(context.Background(), Input{DryRun: true})
	if err != nil {
		t.Fatalf("run dry-run: %v", err)
	}

	if report.PlannedTotal != 2 || report.SkippedByLock != 1 || report.ExecutableTotal != 1 {
		t.Fatalf("unexpected plan totals: %+v", report)
	}
	if len(report.Executable) != 1 || report.Executable[0].EntryKey != "b" {
		t.Fatalf("unexpected executable tasks: %+v", report.Executable)
	}
}

func TestRunDoesNotSkipWhenSourceTextChanges(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.json"
	targetPath := "/tmp/out.json"
	svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
		cfg := testConfig(sourcePath, targetPath)
		return &cfg, nil
	}
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return []byte(`{"hello":"Hello World"}`), nil
		default:
			return nil, filepath.ErrBadPattern
		}
	}
	svc.loadLock = func(_ string) (*lockfile.File, error) {
		return &lockfile.File{RunCompleted: map[string]lockfile.RunCompletion{taskIdentity(targetPath, "hello"): {CompletedAt: time.Now(), SourceHash: hashSourceText("Hello")}}}, nil
	}

	report, err := svc.Run(context.Background(), Input{DryRun: true})
	if err != nil {
		t.Fatalf("run dry-run: %v", err)
	}

	if report.SkippedByLock != 0 || report.ExecutableTotal != 1 {
		t.Fatalf("expected changed source to be executable, got %+v", report)
	}
}

func TestRunDryRunSkipsWrites(t *testing.T) {
	writeCount := 0
	lockSaveCount := 0

	svc := newTestService()
	svc.writeFile = func(_ string, _ []byte) error {
		writeCount++
		return nil
	}
	svc.saveLock = func(_ string, _ lockfile.File) error {
		lockSaveCount++
		return nil
	}

	_, err := svc.Run(context.Background(), Input{DryRun: true})
	if err != nil {
		t.Fatalf("run dry-run: %v", err)
	}
	if writeCount != 0 {
		t.Fatalf("expected no writes in dry-run, got %d", writeCount)
	}
	if lockSaveCount != 0 {
		t.Fatalf("expected no lock writes in dry-run, got %d", lockSaveCount)
	}
}

func TestRunContinueOnErrorReturnsPartialFailureReport(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.json"
	targetPath := "/tmp/out.json"
	svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
		cfg := testConfig(sourcePath, targetPath)
		return &cfg, nil
	}
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return []byte(`{"ok":"hello","bad":"boom"}`), nil
		case targetPath:
			return []byte(`{"existing":"v"}`), nil
		default:
			return nil, filepath.ErrBadPattern
		}
	}
	svc.translate = func(_ context.Context, req translator.Request) (string, error) {
		if req.Source == "boom" {
			return "", errors.New("translation failed")
		}
		return strings.ToUpper(req.Source), nil
	}

	var written []byte
	svc.writeFile = func(path string, content []byte) error {
		if path != targetPath {
			t.Fatalf("unexpected write path %q", path)
		}
		written = append([]byte(nil), content...)
		return nil
	}

	report, err := svc.Run(context.Background(), Input{})
	if err != nil {
		t.Fatalf("run execution: %v", err)
	}
	if report.Succeeded != 1 || report.Failed != 1 {
		t.Fatalf("unexpected execution totals: %+v", report)
	}
	if len(report.Failures) != 1 || report.Failures[0].EntryKey != "bad" {
		t.Fatalf("unexpected failures: %+v", report.Failures)
	}

	var payload map[string]any
	if err := json.Unmarshal(written, &payload); err != nil {
		t.Fatalf("decode written payload: %v", err)
	}
	if payload["ok"] != "HELLO" {
		t.Fatalf("expected translated key to be written, got %+v", payload)
	}
	if payload["existing"] != "v" {
		t.Fatalf("expected existing key preserved, got %+v", payload)
	}
}

func TestRunLockWriterPersistsEachSuccess(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.json"
	targetPath := "/tmp/out.json"
	svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
		cfg := testConfig(sourcePath, targetPath)
		return &cfg, nil
	}
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return []byte(`{"a":"A","b":"B","c":"C"}`), nil
		case targetPath:
			return []byte(`{}`), nil
		default:
			return nil, filepath.ErrBadPattern
		}
	}

	lockWrites := 0
	seenSizes := []int{}
	svc.saveLock = func(_ string, f lockfile.File) error {
		lockWrites++
		seenSizes = append(seenSizes, len(f.RunCompleted))
		for identity, completion := range f.RunCompleted {
			if completion.SourceHash == "" {
				t.Fatalf("expected source hash persisted for %s", identity)
			}
		}
		return nil
	}

	report, err := svc.Run(context.Background(), Input{})
	if err != nil {
		t.Fatalf("run execution: %v", err)
	}
	if report.Succeeded != 3 || report.PersistedToLock != 3 {
		t.Fatalf("unexpected lock persistence totals: %+v", report)
	}
	if lockWrites != 3 {
		t.Fatalf("expected one lock write per success, got %d", lockWrites)
	}
	if seenSizes[len(seenSizes)-1] != 3 {
		t.Fatalf("expected final lock map size 3, got %v", seenSizes)
	}
}

func TestRunWritesMarkdownUsingSourceTemplateWhenTargetMissing(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.md"
	targetPath := "/tmp/out.md"
	source := "---\ntitle: Welcome\n---\n\n# Heading\n\nHello `code` and [docs](https://example.com).\n\n```js\nconsole.log('x')\n```\n"

	svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
		cfg := testConfig(sourcePath, targetPath)
		return &cfg, nil
	}
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return []byte(source), nil
		default:
			return nil, os.ErrNotExist
		}
	}
	svc.translate = func(_ context.Context, req translator.Request) (string, error) {
		return "FR(" + req.Source + ")", nil
	}

	var written []byte
	svc.writeFile = func(path string, content []byte) error {
		if path != targetPath {
			t.Fatalf("unexpected write path %q", path)
		}
		written = append([]byte(nil), content...)
		return nil
	}

	_, err := svc.Run(context.Background(), Input{})
	if err != nil {
		t.Fatalf("run execution: %v", err)
	}

	out := string(written)
	if !strings.Contains(out, "title: Welcome") {
		t.Fatalf("expected frontmatter unchanged, got %q", out)
	}
	if !strings.Contains(out, "```js") || !strings.Contains(out, "console.log('x')") {
		t.Fatalf("expected code fence preserved, got %q", out)
	}
	if !strings.Contains(out, "https://example.com") {
		t.Fatalf("expected link destination preserved, got %q", out)
	}
	if !strings.Contains(out, "FR( Heading") || !strings.Contains(out, "FR(Hello )") {
		t.Fatalf("expected markdown text translated, got %q", out)
	}
}

func TestRunWritesMDXUsingSourceTemplateWhenTargetMissing(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.mdx"
	targetPath := "/tmp/out.mdx"
	source := "---\ntitle: Welcome\n---\n\nimport Tabs from '@theme/Tabs'\n\n<Tabs defaultValue=\"first\">\n  <Tab value=\"first\" label=\"First\">Run command.</Tab>\n</Tabs>\n"

	svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
		cfg := testConfig(sourcePath, targetPath)
		return &cfg, nil
	}
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return []byte(source), nil
		default:
			return nil, os.ErrNotExist
		}
	}
	svc.translate = func(_ context.Context, req translator.Request) (string, error) {
		return "FR(" + req.Source + ")", nil
	}

	var written []byte
	svc.writeFile = func(path string, content []byte) error {
		if path != targetPath {
			t.Fatalf("unexpected write path %q", path)
		}
		written = append([]byte(nil), content...)
		return nil
	}

	_, err := svc.Run(context.Background(), Input{})
	if err != nil {
		t.Fatalf("run execution: %v", err)
	}

	out := string(written)
	if !strings.Contains(out, "import Tabs") {
		t.Fatalf("expected import statement preserved, got %q", out)
	}
	if !strings.Contains(out, "defaultValue=\"first\"") || !strings.Contains(out, "label=\"First\"") {
		t.Fatalf("expected component attributes preserved, got %q", out)
	}
	if !strings.Contains(out, "FR(Run command.)") {
		t.Fatalf("expected prose translated, got %q", out)
	}
}

func TestRunWritesAppleStringsUsingSourceTemplateWhenTargetMissing(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.strings"
	targetPath := "/tmp/out.strings"
	source := `/* Greeting */
"hello" = "Hello";
"multiline" = "First\nSecond";
`

	svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
		cfg := testConfig(sourcePath, targetPath)
		return &cfg, nil
	}
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return []byte(source), nil
		default:
			return nil, os.ErrNotExist
		}
	}
	svc.translate = func(_ context.Context, req translator.Request) (string, error) {
		if req.Source == "First\nSecond" {
			return "Premier\nDeuxieme", nil
		}
		return "FR(" + req.Source + ")", nil
	}

	var written []byte
	svc.writeFile = func(path string, content []byte) error {
		if path != targetPath {
			t.Fatalf("unexpected write path %q", path)
		}
		written = append([]byte(nil), content...)
		return nil
	}

	_, err := svc.Run(context.Background(), Input{})
	if err != nil {
		t.Fatalf("run execution: %v", err)
	}

	out := string(written)
	if !strings.Contains(out, "/* Greeting */") {
		t.Fatalf("expected comment preserved, got %q", out)
	}
	if !strings.Contains(out, `"hello" = "FR(Hello)";`) {
		t.Fatalf("expected greeting translated, got %q", out)
	}
	if !strings.Contains(out, `"multiline" = "Premier\nDeuxieme";`) {
		t.Fatalf("expected multiline translation escaped, got %q", out)
	}
}

func TestRunWritesAppleStringsdictUsingSourceTemplateWhenTargetMissing(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.stringsdict"
	targetPath := "/tmp/out.stringsdict"
	source := `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0">
<dict>
  <key>item_count</key>
  <dict>
    <key>NSStringLocalizedFormatKey</key>
    <string>%#@items@</string>
    <key>items</key>
    <dict>
      <key>one</key>
      <string>%d item</string>
      <key>other</key>
      <string>%d items</string>
    </dict>
  </dict>
</dict>
</plist>
`

	svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
		cfg := testConfig(sourcePath, targetPath)
		return &cfg, nil
	}
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return []byte(source), nil
		default:
			return nil, os.ErrNotExist
		}
	}
	svc.translate = func(_ context.Context, req translator.Request) (string, error) {
		switch req.Source {
		case "%d item":
			return "%d article", nil
		case "%d items":
			return "%d articles", nil
		default:
			return req.Source, nil
		}
	}

	var written []byte
	svc.writeFile = func(path string, content []byte) error {
		if path != targetPath {
			t.Fatalf("unexpected write path %q", path)
		}
		written = append([]byte(nil), content...)
		return nil
	}

	_, err := svc.Run(context.Background(), Input{})
	if err != nil {
		t.Fatalf("run execution: %v", err)
	}

	out := string(written)
	if !strings.Contains(out, "<string>%#@items@</string>") {
		t.Fatalf("expected format placeholder preserved, got %q", out)
	}
	if !strings.Contains(out, "<string>%d article</string>") {
		t.Fatalf("expected one plural category translated, got %q", out)
	}
	if !strings.Contains(out, "<string>%d articles</string>") {
		t.Fatalf("expected other plural category translated, got %q", out)
	}
}

func TestRunWritesCSVUsingSourceTemplateWhenTargetMissing(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.csv"
	targetPath := "/tmp/out.csv"
	source := "key,source,target\nhello,Hello,Hello\n"

	svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
		cfg := testConfig(sourcePath, targetPath)
		return &cfg, nil
	}
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return []byte(source), nil
		default:
			return nil, os.ErrNotExist
		}
	}
	svc.translate = func(_ context.Context, req translator.Request) (string, error) {
		return "FR(" + req.Source + ")", nil
	}

	var written []byte
	svc.writeFile = func(path string, content []byte) error {
		if path != targetPath {
			t.Fatalf("unexpected write path %q", path)
		}
		written = append([]byte(nil), content...)
		return nil
	}

	_, err := svc.Run(context.Background(), Input{})
	if err != nil {
		t.Fatalf("run execution: %v", err)
	}

	out := string(written)
	if !strings.Contains(out, "key,source,target") {
		t.Fatalf("expected csv headers preserved, got %q", out)
	}
	if !strings.Contains(out, "hello,Hello,FR(Hello)") {
		t.Fatalf("expected csv translation written to target column, got %q", out)
	}
}

func TestRunReturnsFatalErrorWhenLockWriteFails(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.json"
	targetPath := "/tmp/out.json"
	svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
		cfg := testConfig(sourcePath, targetPath)
		return &cfg, nil
	}
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return []byte(`{"a":"A"}`), nil
		case targetPath:
			return []byte(`{}`), nil
		default:
			return nil, filepath.ErrBadPattern
		}
	}
	svc.saveLock = func(_ string, _ lockfile.File) error {
		return errors.New("disk full")
	}

	writeCount := 0
	svc.writeFile = func(_ string, _ []byte) error {
		writeCount++
		return nil
	}

	_, err := svc.Run(context.Background(), Input{})
	if err == nil || !strings.Contains(err.Error(), "persist lock state") {
		t.Fatalf("expected fatal lock persistence error, got %v", err)
	}
	if writeCount != 0 {
		t.Fatalf("expected no output flush on fatal lock error, got %d writes", writeCount)
	}
}

func TestRunDryRunReportsPruneCandidates(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.json"
	targetPath := "/tmp/out.json"
	svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
		cfg := testConfig(sourcePath, targetPath)
		return &cfg, nil
	}
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return []byte(`{"hello":"Hello","nested.title":"Title"}`), nil
		case targetPath:
			return []byte(`{"hello":"Bonjour","nested":{"title":"Titre","old":"Ancien"},"legacy":"Legacy"}`), nil
		default:
			return nil, filepath.ErrBadPattern
		}
	}

	report, err := svc.Run(context.Background(), Input{DryRun: true, Prune: true})
	if err != nil {
		t.Fatalf("run dry-run prune: %v", err)
	}
	if len(report.PruneCandidates) != 2 {
		t.Fatalf("expected 2 prune candidates, got %+v", report.PruneCandidates)
	}
	if report.PruneCandidates[0].EntryKey != "legacy" || report.PruneCandidates[1].EntryKey != "nested.old" {
		t.Fatalf("unexpected prune candidates ordering: %+v", report.PruneCandidates)
	}
}

func TestRunPruneRemovesStaleKeysForJSONAndNestedKeys(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.json"
	targetPath := "/tmp/out.json"
	svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
		cfg := testConfig(sourcePath, targetPath)
		return &cfg, nil
	}
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return []byte(`{"hello":"Hello","nested.title":"Title"}`), nil
		case targetPath:
			return []byte(`{"hello":"Bonjour","nested":{"title":"Titre","old":"Ancien"},"legacy":"Legacy"}`), nil
		default:
			return nil, filepath.ErrBadPattern
		}
	}
	svc.translate = func(_ context.Context, req translator.Request) (string, error) {
		if req.Source == "Hello" {
			return "Salut", nil
		}
		return "Titre mis à jour", nil
	}

	var written []byte
	svc.writeFile = func(path string, content []byte) error {
		if path != targetPath {
			t.Fatalf("unexpected write path %q", path)
		}
		written = append([]byte(nil), content...)
		return nil
	}

	report, err := svc.Run(context.Background(), Input{Prune: true})
	if err != nil {
		t.Fatalf("run prune: %v", err)
	}
	if report.PruneApplied != 2 {
		t.Fatalf("expected 2 prune deletions applied, got %+v", report)
	}

	var payload map[string]any
	if err := json.Unmarshal(written, &payload); err != nil {
		t.Fatalf("decode written payload: %v", err)
	}
	if _, ok := payload["legacy"]; ok {
		t.Fatalf("expected legacy key pruned, got %+v", payload)
	}
	nested, ok := payload["nested"].(map[string]any)
	if !ok {
		t.Fatalf("expected nested object, got %+v", payload)
	}
	if _, ok := nested["old"]; ok {
		t.Fatalf("expected nested old key pruned, got %+v", nested)
	}
	if nested["title"] != "Titre mis à jour" {
		t.Fatalf("expected nested title preserved and updated, got %+v", nested)
	}
}

func TestRunPruneSafetyLimitBlocksMassDeletion(t *testing.T) {
	svc := newTestService()
	sourcePath := "/tmp/source.json"
	targetPath := "/tmp/out.json"
	svc.loadConfig = func(_ string) (*config.I18NConfig, error) {
		cfg := testConfig(sourcePath, targetPath)
		return &cfg, nil
	}
	svc.readFile = func(path string) ([]byte, error) {
		switch path {
		case sourcePath:
			return []byte(`{"hello":"Hello"}`), nil
		case targetPath:
			return []byte(`{"hello":"Bonjour","old":"ancien","older":"ancien2"}`), nil
		default:
			return nil, filepath.ErrBadPattern
		}
	}

	_, err := svc.Run(context.Background(), Input{Prune: true, PruneLimit: 1})
	if err == nil || !strings.Contains(err.Error(), "prune safety limit exceeded") {
		t.Fatalf("expected prune safety limit error, got %v", err)
	}
}

func newTestService() *Service {
	now := time.Unix(1700000000, 0).UTC()
	sourcePath := "/tmp/source.json"

	return &Service{
		loadConfig: func(_ string) (*config.I18NConfig, error) {
			cfg := testConfig(sourcePath, "/tmp/out.json")
			return &cfg, nil
		},
		loadLock: func(_ string) (*lockfile.File, error) {
			return &lockfile.File{LocaleStates: map[string]lockfile.LocaleCheckpoint{}, RunCompleted: map[string]lockfile.RunCompletion{}}, nil
		},
		saveLock: func(_ string, _ lockfile.File) error { return nil },
		readFile: func(path string) ([]byte, error) {
			switch path {
			case sourcePath:
				return []byte(`{"hello":"Hello"}`), nil
			case "/tmp/out.json":
				return []byte(`{}`), nil
			default:
				return nil, filepath.ErrBadPattern
			}
		},
		writeFile: func(_ string, _ []byte) error { return nil },
		translate: func(_ context.Context, req translator.Request) (string, error) {
			return strings.ToUpper(req.Source), nil
		},
		newParser: translationfileparser.NewDefaultStrategy,
		now:       func() time.Time { return now },
		numCPU:    func() int { return 2 },
	}
}

func testConfig(sourcePath, targetPath string) config.I18NConfig {
	return config.I18NConfig{
		Locales: config.LocaleConfig{
			Source:  "en",
			Targets: []string{"fr"},
		},
		Buckets: map[string]config.BucketConfig{
			"ui": {
				Files: []config.BucketFileMapping{{
					From: sourcePath,
					To:   targetPath,
				}},
			},
		},
		Groups: map[string]config.GroupConfig{
			"default": {
				Targets: []string{"fr"},
				Buckets: []string{"ui"},
			},
		},
		LLM: config.LLMConfig{
			Profiles: map[string]config.LLMProfile{
				"default": {
					Provider: "openai",
					Model:    "gpt-4.1-mini",
					Prompt:   "Translate {{source}} to {{target}}: {{input}}",
				},
			},
		},
	}
}

func TestShouldIgnoreSourcePath(t *testing.T) {
	targets := []string{"fr", "es", "zh"}
	if !shouldIgnoreSourcePath("docs/fr/index.mdx", targets) {
		t.Fatalf("expected docs/fr/index.mdx to be ignored")
	}
	if !shouldIgnoreSourcePath("docs/es/guides/quickstart.mdx", targets) {
		t.Fatalf("expected nested locale path to be ignored")
	}
	if shouldIgnoreSourcePath("docs/index.mdx", targets) {
		t.Fatalf("expected root docs source path not to be ignored")
	}
}

func TestResolveSourcePathsWithDoublestar(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "docs", "nested"), 0o755); err != nil {
		t.Fatalf("mkdir nested docs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "docs", "index.mdx"), []byte("hi"), 0o644); err != nil {
		t.Fatalf("write root mdx: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "docs", "nested", "guide.mdx"), []byte("hi"), 0o644); err != nil {
		t.Fatalf("write nested mdx: %v", err)
	}

	pattern := filepath.Join(dir, "docs", "**", "*.mdx")
	paths, err := resolveSourcePaths(pattern)
	if err != nil {
		t.Fatalf("resolve source paths: %v", err)
	}
	if len(paths) != 2 {
		t.Fatalf("expected 2 paths, got %d (%v)", len(paths), paths)
	}
}

func TestResolveTargetPathWithDoublestar(t *testing.T) {
	sourcePattern := "docs/**/*.mdx"
	targetPattern := "docs/fr/**/*.mdx"
	sourcePath := "docs/guides/quickstart.mdx"

	got, err := resolveTargetPath(sourcePattern, targetPattern, sourcePath)
	if err != nil {
		t.Fatalf("resolve target path: %v", err)
	}
	if want := "docs/fr/guides/quickstart.mdx"; got != want {
		t.Fatalf("target path = %q, want %q", got, want)
	}
}

func TestResolveTargetPathRequiresDoublestarInTargetWhenSourceHasIt(t *testing.T) {
	_, err := resolveTargetPath("docs/**/*.mdx", "docs/fr/index.mdx", "docs/index.mdx")
	if err == nil || !strings.Contains(err.Error(), "must include glob tokens") {
		t.Fatalf("expected doublestar mapping error, got %v", err)
	}
}

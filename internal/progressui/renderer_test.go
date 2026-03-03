package progressui

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestParseMode(t *testing.T) {
	t.Parallel()

	mode, err := ParseMode("auto")
	if err != nil {
		t.Fatalf("parse auto mode: %v", err)
	}
	if mode != ModeAuto {
		t.Fatalf("unexpected mode: %q", mode)
	}

	if _, err := ParseMode("blob"); err == nil {
		t.Fatal("expected invalid mode error")
	}
}

func TestIsEnabledMatrix(t *testing.T) {
	t.Parallel()

	if IsEnabled(ModeOff, bytes.NewBuffer(nil), nil) {
		t.Fatal("mode off should be disabled")
	}
	if !IsEnabled(ModeOn, bytes.NewBuffer(nil), nil) {
		t.Fatal("mode on should be enabled")
	}
	if !IsEnabled(ModeAuto, bytes.NewBuffer(nil), func(w io.Writer) bool { _ = w; return true }) {
		t.Fatal("mode auto should be enabled when tty")
	}
}

func TestModelRendersPhaseAndProgress(t *testing.T) {
	t.Parallel()

	m := newModel("Translating", ModeOn, defaultSpinnerTick, Options{})

	next, _ := m.Update(phaseMsg{text: "Planning tasks..."})
	m, _ = next.(model)
	next, _ = m.Update(planMsg{total: 10})
	m, _ = next.(model)
	next, _ = m.Update(taskDoneMsg{succeeded: 3, failed: 1, total: 10})
	m, _ = next.(model)

	view := m.View().Content
	if !strings.Contains(view, "Planning tasks") {
		t.Fatalf("expected phase in view, got %q", view)
	}
	if !strings.Contains(view, "4/10") {
		t.Fatalf("expected completion ratio in view, got %q", view)
	}
	if !strings.Contains(view, "ok=3") || !strings.Contains(view, "fail=1") {
		t.Fatalf("expected counters in view, got %q", view)
	}
}

func TestModelIndeterminateView(t *testing.T) {
	t.Parallel()

	m := newModel("Translating", ModeOn, defaultSpinnerTick, Options{})
	next, _ := m.Update(taskDoneMsg{succeeded: 1, failed: 0, total: 0})
	m, _ = next.(model)
	view := m.View().Content

	if !strings.Contains(view, "estimating workload") {
		t.Fatalf("expected indeterminate message, got %q", view)
	}
	if strings.Contains(view, "/0") {
		t.Fatalf("expected no determinate ratio, got %q", view)
	}
}

func TestModelShowsFileStatuses(t *testing.T) {
	t.Parallel()

	m := newModel("Translating", ModeOn, defaultSpinnerTick, Options{})
	next, _ := m.Update(taskStartedMsg{targetPath: "/tmp/fr/ui.json", entryKey: "welcome"})
	m, _ = next.(model)
	next, _ = m.Update(taskStatusMsg{
		targetPath:    "/tmp/fr/ui.json",
		entryKey:      "welcome",
		taskSucceeded: true,
	})
	m, _ = next.(model)

	next, _ = m.Update(taskStartedMsg{targetPath: "/tmp/fr/errors.json", entryKey: "network"})
	m, _ = next.(model)
	next, _ = m.Update(taskStatusMsg{
		targetPath:    "/tmp/fr/errors.json",
		entryKey:      "network",
		taskSucceeded: false,
		failureReason: "rate limit",
	})
	m, _ = next.(model)

	view := m.View().Content
	if !strings.Contains(view, "Files") {
		t.Fatalf("expected files section, got %q", view)
	}
	if !strings.Contains(view, "ui.json") || !strings.Contains(view, "[done] ok=1 fail=0") {
		t.Fatalf("expected successful file row, got %q", view)
	}
	if !strings.Contains(view, "errors.json") || !strings.Contains(view, "[failed] ok=0 fail=1") {
		t.Fatalf("expected failed file row, got %q", view)
	}
	if !strings.Contains(view, "reason=rate limit") {
		t.Fatalf("expected failure reason in file row, got %q", view)
	}
}

func TestModelFileStatusSortsProcessingFirst(t *testing.T) {
	t.Parallel()

	m := newModel("Translating", ModeOn, defaultSpinnerTick, Options{})
	next, _ := m.Update(taskStartedMsg{targetPath: "/tmp/fr/done.json", entryKey: "welcome"})
	m, _ = next.(model)
	next, _ = m.Update(taskStatusMsg{
		targetPath:    "/tmp/fr/done.json",
		entryKey:      "welcome",
		taskSucceeded: true,
	})
	m, _ = next.(model)

	next, _ = m.Update(taskStartedMsg{targetPath: "/tmp/fr/live.json", entryKey: "loading"})
	m, _ = next.(model)

	view := m.View().Content
	liveIdx := strings.Index(view, "live.json")
	doneIdx := strings.Index(view, "done.json")
	if liveIdx == -1 || doneIdx == -1 {
		t.Fatalf("expected both file rows in view, got %q", view)
	}
	if liveIdx > doneIdx {
		t.Fatalf("expected processing file to appear first, got %q", view)
	}
	if !strings.Contains(view, "[processing]") {
		t.Fatalf("expected processing status marker, got %q", view)
	}
}

func TestModelAdjustsProgressWidthOnWindowResize(t *testing.T) {
	t.Parallel()

	m := newModel("Translating", ModeOn, defaultSpinnerTick, Options{})
	next, _ := m.Update(tea.WindowSizeMsg{Width: 10, Height: 10})
	m, _ = next.(model)
	if got := m.bar.Width(); got != defaultBarMinWidth {
		t.Fatalf("expected min clamped bar width %d, got %d", defaultBarMinWidth, got)
	}

	next, _ = m.Update(tea.WindowSizeMsg{Width: 200, Height: 20})
	m, _ = next.(model)
	if got := m.bar.Width(); got != defaultBarMaxWidth {
		t.Fatalf("expected max clamped bar width %d, got %d", defaultBarMaxWidth, got)
	}
}

func TestModelCompleteClearsView(t *testing.T) {
	t.Parallel()

	m := newModel("Translating", ModeOn, defaultSpinnerTick, Options{})
	next, cmd := m.Update(completeMsg{})
	m, _ = next.(model)

	if !m.done {
		t.Fatal("expected model to be marked done")
	}
	if m.View().Content != "" {
		t.Fatalf("expected empty view, got %q", m.View().Content)
	}
	if cmd == nil {
		t.Fatal("expected quit command")
	}
}

func TestDebugLoggingDisabledByDefault(t *testing.T) {
	t.Setenv(envProgressDebug, "")
	t.Setenv(envProgressDebugFilePath, "")

	logPath := filepath.Join(t.TempDir(), "run.log")
	r := New(bytes.NewBuffer(nil), ModeOn, Options{
		IsTTYFn:      func(_ io.Writer) bool { return false },
		DebugLogPath: logPath,
	})
	r.Phase("Planning tasks...")
	r.Close()

	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Fatalf("expected no debug log file by default, stat err=%v", err)
	}
}

func TestDebugLoggingEnabledViaEnvDefaultPath(t *testing.T) {
	t.Setenv(envProgressDebug, "1")
	t.Setenv(envProgressDebugFilePath, "")

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}
	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(cwd)
	})

	r := New(bytes.NewBuffer(nil), ModeOn, Options{IsTTYFn: func(_ io.Writer) bool { return false }})
	r.Plan(3)
	r.TaskDone(1, 0, 3)
	r.Complete()
	r.Close()

	data, err := os.ReadFile(defaultDebugLogPath)
	if err != nil {
		t.Fatalf("read default debug log path: %v", err)
	}
	logText := string(data)
	if !strings.Contains(logText, "msg=\"renderer started\"") {
		t.Fatalf("expected renderer start entry, got %q", logText)
	}
	if !strings.Contains(logText, "msg=\"task done\"") {
		t.Fatalf("expected task done entry, got %q", logText)
	}
}

func TestDebugLoggingCustomPathViaEnv(t *testing.T) {
	t.Setenv(envProgressDebug, "true")
	customPath := filepath.Join(t.TempDir(), "custom", "progress.log")
	t.Setenv(envProgressDebugFilePath, customPath)

	r := New(bytes.NewBuffer(nil), ModeOn, Options{IsTTYFn: func(_ io.Writer) bool { return false }})
	r.Phase("Finalizing output...")
	r.Close()

	data, err := os.ReadFile(customPath)
	if err != nil {
		t.Fatalf("read custom debug log: %v", err)
	}
	if !strings.Contains(string(data), "msg=phase") {
		t.Fatalf("expected phase log entry, got %q", string(data))
	}
}

func TestDebugLoggingSetupFailureWritesStderr(t *testing.T) {
	t.Setenv(envProgressDebug, "")
	t.Setenv(envProgressDebugFilePath, "")

	oldStderr := os.Stderr
	rPipe, wPipe, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stderr pipe: %v", err)
	}
	os.Stderr = wPipe
	t.Cleanup(func() {
		os.Stderr = oldStderr
		_ = rPipe.Close()
	})

	renderer := New(bytes.NewBuffer(nil), ModeOn, Options{
		IsTTYFn:        func(_ io.Writer) bool { return false },
		EnableDebugLog: true,
		DebugLogPath:   "/dev/null/progress.log",
	})
	renderer.Close()
	_ = wPipe.Close()

	data, readErr := io.ReadAll(rPipe)
	if readErr != nil {
		t.Fatalf("read stderr: %v", readErr)
	}
	if !strings.Contains(string(data), "progress debug logging disabled") {
		t.Fatalf("expected stderr warning when debug setup fails, got %q", string(data))
	}
}

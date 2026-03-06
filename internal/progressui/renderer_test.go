package progressui

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/spinner"
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
	if !strings.Contains(view, "elapsed=") {
		t.Fatalf("expected elapsed timer in view, got %q", view)
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

func TestModelShowsTokenUsageWhenReported(t *testing.T) {
	t.Parallel()

	m := newModel("Translating", ModeOn, defaultSpinnerTick, Options{})
	next, _ := m.Update(tokenUsageMsg{
		promptTokens:     318,
		completionTokens: 167,
		totalTokens:      485,
	})
	m, _ = next.(model)

	view := m.View().Content
	if !strings.Contains(view, "prompt_tokens=318 completion_tokens=167 total_tokens=485") {
		t.Fatalf("expected token usage line in view, got %q", view)
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

func TestRendererNonInteractiveMethodsAndGuards(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	r := New(&out, ModeOn, Options{
		IsTTYFn: func(_ io.Writer) bool { return false },
	})

	r.Phase("Planning...")
	r.Plan(3)
	r.TaskStarted(" ", "ignored")
	r.TaskStarted("/tmp/fr/app.json", "hello")
	r.TaskStatus(" ", "ignored", true, "")
	r.TaskStatus("/tmp/fr/app.json", "hello", false, "rate limit")
	r.TokenUsage(10, 5, 15)
	r.TaskDone(3, 0, 3)
	r.Complete()

	beforeClose := out.String()
	if !strings.Contains(beforeClose, "progress phase=Planning...") {
		t.Fatalf("expected phase line in plain output, got %q", beforeClose)
	}
	if !strings.Contains(beforeClose, "progress executable_total=3") {
		t.Fatalf("expected plan line in plain output, got %q", beforeClose)
	}
	if !strings.Contains(beforeClose, "progress completed=3/3 succeeded=3 failed=0") {
		t.Fatalf("expected task summary line in plain output, got %q", beforeClose)
	}
	if !strings.Contains(beforeClose, "progress done succeeded=3 failed=0") {
		t.Fatalf("expected completion line in plain output, got %q", beforeClose)
	}

	r.Close()
	r.Close()

	r.Phase("after close")
	r.Plan(99)
	r.TaskStarted("/tmp/fr/app.json", "x")
	r.TaskStatus("/tmp/fr/app.json", "x", true, "")
	r.TaskDone(9, 9, 9)
	r.TokenUsage(1, 1, 2)
	r.Complete()

	if got := out.String(); got != beforeClose {
		t.Fatalf("expected no output after close; before=%q after=%q", beforeClose, got)
	}
}

func TestRendererModeAutoSuppressesPlainProgressLines(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	r := New(&out, ModeAuto, Options{
		IsTTYFn: func(_ io.Writer) bool { return false },
	})
	r.Phase("Planning...")
	r.Plan(2)
	r.TaskDone(1, 0, 2)
	r.Complete()
	r.Close()

	got := out.String()
	if strings.Contains(got, "progress phase=") || strings.Contains(got, "progress executable_total=") || strings.Contains(got, "progress completed=") {
		t.Fatalf("expected auto mode to suppress plain progress logs, got %q", got)
	}
	if !strings.Contains(got, "progress done succeeded=1 failed=0") {
		t.Fatalf("expected completion line, got %q", got)
	}
}

func TestIsEnabledUnknownModeReturnsFalse(t *testing.T) {
	t.Parallel()

	if IsEnabled(Mode("mystery"), bytes.NewBuffer(nil), nil) {
		t.Fatal("unknown mode should be disabled")
	}
}

func TestDetectTTYBranches(t *testing.T) {
	t.Parallel()

	if !detectTTY(bytes.NewBuffer(nil), func(_ io.Writer) bool { return true }) {
		t.Fatal("expected custom detector branch to return true")
	}
	if detectTTY(bytes.NewBuffer(nil), nil) {
		t.Fatal("non-file writer should not be considered a tty")
	}

	f, err := os.CreateTemp(t.TempDir(), "detect-tty-*")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	defer func() { _ = f.Close() }()
	if detectTTY(f, nil) {
		t.Fatal("regular file should not be considered a tty")
	}
}

func TestModelInitAndSpinnerDonePath(t *testing.T) {
	t.Parallel()

	m := newModel("Translating", ModeOn, defaultSpinnerTick, Options{})
	if cmd := m.Init(); cmd == nil {
		t.Fatal("expected init command")
	}

	m.done = true
	next, _ := m.Update(spinner.TickMsg{})
	updated, ok := next.(model)
	if !ok {
		t.Fatalf("expected updated model type, got %T", next)
	}
	if !updated.done {
		t.Fatal("expected done state to be preserved on spinner ticks")
	}
}

func TestSetProgressCmdClampsCompletedBounds(t *testing.T) {
	t.Parallel()

	m := newModel("Translating", ModeOn, defaultSpinnerTick, Options{})
	m.total = 2
	m.succeeded = -1
	m.failed = 0
	if cmd := m.setProgressCmd(); cmd == nil {
		t.Fatal("expected command when total is positive")
	}
	next, _ := m.Update(taskDoneMsg{succeeded: 5, failed: 5, total: 2})
	m, _ = next.(model)
	if m.succeeded != 5 || m.failed != 5 {
		t.Fatalf("expected task counts to update, got succeeded=%d failed=%d", m.succeeded, m.failed)
	}
}

func TestStatusLabelEdgeCases(t *testing.T) {
	t.Parallel()

	if got := statusLabel(" "); got != "(unknown)" {
		t.Fatalf("expected unknown label for blank path, got %q", got)
	}
	if got := statusLabel("/"); got != "/" {
		t.Fatalf("expected slash path label to remain unchanged, got %q", got)
	}
}

func TestParseBoolEnvOnAndInvalidValues(t *testing.T) {
	t.Parallel()

	if !parseBoolEnv("on") {
		t.Fatal("expected on to parse as true")
	}
	if parseBoolEnv("definitely-not-bool") {
		t.Fatal("expected invalid bool value to parse as false")
	}
}

func TestNewDebugLoggerErrorPaths(t *testing.T) {
	t.Parallel()

	if _, _, err := newDebugLogger(" "); err == nil {
		t.Fatal("expected empty path error")
	}

	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatalf("write blocker file: %v", err)
	}
	if _, _, err := newDebugLogger(filepath.Join(blocker, "run.log")); err == nil {
		t.Fatal("expected mkdir/open failure when parent is a file")
	}
}

func TestTriggerInterruptRunsCallbackOnce(t *testing.T) {
	t.Parallel()

	calls := 0
	r := &Renderer{
		onInterrupt: func() {
			calls++
		},
	}
	r.triggerInterrupt()
	r.triggerInterrupt()
	r.triggerInterrupt()

	if calls != 1 {
		t.Fatalf("expected interrupt callback to run once, got %d", calls)
	}
}

func TestLogPlainThrottleAndForce(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	r := &Renderer{
		w:    &out,
		mode: ModeOn,
	}
	r.logPlain("first", false)
	r.logPlain("second", false)
	r.logPlain("forced", true)

	got := out.String()
	if !strings.Contains(got, "first\n") {
		t.Fatalf("expected initial log line, got %q", got)
	}
	if strings.Contains(got, "second\n") {
		t.Fatalf("expected throttled line to be suppressed, got %q", got)
	}
	if !strings.Contains(got, "forced\n") {
		t.Fatalf("expected forced line, got %q", got)
	}

	r.lastPlainLog = time.Now().Add(-2 * time.Second)
	r.logPlain("third", false)
	if !strings.Contains(out.String(), "third\n") {
		t.Fatalf("expected line after throttle window, got %q", out.String())
	}
}

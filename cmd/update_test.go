package cmd

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestUpdateCommand(t *testing.T) {
	originalRunner := selfUpdateRunner
	t.Cleanup(func() {
		selfUpdateRunner = originalRunner
	})

	called := false
	selfUpdateRunner = func(_ context.Context, version string, stdout, _ io.Writer) error {
		called = true
		if version != "" {
			t.Fatalf("unexpected version: %q", version)
		}
		_, _ = io.WriteString(stdout, "updated\n")
		return nil
	}

	cmd := newUpdateCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute update command: %v", err)
	}

	if !called {
		t.Fatalf("expected selfUpdateRunner to be called")
	}

	if got, want := out.String(), "updated\n"; got != want {
		t.Fatalf("unexpected output: got %q want %q", got, want)
	}
}

func TestUpdateCommandVersionArg(t *testing.T) {
	originalRunner := selfUpdateRunner
	t.Cleanup(func() {
		selfUpdateRunner = originalRunner
	})

	selfUpdateRunner = func(_ context.Context, version string, _, _ io.Writer) error {
		if got, want := version, "v1.2.3"; got != want {
			t.Fatalf("version mismatch: got %q want %q", got, want)
		}
		return nil
	}

	cmd := newUpdateCmd()
	cmd.SetArgs([]string{"v1.2.3"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute update command with version: %v", err)
	}
}

func TestUpdateCommandRunnerError(t *testing.T) {
	originalRunner := selfUpdateRunner
	t.Cleanup(func() {
		selfUpdateRunner = originalRunner
	})

	selfUpdateRunner = func(_ context.Context, _ string, _, _ io.Writer) error {
		return errors.New("network failure")
	}

	cmd := newUpdateCmd()
	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected error")
	}

	if !strings.Contains(err.Error(), "self update: network failure") {
		t.Fatalf("unexpected error: %v", err)
	}
}

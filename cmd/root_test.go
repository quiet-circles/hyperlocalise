package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootCommandOutput(t *testing.T) {
	cmd := newRootCmd("")
	b := bytes.NewBufferString("")

	cmd.SetArgs([]string{"-h"})
	cmd.SetOut(b)

	cmdErr := cmd.Execute()
	if cmdErr != nil {
		t.Fatalf("run root help: %v", cmdErr)
	}

	if !strings.Contains(b.String(), "Usage:") {
		t.Fatalf("expected help output, got %q", b.String())
	}
}

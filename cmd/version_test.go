package cmd

import (
	"bytes"
	"fmt"
	"io"
	"testing"
)

func TestVersionCommand(t *testing.T) {
	version := "v1.0.0"
	cmd := newVersionCmd(version)
	b := bytes.NewBufferString("")
	cmd.SetOut(b)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("execute version command: %v", err)
	}

	out, err := io.ReadAll(b)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}

	if got, want := string(out), fmt.Sprintf("hyperlocalise: %s\n", version); got != want {
		t.Fatalf("unexpected output: got %q want %q", got, want)
	}
}

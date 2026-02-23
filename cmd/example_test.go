package cmd

import (
	"bytes"
	"io"
	"testing"
)

func TestExampleCommand(t *testing.T) {
	testCases := []struct {
		name     string
		args     []string
		expected string
		err      bool
	}{
		{
			name:     "normal multiply",
			args:     []string{"2", "3", "--multiply"},
			expected: "6\n",
			err:      false,
		},
		{
			name:     "invalid multiply",
			args:     []string{"2", "s", "--multiply"},
			expected: "",
			err:      true,
		},
		{
			name:     "valid add",
			args:     []string{"2", "3", "-a"},
			expected: "5\n",
			err:      false,
		},
		{
			name:     "invalid add",
			args:     []string{"s", "3", "-a"},
			expected: "",
			err:      true,
		},
		{
			name:     "missing operation",
			args:     []string{"2", "3"},
			expected: "",
			err:      true,
		},
		{
			name:     "multiple operations",
			args:     []string{"2", "3", "-a", "-m"},
			expected: "",
			err:      true,
		},
	}

	for _, tc := range testCases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			cmd := newExampleCmd()
			b := bytes.NewBufferString("")

			cmd.SetArgs(tc.args)
			cmd.SetOut(b)

			err := cmd.Execute()
			out, readErr := io.ReadAll(b)
			if readErr != nil {
				t.Fatalf("read output: %v", readErr)
			}

			if tc.err {
				if err == nil {
					t.Fatalf("expected error")
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}

				if got, want := string(out), tc.expected; got != want {
					t.Fatalf("unexpected output: got %q want %q", got, want)
				}
			}
		})
	}
}

package shell_test

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// =============================================================================
// IMPORTANT: These are END-TO-END INTEGRATION tests, NOT unit tests.
//
// These tests launch the ACTUAL COMPILED BINARY as a subprocess and pipe input
// to it. This is fundamentally different from the unit tests in shell_test.go
// which use NewWithIO() to test the shell in-process.
//
// WHY THIS MATTERS:
// - Unit tests can pass while the actual binary is broken
// - The lexer fix for $(...) might work in-process but fail when the binary
//   is built and run separately
// - These tests catch integration bugs that unit tests miss
//
// DO NOT DELETE thinking these are "duplicates" of the unit tests.
// They test different things at different levels.
// =============================================================================

func getFshPath(t *testing.T) string {
	// Look for fsh in build directory relative to repo root
	// Tests run from go/pkg/shell/ directory, so go up three levels
	path := filepath.Join("..", "..", "..", "build", "fsh")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("fsh executable not found at %s - run 'just build' first", path)
	}
	absPath, _ := filepath.Abs(path)
	return absPath
}

func runFsh(t *testing.T, input string) (stdout, stderr string, exitCode int) {
	fsh := getFshPath(t)

	cmd := exec.Command(fsh)
	cmd.Stdin = strings.NewReader(input)

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err := cmd.Run()
	exitCode = 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("failed to run fsh: %v", err)
		}
	}

	return outBuf.String(), errBuf.String(), exitCode
}

func TestIntegration_CommandSubstitution(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{
			name:     "echo with which echo",
			input:    "echo $(which echo)\n",
			contains: "/echo",
		},
		{
			name:     "simple command substitution",
			input:    "echo $(echo hello)\n",
			contains: "hello",
		},
		{
			name:     "nested command substitution",
			input:    "echo $(echo $(echo nested))\n",
			contains: "nested",
		},
		{
			name:     "backtick substitution",
			input:    "echo `echo backtick`\n",
			contains: "backtick",
		},
		{
			name:     "substitution with pipe",
			input:    "echo $(echo hello | tr a-z A-Z)\n",
			contains: "HELLO",
		},
		{
			name:     "substitution with spaces",
			input:    "echo $(echo hello world)\n",
			contains: "hello world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, exitCode := runFsh(t, tt.input)

			assert.Equal(t, 0, exitCode, "stderr: %s", stderr)

			assert.Contains(t, stdout, tt.contains, "stderr: %s", stderr)

		})
	}
}

func TestIntegration_BasicCommands(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{
			name:     "echo",
			input:    "echo hello\n",
			contains: "hello",
		},
		{
			name:     "pipeline",
			input:    "echo hello | tr a-z A-Z\n",
			contains: "HELLO",
		},
		{
			name:     "pwd",
			input:    "pwd\n",
			contains: "/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, exitCode := runFsh(t, tt.input)

			assert.Equal(t, 0, exitCode, "stderr: %s", stderr)

			assert.Contains(t, stdout, tt.contains, "stderr: %s", stderr)

		})
	}
}

package shell

import (
	"bytes"
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

// Diagnostic output format (from spec/diagnostics.md):
//   Line 1: The input line
//   Line 2: Caret markers (^) showing error location
//   Line 3: "error: " prefix followed by message
//
// Example for input "|":
//   |
//   ^
//   error: unexpected operator at end

// assertDiagnostic validates that the diagnostic output matches the expected format.
// It checks:
// - The input line appears in the output
// - Caret markers (^) appear on the line after the input
// - "error: " prefix followed by the expected message
func assertDiagnostic(t *testing.T, output, inputLine, expectedMessage string) {
	t.Helper()

	lines := strings.Split(output, "\n")

	// Find the input line in output
	inputLineIdx := -1
	for i, line := range lines {
		if line == inputLine {
			inputLineIdx = i
			break
		}
	}
	assert.NotEqual(t, -1, inputLineIdx)

	// Next line should be caret markers
	assert.Less(t, inputLineIdx+1, len(lines))

	caretLine := lines[inputLineIdx+1]
	assert.True(t, regexp.MustCompile(`^\s*\^+$`).MatchString(caretLine))

	// Next line should be "error: <message>"
	assert.Less(t, inputLineIdx+2, len(lines))

	errorLine := lines[inputLineIdx+2]
	expectedPrefix := "error: " + expectedMessage
	assert.Equal(t, expectedPrefix, errorLine)

}

func TestRun_SimpleCommand(t *testing.T) {
	stdin := strings.NewReader("echo hello\n")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	sh := NewWithIO(stdin, stdout, stderr, false)
	ctx := context.Background()

	exitCode := sh.Run(ctx)

	assert.Equal(t, 0, exitCode)

	assert.Contains(t, stdout.String(), "hello")

	assert.LessOrEqual(t, stderr.Len(), 0)

}

func TestRun_Pipeline(t *testing.T) {
	stdin := strings.NewReader("echo hello world | tr a-z A-Z\n")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	sh := NewWithIO(stdin, stdout, stderr, false)
	ctx := context.Background()

	exitCode := sh.Run(ctx)

	assert.Equal(t, 0, exitCode)

	output := strings.TrimSpace(stdout.String())
	assert.Equal(t, "HELLO WORLD", output)

}

func TestRun_ParseErrorGraceful(t *testing.T) {
	// Input with parse error followed by valid command
	stdin := strings.NewReader("|\necho recovered\n")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	sh := NewWithIO(stdin, stdout, stderr, false)
	ctx := context.Background()

	exitCode := sh.Run(ctx)

	// Should continue after parse error
	assert.Contains(t, stdout.String(), "recovered")

	// Should report parse error to stderr with exact diagnostic format
	assertDiagnostic(t, stderr.String(), "|", "unexpected operator at start: |")

	// Last command succeeded, so exit code should be 0
	assert.Equal(t, 0, exitCode)

}

func TestRunCommand_SingleCommand(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	sh := NewWithIO(nil, stdout, stderr, false)
	ctx := context.Background()

	exitCode := sh.RunCommand(ctx, "echo test")

	assert.Equal(t, 0, exitCode)

	assert.Contains(t, stdout.String(), "test")

}

func TestRunCommand_EmptyCommand(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	sh := NewWithIO(nil, stdout, stderr, false)
	ctx := context.Background()

	exitCode := sh.RunCommand(ctx, "")

	assert.Equal(t, 0, exitCode)

}

func TestRunCommand_FailedCommand(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	sh := NewWithIO(nil, stdout, stderr, false)
	ctx := context.Background()

	exitCode := sh.RunCommand(ctx, "false")

	assert.NotEqual(t, 0, exitCode)

}

func TestRunScript(t *testing.T) {
	// Create a temporary script file
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "test.sh")

	scriptContent := "echo line1\necho line2\n"
	require.NoError(t, os.WriteFile(scriptPath, []byte(scriptContent), 0644))

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	sh := NewWithIO(nil, stdout, stderr, false)
	ctx := context.Background()

	exitCode := sh.RunScript(ctx, scriptPath)

	assert.Equal(t, 0, exitCode)

	output := stdout.String()
	assert.Contains(t, output, "line1")

	assert.Contains(t, output, "line2")

}

func TestRunScript_NonexistentFile(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	sh := NewWithIO(nil, stdout, stderr, false)
	ctx := context.Background()

	exitCode := sh.RunScript(ctx, "/nonexistent/path/to/script.sh")

	assert.Equal(t, 1, exitCode)

	assert.Contains(t, stderr.String(), "cannot open script")

}

func TestExitCodePropagation(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	sh := NewWithIO(nil, stdout, stderr, false)
	ctx := context.Background()

	// Run a failing command
	exitCode := sh.RunCommand(ctx, "false")
	assert.NotEqual(t, 0, exitCode)

	// Check that lastExitCode was updated
	assert.NotEqual(t, 0, sh.lastExitCode)

	// Run a successful command
	exitCode = sh.RunCommand(ctx, "true")
	assert.Equal(t, 0, exitCode)

	// Check that lastExitCode was updated
	assert.Equal(t, 0, sh.lastExitCode)

}

func TestEmptyInputHandling(t *testing.T) {
	// Multiple empty lines followed by a command
	stdin := strings.NewReader("\n\n\necho done\n")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	sh := NewWithIO(stdin, stdout, stderr, false)
	ctx := context.Background()

	exitCode := sh.Run(ctx)

	assert.Equal(t, 0, exitCode)

	assert.Contains(t, stdout.String(), "done")

}

func TestEOFHandling(t *testing.T) {
	// Empty input (immediate EOF)
	stdin := strings.NewReader("")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	sh := NewWithIO(stdin, stdout, stderr, false)
	ctx := context.Background()

	exitCode := sh.Run(ctx)

	// Should exit gracefully with code 0
	assert.Equal(t, 0, exitCode)

	assert.LessOrEqual(t, stderr.Len(), 0)

}

func TestInteractiveMode_WelcomeMessage(t *testing.T) {
	stdin := strings.NewReader("")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	sh := NewWithIO(stdin, stdout, stderr, true)
	ctx := context.Background()

	sh.Run(ctx)

	assert.Contains(t, stdout.String(), "Welcome to Foundation Shell")

}

func TestInteractiveMode_Prompt(t *testing.T) {
	// Note: readline requires io.ReadCloser for stdin, so we use os.Pipe
	// to create a proper stdin that readline can use
	r, w, err := os.Pipe()
	require.Nil(t, err)

	defer r.Close()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	sh := NewWithIO(r, stdout, stderr, true)
	ctx := context.Background()

	// Write command and close to signal EOF
	go func() {
		w.WriteString("echo test\n")
		w.Close()
	}()

	sh.Run(ctx)

	output := stdout.String()
	// Should contain welcome message and command output
	assert.Contains(t, output, "Welcome to Foundation Shell")

	assert.Contains(t, output, "test")

}

func TestGetPrompt_SuccessColor(t *testing.T) {
	sh := &Shell{lastExitCode: 0}
	prompt := sh.getPrompt()

	assert.Contains(t, prompt, promptSuccess)

	assert.Contains(t, prompt, "$ ")

}

func TestGetPrompt_FailureColor(t *testing.T) {
	sh := &Shell{lastExitCode: 1}
	prompt := sh.getPrompt()

	assert.Contains(t, prompt, promptFailure)

	assert.Contains(t, prompt, "$ ")

}

func TestNew(t *testing.T) {
	sh := New(true)

	assert.Equal(t, os.Stdin, sh.stdin)

	assert.Equal(t, os.Stdout, sh.stdout)

	assert.Equal(t, os.Stderr, sh.stderr)

	assert.True(t, sh.isInteractive)

	assert.Equal(t, 0, sh.lastExitCode)

}

func TestNewWithIO(t *testing.T) {
	stdin := strings.NewReader("")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	sh := NewWithIO(stdin, stdout, stderr, false)

	assert.Equal(t, stdin, sh.stdin)

	assert.Equal(t, stdout, sh.stdout)

	assert.Equal(t, stderr, sh.stderr)

	assert.False(t, sh.isInteractive)

}

func TestIsTerminal(t *testing.T) {
	// With a buffer (not a terminal)
	sh := NewWithIO(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{}, false)
	assert.False(t, sh.IsTerminal())

}

func TestContextCancellation(t *testing.T) {
	// Create a context that will be cancelled
	ctx, cancel := context.WithCancel(context.Background())

	// Use a pipe so we can control input
	r, w, _ := os.Pipe()
	defer r.Close()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	sh := NewWithIO(r, stdout, stderr, false)

	// Run in a goroutine so we can verify it exits
	done := make(chan int, 1)
	go func() {
		done <- sh.Run(ctx)
	}()

	// Give the goroutine time to start and block on read
	time.Sleep(10 * time.Millisecond)

	// Cancel context and close the write end of the pipe
	// This should cause the read to unblock
	cancel()
	w.Close()

	// Wait for completion or timeout
	select {
	case <-done:
		// Good, it exited
	case <-time.After(200 * time.Millisecond):
		t.Error("expected Run to exit after context cancellation and pipe close")
	}
}

func TestMultipleCommands(t *testing.T) {
	stdin := strings.NewReader("echo first\necho second\necho third\n")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	sh := NewWithIO(stdin, stdout, stderr, false)
	ctx := context.Background()

	exitCode := sh.Run(ctx)

	assert.Equal(t, 0, exitCode)

	output := stdout.String()
	assert.Contains(t, output, "first")

	assert.Contains(t, output, "second")

	assert.Contains(t, output, "third")

}

func TestRunCommand_ParseError(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	sh := NewWithIO(nil, stdout, stderr, false)
	ctx := context.Background()

	exitCode := sh.RunCommand(ctx, "|")

	assert.Equal(t, 1, exitCode)

	// Should report parse error to stderr with exact diagnostic format
	assertDiagnostic(t, stderr.String(), "|", "unexpected operator at start: |")
}

func TestCommandSubstitution(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{
			name:     "echo with which echo",
			input:    "echo $(which echo)\n",
			contains: "/echo", // path will contain /echo (e.g., /bin/echo or /usr/bin/echo)
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdin := strings.NewReader(tt.input)
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}

			sh := NewWithIO(stdin, stdout, stderr, false)
			ctx := context.Background()

			exitCode := sh.Run(ctx)

			assert.Equal(t, 0, exitCode)

			output := stdout.String()
			assert.Contains(t, output, tt.contains)

		})
	}
}

func TestLogicalOperators(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "AND success",
			input:    "true && echo success\n",
			expected: "success",
		},
		{
			name:     "AND failure",
			input:    "false && echo should_not_appear\n",
			expected: "",
		},
		{
			name:     "OR success",
			input:    "true || echo should_not_appear\n",
			expected: "",
		},
		{
			name:     "OR failure",
			input:    "false || echo recovered\n",
			expected: "recovered",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdin := strings.NewReader(tt.input)
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}

			sh := NewWithIO(stdin, stdout, stderr, false)
			ctx := context.Background()

			sh.Run(ctx)

			output := stdout.String()
			if tt.expected == "" {
				assert.NotContains(t, output, "should_not_appear")

			} else {
				assert.Contains(t, output, tt.expected)

			}
		})
	}
}

package shell

import (
	"bytes"
	"context"
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
	if inputLineIdx == -1 {
		t.Errorf("diagnostic output missing input line %q\ngot:\n%s", inputLine, output)
		return
	}

	// Next line should be caret markers
	if inputLineIdx+1 >= len(lines) {
		t.Errorf("diagnostic output missing caret line after input\ngot:\n%s", output)
		return
	}
	caretLine := lines[inputLineIdx+1]
	if !regexp.MustCompile(`^\s*\^+$`).MatchString(caretLine) {
		t.Errorf("diagnostic caret line should be spaces and ^ only, got %q\ngot:\n%s", caretLine, output)
		return
	}

	// Next line should be "error: <message>"
	if inputLineIdx+2 >= len(lines) {
		t.Errorf("diagnostic output missing error message line\ngot:\n%s", output)
		return
	}
	errorLine := lines[inputLineIdx+2]
	expectedPrefix := "error: " + expectedMessage
	if errorLine != expectedPrefix {
		t.Errorf("diagnostic error message mismatch\nexpected: %q\ngot: %q", expectedPrefix, errorLine)
	}
}

func TestRun_SimpleCommand(t *testing.T) {
	stdin := strings.NewReader("echo hello\n")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	sh := NewWithIO(stdin, stdout, stderr, false)
	ctx := context.Background()

	exitCode := sh.Run(ctx)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}

	if !strings.Contains(stdout.String(), "hello") {
		t.Errorf("expected output to contain 'hello', got %q", stdout.String())
	}

	if stderr.Len() > 0 {
		t.Errorf("expected no stderr, got %q", stderr.String())
	}
}

func TestRun_Pipeline(t *testing.T) {
	stdin := strings.NewReader("echo hello world | tr a-z A-Z\n")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	sh := NewWithIO(stdin, stdout, stderr, false)
	ctx := context.Background()

	exitCode := sh.Run(ctx)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}

	output := strings.TrimSpace(stdout.String())
	if output != "HELLO WORLD" {
		t.Errorf("expected 'HELLO WORLD', got %q", output)
	}
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
	if !strings.Contains(stdout.String(), "recovered") {
		t.Errorf("expected output to contain 'recovered', got %q", stdout.String())
	}

	// Should report parse error to stderr with exact diagnostic format
	assertDiagnostic(t, stderr.String(), "|", "unexpected operator at end")

	// Last command succeeded, so exit code should be 0
	if exitCode != 0 {
		t.Errorf("expected exit code 0 after recovery, got %d", exitCode)
	}
}

func TestRunCommand_SingleCommand(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	sh := NewWithIO(nil, stdout, stderr, false)
	ctx := context.Background()

	exitCode := sh.RunCommand(ctx, "echo test")

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}

	if !strings.Contains(stdout.String(), "test") {
		t.Errorf("expected output to contain 'test', got %q", stdout.String())
	}
}

func TestRunCommand_EmptyCommand(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	sh := NewWithIO(nil, stdout, stderr, false)
	ctx := context.Background()

	exitCode := sh.RunCommand(ctx, "")

	if exitCode != 0 {
		t.Errorf("expected exit code 0 for empty command, got %d", exitCode)
	}
}

func TestRunCommand_FailedCommand(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	sh := NewWithIO(nil, stdout, stderr, false)
	ctx := context.Background()

	exitCode := sh.RunCommand(ctx, "false")

	if exitCode == 0 {
		t.Errorf("expected non-zero exit code, got %d", exitCode)
	}
}

func TestRunScript(t *testing.T) {
	// Create a temporary script file
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "test.sh")

	scriptContent := "echo line1\necho line2\n"
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0644); err != nil {
		t.Fatalf("failed to create script file: %v", err)
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	sh := NewWithIO(nil, stdout, stderr, false)
	ctx := context.Background()

	exitCode := sh.RunScript(ctx, scriptPath)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}

	output := stdout.String()
	if !strings.Contains(output, "line1") {
		t.Errorf("expected output to contain 'line1', got %q", output)
	}
	if !strings.Contains(output, "line2") {
		t.Errorf("expected output to contain 'line2', got %q", output)
	}
}

func TestRunScript_NonexistentFile(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	sh := NewWithIO(nil, stdout, stderr, false)
	ctx := context.Background()

	exitCode := sh.RunScript(ctx, "/nonexistent/path/to/script.sh")

	if exitCode != 1 {
		t.Errorf("expected exit code 1 for nonexistent file, got %d", exitCode)
	}

	if !strings.Contains(stderr.String(), "cannot open script") {
		t.Errorf("expected error message about opening script, got %q", stderr.String())
	}
}

func TestExitCodePropagation(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	sh := NewWithIO(nil, stdout, stderr, false)
	ctx := context.Background()

	// Run a failing command
	exitCode := sh.RunCommand(ctx, "false")
	if exitCode == 0 {
		t.Errorf("expected non-zero exit code from 'false', got %d", exitCode)
	}

	// Check that lastExitCode was updated
	if sh.lastExitCode == 0 {
		t.Errorf("expected lastExitCode to be non-zero")
	}

	// Run a successful command
	exitCode = sh.RunCommand(ctx, "true")
	if exitCode != 0 {
		t.Errorf("expected exit code 0 from 'true', got %d", exitCode)
	}

	// Check that lastExitCode was updated
	if sh.lastExitCode != 0 {
		t.Errorf("expected lastExitCode to be 0, got %d", sh.lastExitCode)
	}
}

func TestEmptyInputHandling(t *testing.T) {
	// Multiple empty lines followed by a command
	stdin := strings.NewReader("\n\n\necho done\n")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	sh := NewWithIO(stdin, stdout, stderr, false)
	ctx := context.Background()

	exitCode := sh.Run(ctx)

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}

	if !strings.Contains(stdout.String(), "done") {
		t.Errorf("expected output to contain 'done', got %q", stdout.String())
	}
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
	if exitCode != 0 {
		t.Errorf("expected exit code 0 on EOF, got %d", exitCode)
	}

	if stderr.Len() > 0 {
		t.Errorf("expected no stderr on EOF, got %q", stderr.String())
	}
}

func TestInteractiveMode_WelcomeMessage(t *testing.T) {
	stdin := strings.NewReader("")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	sh := NewWithIO(stdin, stdout, stderr, true)
	ctx := context.Background()

	sh.Run(ctx)

	if !strings.Contains(stdout.String(), "Welcome to Foundation Shell") {
		t.Errorf("expected welcome message in interactive mode, got %q", stdout.String())
	}
}

func TestInteractiveMode_Prompt(t *testing.T) {
	// Note: readline requires io.ReadCloser for stdin, so we use os.Pipe
	// to create a proper stdin that readline can use
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
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
	if !strings.Contains(output, "Welcome to Foundation Shell") {
		t.Errorf("expected welcome message in output, got %q", output)
	}
	if !strings.Contains(output, "test") {
		t.Errorf("expected 'test' in output, got %q", output)
	}
}

func TestGetPrompt_SuccessColor(t *testing.T) {
	sh := &Shell{lastExitCode: 0}
	prompt := sh.getPrompt()

	if !strings.Contains(prompt, promptSuccess) {
		t.Errorf("expected success styling in prompt, got %q", prompt)
	}
	if !strings.Contains(prompt, "$ ") {
		t.Errorf("expected '$ ' in prompt, got %q", prompt)
	}
}

func TestGetPrompt_FailureColor(t *testing.T) {
	sh := &Shell{lastExitCode: 1}
	prompt := sh.getPrompt()

	if !strings.Contains(prompt, promptFailure) {
		t.Errorf("expected failure styling in prompt, got %q", prompt)
	}
	if !strings.Contains(prompt, "$ ") {
		t.Errorf("expected '$ ' in prompt, got %q", prompt)
	}
}

func TestNew(t *testing.T) {
	sh := New(true)

	if sh.stdin != os.Stdin {
		t.Error("expected stdin to be os.Stdin")
	}
	if sh.stdout != os.Stdout {
		t.Error("expected stdout to be os.Stdout")
	}
	if sh.stderr != os.Stderr {
		t.Error("expected stderr to be os.Stderr")
	}
	if !sh.isInteractive {
		t.Error("expected isInteractive to be true")
	}
	if sh.lastExitCode != 0 {
		t.Errorf("expected lastExitCode to be 0, got %d", sh.lastExitCode)
	}
}

func TestNewWithIO(t *testing.T) {
	stdin := strings.NewReader("")
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	sh := NewWithIO(stdin, stdout, stderr, false)

	if sh.stdin != stdin {
		t.Error("expected stdin to match")
	}
	if sh.stdout != stdout {
		t.Error("expected stdout to match")
	}
	if sh.stderr != stderr {
		t.Error("expected stderr to match")
	}
	if sh.isInteractive {
		t.Error("expected isInteractive to be false")
	}
}

func TestIsTerminal(t *testing.T) {
	// With a buffer (not a terminal)
	sh := NewWithIO(strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{}, false)
	if sh.IsTerminal() {
		t.Error("expected IsTerminal to return false for non-file stdin")
	}
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

	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}

	output := stdout.String()
	if !strings.Contains(output, "first") {
		t.Errorf("expected output to contain 'first', got %q", output)
	}
	if !strings.Contains(output, "second") {
		t.Errorf("expected output to contain 'second', got %q", output)
	}
	if !strings.Contains(output, "third") {
		t.Errorf("expected output to contain 'third', got %q", output)
	}
}

func TestRunCommand_ParseError(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	sh := NewWithIO(nil, stdout, stderr, false)
	ctx := context.Background()

	exitCode := sh.RunCommand(ctx, "|")

	if exitCode != 1 {
		t.Errorf("expected exit code 1 for parse error, got %d", exitCode)
	}

	// Should report parse error to stderr with exact diagnostic format
	assertDiagnostic(t, stderr.String(), "|", "unexpected operator at end")
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

			if exitCode != 0 {
				t.Errorf("expected exit code 0, got %d, stderr: %s", exitCode, stderr.String())
			}

			output := stdout.String()
			if !strings.Contains(output, tt.contains) {
				t.Errorf("expected output to contain %q, got %q", tt.contains, output)
			}
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
				if strings.Contains(output, "should_not_appear") {
					t.Errorf("expected output to NOT contain 'should_not_appear', got %q", output)
				}
			} else {
				if !strings.Contains(output, tt.expected) {
					t.Errorf("expected output to contain %q, got %q", tt.expected, output)
				}
			}
		})
	}
}

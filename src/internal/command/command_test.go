package command

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"foundation-shell/pkg/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecuteExternalCommand(t *testing.T) {
	spec := &parser.CommandSpec{
		Args: []string{"echo", "hello"},
	}

	var stdout, stderr bytes.Buffer
	exitCode, err := Execute(context.Background(), spec, nil, &stdout, &stderr)

	require.Nil(t, err)

	assert.Equal(t, 0, exitCode)

	got := stdout.String()
	assert.Equal(t, "hello\n", got)

}

func TestExecuteWithArgs(t *testing.T) {
	spec := &parser.CommandSpec{
		Args: []string{"echo", "one", "two", "three"},
	}

	var stdout, stderr bytes.Buffer
	exitCode, err := Execute(context.Background(), spec, nil, &stdout, &stderr)

	require.Nil(t, err)

	assert.Equal(t, 0, exitCode)

	got := stdout.String()
	assert.Equal(t, "one two three\n", got)

}

func TestInputRedirection(t *testing.T) {
	// Create a temp file with content
	tmpDir := t.TempDir()
	inputFile := filepath.Join(tmpDir, "input.txt")
	require.NoError(t, os.WriteFile(inputFile, []byte("test input content"), 0644))

	spec := &parser.CommandSpec{
		Args:      []string{"cat"},
		InputFile: inputFile,
	}

	var stdout, stderr bytes.Buffer
	exitCode, err := Execute(context.Background(), spec, nil, &stdout, &stderr)

	require.Nil(t, err)

	assert.Equal(t, 0, exitCode)

	got := stdout.String()
	assert.Equal(t, "test input content", got)

}

func TestOutputRedirection(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "output.txt")

	spec := &parser.CommandSpec{
		Args:       []string{"echo", "test"},
		OutputFile: outputFile,
	}

	var stdout, stderr bytes.Buffer
	exitCode, err := Execute(context.Background(), spec, nil, &stdout, &stderr)

	require.Nil(t, err)

	assert.Equal(t, 0, exitCode)

	// Verify file contents
	content, err := os.ReadFile(outputFile)
	require.Nil(t, err)

	assert.Equal(t, "test\n", string(content))

	// stdout should be empty since it was redirected
	assert.Equal(t, "", stdout.String())

}

func TestAppendRedirection(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "output.txt")

	// First command: write line1
	spec1 := &parser.CommandSpec{
		Args:         []string{"echo", "line1"},
		OutputFile:   outputFile,
		AppendOutput: false, // truncate/overwrite
	}

	var stdout, stderr bytes.Buffer
	exitCode, err := Execute(context.Background(), spec1, nil, &stdout, &stderr)
	require.Nil(t, err)

	assert.Equal(t, 0, exitCode)

	// Second command: append line2
	spec2 := &parser.CommandSpec{
		Args:         []string{"echo", "line2"},
		OutputFile:   outputFile,
		AppendOutput: true, // append
	}

	stdout.Reset()
	stderr.Reset()
	exitCode, err = Execute(context.Background(), spec2, nil, &stdout, &stderr)
	require.Nil(t, err)

	assert.Equal(t, 0, exitCode)

	// Verify file contains both lines
	content, err := os.ReadFile(outputFile)
	require.Nil(t, err)

	expected := "line1\nline2\n"
	assert.Equal(t, expected, string(content))

}

func TestErrorRedirection(t *testing.T) {
	tmpDir := t.TempDir()
	errorFile := filepath.Join(tmpDir, "error.txt")

	// Use a command that writes to stderr - ls on a non-existent file
	spec := &parser.CommandSpec{
		Args:      []string{"ls", "/nonexistent_path_that_does_not_exist_12345"},
		ErrorFile: errorFile,
	}

	var stdout, stderr bytes.Buffer
	exitCode, err := Execute(context.Background(), spec, nil, &stdout, &stderr)

	// ls should fail with non-zero exit code
	assert.NotEqual(t, 0, exitCode)

	// Read error file
	content, err := os.ReadFile(errorFile)
	require.Nil(t, err)

	// Should contain some error message
	assert.NotEqual(t, 0, len(content))

	// stderr buffer should be empty since it was redirected
	assert.Equal(t, "", stderr.String())

}

func TestBuiltinExecution(t *testing.T) {
	spec := &parser.CommandSpec{
		Args: []string{"pwd"},
	}

	var stdout, stderr bytes.Buffer
	exitCode, err := Execute(context.Background(), spec, nil, &stdout, &stderr)

	require.Nil(t, err)

	assert.Equal(t, 0, exitCode)

	// Get current working directory
	cwd, err := os.Getwd()
	require.Nil(t, err)

	// pwd output should match current directory
	got := strings.TrimSpace(stdout.String())
	assert.Equal(t, cwd, got)

}

func TestCommandNotFound(t *testing.T) {
	spec := &parser.CommandSpec{
		Args: []string{"nonexistent_command_that_does_not_exist_12345"},
	}

	var stdout, stderr bytes.Buffer
	exitCode, err := Execute(context.Background(), spec, nil, &stdout, &stderr)

	// Not found is an ORDINARY failure: status 127, canonical message on
	// stderr, no chain-fatal error.
	require.Nil(t, err)
	assert.Equal(t, 127, exitCode)
	assert.Equal(t, "nonexistent_command_that_does_not_exist_12345: command not found\n", stderr.String())
}

func TestContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	spec := &parser.CommandSpec{
		Args: []string{"sleep", "10"},
	}

	var stdout, stderr bytes.Buffer

	// Cancel after a short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	exitCode, err := Execute(ctx, spec, nil, &stdout, &stderr)
	elapsed := time.Since(start)

	// Should complete quickly (not 10 seconds)
	assert.LessOrEqual(t, elapsed, 5*time.Second)

	// Cancellation is treated as an interrupt: status 130, no error —
	// the chain continues per operator logic.
	assert.Equal(t, 130, exitCode)
	assert.Nil(t, err)
}

func TestEmptyCommand(t *testing.T) {
	// Test with nil spec
	exitCode, err := Execute(context.Background(), nil, nil, nil, nil)
	assert.Equal(t, ErrEmptyCommand, err)

	assert.Equal(t, 1, exitCode)

	// Test with empty args
	spec := &parser.CommandSpec{
		Args: []string{},
	}
	exitCode, err = Execute(context.Background(), spec, nil, nil, nil)
	assert.Equal(t, ErrEmptyCommand, err)

	assert.Equal(t, 1, exitCode)

}

func TestInputFileNotFound(t *testing.T) {
	spec := &parser.CommandSpec{
		Args:      []string{"cat"},
		InputFile: "/nonexistent_file_that_does_not_exist_12345",
	}

	var stdout, stderr bytes.Buffer
	exitCode, err := Execute(context.Background(), spec, nil, &stdout, &stderr)

	// Redirection open failure: status 1, canonical message with the
	// UNWRAPPED os reason (filename appears exactly once), no error.
	require.Nil(t, err)
	assert.Equal(t, 1, exitCode)
	assert.Equal(t,
		"cannot open input file /nonexistent_file_that_does_not_exist_12345: no such file or directory\n",
		stderr.String())
}

func TestBuiltinWithOutputRedirection(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "pwd_output.txt")

	spec := &parser.CommandSpec{
		Args:       []string{"pwd"},
		OutputFile: outputFile,
	}

	var stdout, stderr bytes.Buffer
	exitCode, err := Execute(context.Background(), spec, nil, &stdout, &stderr)

	require.Nil(t, err)

	assert.Equal(t, 0, exitCode)

	// Get current working directory
	cwd, err := os.Getwd()
	require.Nil(t, err)

	// Read file and verify
	content, err := os.ReadFile(outputFile)
	require.Nil(t, err)

	got := strings.TrimSpace(string(content))
	assert.Equal(t, cwd, got)

	// stdout should be empty since it was redirected
	assert.Equal(t, "", stdout.String())

}

func TestSignalTermination(t *testing.T) {
	// Test that context cancellation properly terminates child processes
	ctx, cancel := context.WithCancel(context.Background())

	spec := &parser.CommandSpec{
		Args: []string{"sleep", "10"},
	}

	var stdout, stderr bytes.Buffer

	done := make(chan struct {
		exitCode int
		err      error
	})

	go func() {
		exitCode, err := Execute(ctx, spec, nil, &stdout, &stderr)
		done <- struct {
			exitCode int
			err      error
		}{exitCode, err}
	}()

	// Wait for process to start, then cancel
	time.Sleep(50 * time.Millisecond)
	cancel()

	// Wait for completion
	select {
	case result := <-done:
		assert.NotEqual(t, 0, result.exitCode)

	case <-time.After(2 * time.Second):
		t.Fatal("command did not terminate after context cancellation")
	}
}

// The old TestChildInOwnProcessGroup asserted nothing about signal behavior;
// the real tests of the SIGINT-forwarding design live in signal_test.go.

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
)

func TestExecuteExternalCommand(t *testing.T) {
	spec := &parser.CommandSpec{
		Args: []string{"echo", "hello"},
	}

	var stdout, stderr bytes.Buffer
	exitCode, err := Execute(context.Background(), spec, nil, &stdout, &stderr)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	if got := stdout.String(); got != "hello\n" {
		t.Errorf("expected output %q, got %q", "hello\n", got)
	}
}

func TestExecuteWithArgs(t *testing.T) {
	spec := &parser.CommandSpec{
		Args: []string{"echo", "one", "two", "three"},
	}

	var stdout, stderr bytes.Buffer
	exitCode, err := Execute(context.Background(), spec, nil, &stdout, &stderr)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	if got := stdout.String(); got != "one two three\n" {
		t.Errorf("expected output %q, got %q", "one two three\n", got)
	}
}

func TestInputRedirection(t *testing.T) {
	// Create a temp file with content
	tmpDir := t.TempDir()
	inputFile := filepath.Join(tmpDir, "input.txt")
	if err := os.WriteFile(inputFile, []byte("test input content"), 0644); err != nil {
		t.Fatalf("failed to create input file: %v", err)
	}

	spec := &parser.CommandSpec{
		Args:      []string{"cat"},
		InputFile: inputFile,
	}

	var stdout, stderr bytes.Buffer
	exitCode, err := Execute(context.Background(), spec, nil, &stdout, &stderr)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}
	if got := stdout.String(); got != "test input content" {
		t.Errorf("expected output %q, got %q", "test input content", got)
	}
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

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}

	// Verify file contents
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	if string(content) != "test\n" {
		t.Errorf("expected file content %q, got %q", "test\n", string(content))
	}

	// stdout should be empty since it was redirected
	if stdout.String() != "" {
		t.Errorf("expected empty stdout, got %q", stdout.String())
	}
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
	if err != nil {
		t.Fatalf("unexpected error on first command: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}

	// Second command: append line2
	spec2 := &parser.CommandSpec{
		Args:         []string{"echo", "line2"},
		OutputFile:   outputFile,
		AppendOutput: true, // append
	}

	stdout.Reset()
	stderr.Reset()
	exitCode, err = Execute(context.Background(), spec2, nil, &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error on second command: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}

	// Verify file contains both lines
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	expected := "line1\nline2\n"
	if string(content) != expected {
		t.Errorf("expected file content %q, got %q", expected, string(content))
	}
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
	if exitCode == 0 {
		t.Errorf("expected non-zero exit code, got 0")
	}

	// Read error file
	content, err := os.ReadFile(errorFile)
	if err != nil {
		t.Fatalf("failed to read error file: %v", err)
	}

	// Should contain some error message
	if len(content) == 0 {
		t.Error("expected error output in file, got empty")
	}

	// stderr buffer should be empty since it was redirected
	if stderr.String() != "" {
		t.Errorf("expected empty stderr buffer, got %q", stderr.String())
	}
}

func TestBuiltinExecution(t *testing.T) {
	spec := &parser.CommandSpec{
		Args: []string{"pwd"},
	}

	var stdout, stderr bytes.Buffer
	exitCode, err := Execute(context.Background(), spec, nil, &stdout, &stderr)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}

	// pwd output should match current directory
	got := strings.TrimSpace(stdout.String())
	if got != cwd {
		t.Errorf("expected pwd output %q, got %q", cwd, got)
	}
}

func TestCommandNotFound(t *testing.T) {
	spec := &parser.CommandSpec{
		Args: []string{"nonexistent_command_that_does_not_exist_12345"},
	}

	var stdout, stderr bytes.Buffer
	exitCode, err := Execute(context.Background(), spec, nil, &stdout, &stderr)

	if exitCode == 0 {
		t.Error("expected non-zero exit code for command not found")
	}
	if err == nil {
		t.Error("expected error for command not found")
	}
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
	if elapsed > 2*time.Second {
		t.Errorf("context cancellation took too long: %v", elapsed)
	}

	// Should have non-zero exit code
	if exitCode == 0 {
		t.Error("expected non-zero exit code for cancelled command")
	}

	// Should return context error
	if err != context.Canceled {
		t.Errorf("expected context.Canceled error, got %v", err)
	}
}

func TestEmptyCommand(t *testing.T) {
	// Test with nil spec
	exitCode, err := Execute(context.Background(), nil, nil, nil, nil)
	if err != ErrEmptyCommand {
		t.Errorf("expected ErrEmptyCommand for nil spec, got %v", err)
	}
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}

	// Test with empty args
	spec := &parser.CommandSpec{
		Args: []string{},
	}
	exitCode, err = Execute(context.Background(), spec, nil, nil, nil)
	if err != ErrEmptyCommand {
		t.Errorf("expected ErrEmptyCommand for empty args, got %v", err)
	}
	if exitCode != 1 {
		t.Errorf("expected exit code 1, got %d", exitCode)
	}
}

func TestInputFileNotFound(t *testing.T) {
	spec := &parser.CommandSpec{
		Args:      []string{"cat"},
		InputFile: "/nonexistent_file_that_does_not_exist_12345",
	}

	var stdout, stderr bytes.Buffer
	exitCode, err := Execute(context.Background(), spec, nil, &stdout, &stderr)

	if exitCode == 0 {
		t.Error("expected non-zero exit code for missing input file")
	}
	if err == nil {
		t.Error("expected error for missing input file")
	}
	if !strings.Contains(err.Error(), "cannot open input file") {
		t.Errorf("expected 'cannot open input file' error, got %v", err)
	}
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

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exitCode != 0 {
		t.Errorf("expected exit code 0, got %d", exitCode)
	}

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}

	// Read file and verify
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	got := strings.TrimSpace(string(content))
	if got != cwd {
		t.Errorf("expected file content %q, got %q", cwd, got)
	}

	// stdout should be empty since it was redirected
	if stdout.String() != "" {
		t.Errorf("expected empty stdout, got %q", stdout.String())
	}
}

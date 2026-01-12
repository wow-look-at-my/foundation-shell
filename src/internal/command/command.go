// Package command provides shell command execution, including builtins and external commands.
package command

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"

	"foundation-shell/pkg/parser"
)

var (
	// ErrEmptyCommand is returned when the command has no arguments.
	ErrEmptyCommand = errors.New("empty command")
)

// Execute runs a command based on the provided CommandSpec.
// It handles I/O redirection, builtin commands, and external command execution.
// Returns the exit code (0 for success) and any error that occurred.
func Execute(ctx context.Context, spec *parser.CommandSpec, stdin io.Reader, stdout, stderr io.Writer) (exitCode int, err error) {
	// Handle empty command
	if spec == nil || len(spec.Args) == 0 {
		return 1, ErrEmptyCommand
	}

	// Set up I/O redirection
	actualStdin := stdin
	actualStdout := stdout
	actualStderr := stderr

	// Track opened files for cleanup
	var filesToClose []*os.File
	defer func() {
		for _, f := range filesToClose {
			f.Close()
		}
	}()

	// Input redirection
	if spec.InputFile != "" {
		inputFile, err := os.Open(spec.InputFile)
		if err != nil {
			return 1, fmt.Errorf("cannot open input file %s: %w", spec.InputFile, err)
		}
		filesToClose = append(filesToClose, inputFile)
		actualStdin = inputFile
	}

	// Output redirection
	if spec.OutputFile != "" {
		outputFile, err := openOutputFile(spec.OutputFile, spec.AppendOutput)
		if err != nil {
			return 1, fmt.Errorf("cannot open output file %s: %w", spec.OutputFile, err)
		}
		filesToClose = append(filesToClose, outputFile)
		actualStdout = outputFile
	}

	// Error redirection
	if spec.ErrorFile != "" {
		errorFile, err := openOutputFile(spec.ErrorFile, spec.AppendError)
		if err != nil {
			return 1, fmt.Errorf("cannot open error file %s: %w", spec.ErrorFile, err)
		}
		filesToClose = append(filesToClose, errorFile)
		actualStderr = errorFile
	}

	commandName := spec.Args[0]
	commandArgs := spec.Args[1:]

	// Check if it's a builtin command
	if IsBuiltin(commandName) {
		err := ExecuteBuiltin(commandName, commandArgs, actualStdin, actualStdout, actualStderr)
		if err != nil {
			return 1, err
		}
		return 0, nil
	}

	// Execute external command
	return executeExternal(ctx, commandName, commandArgs, actualStdin, actualStdout, actualStderr)
}

// openOutputFile opens a file for writing, creating it if necessary.
// If append is true, the file is opened in append mode.
func openOutputFile(path string, appendMode bool) (*os.File, error) {
	flags := os.O_WRONLY | os.O_CREATE
	if appendMode {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}
	return os.OpenFile(path, flags, 0644)
}

// executeExternal runs an external command using os/exec.
func executeExternal(ctx context.Context, name string, args []string, stdin io.Reader, stdout, stderr io.Writer) (int, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	err := cmd.Run()
	if err != nil {
		// Check for context cancellation
		if ctx.Err() != nil {
			return 1, ctx.Err()
		}

		// Try to get the exit code
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode(), nil
		}

		// Command not found or other error
		return 1, err
	}

	return 0, nil
}

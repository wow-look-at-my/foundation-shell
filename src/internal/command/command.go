// Package command provides shell command execution, including builtins and external commands.
package command

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"foundation-shell/pkg/parser"
)

var (
	// ErrEmptyCommand is returned when the command has no arguments.
	ErrEmptyCommand = errors.New("empty command")
)

// ErrExit is the sentinel returned by the exit builtin. It never terminates
// the process directly; each layer decides what it means (execution.md
// §exit): the top-level chain stops and the shell terminates with Code, a
// pipeline segment merely reports Code as its status, and a command
// substitution stops only its own sequence.
type ErrExit struct{ Code int }

func (e ErrExit) Error() string {
	return fmt.Sprintf("exit %d", e.Code)
}

// LastStatus reports the shell's last recorded exit status — the same value
// $? expands to. The shell layer wires it at startup; the exit builtin uses
// it for no-argument exit. Nil means 0 (e.g. in unit tests or before any
// command has run).
var LastStatus func() int

func lastStatusValue() int {
	if LastStatus == nil {
		return 0
	}
	return LastStatus()
}

// OSReason returns the bare OS error text of err ("permission denied",
// "no such file or directory", "is a directory"), unwrapped from any
// *os.PathError / *exec.Error / syscall wrapping so messages never double
// the filename or carry a "fork/exec <path>:" prefix (execution.md).
func OSReason(err error) string {
	for {
		unwrapped := errors.Unwrap(err)
		if unwrapped == nil {
			return err.Error()
		}
		err = unwrapped
	}
}

// Execute runs a command based on the provided CommandSpec.
// It handles I/O redirection, standalone assignments, builtin commands, and
// external command execution. Returns the exit code (0 for success) and any
// error that occurred. The error return is reserved for the ErrExit sentinel
// and internal invariants: ordinary failures (command not found, redirection
// open failure, builtin failures) print ONE message to stderr and are
// reported as exit statuses so operator logic continues the chain.
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

	// Redirection open failures are ordinary status-1 failures: the
	// canonical message (redirection.md §9) prints to the command's stderr
	// (the ErrorFile opens last, so open failures always hit the original
	// stderr) and the command does NOT execute.

	// Input redirection
	if spec.InputFile != "" {
		inputFile, err := os.Open(spec.InputFile)
		if err != nil {
			fmt.Fprintf(stderr, "cannot open input file %s: %s\n", spec.InputFile, OSReason(err))
			return 1, nil
		}
		filesToClose = append(filesToClose, inputFile)
		actualStdin = inputFile
	}

	// Output redirection
	if spec.OutputFile != "" {
		outputFile, err := openOutputFile(spec.OutputFile, spec.AppendOutput)
		if err != nil {
			fmt.Fprintf(stderr, "cannot open output file %s: %s\n", spec.OutputFile, OSReason(err))
			return 1, nil
		}
		filesToClose = append(filesToClose, outputFile)
		actualStdout = outputFile
	}

	// Error redirection
	if spec.ErrorFile != "" {
		errorFile, err := openOutputFile(spec.ErrorFile, spec.AppendError)
		if err != nil {
			fmt.Fprintf(stderr, "cannot open error file %s: %s\n", spec.ErrorFile, OSReason(err))
			return 1, nil
		}
		filesToClose = append(filesToClose, errorFile)
		actualStderr = errorFile
	}

	// Standalone assignment (execution.md §Standalone Assignment): the
	// parser marked this command's sole word as NAME=VALUE. Perform it
	// in-process; like builtins, this mutates the parent shell even inside
	// pipelines and substitutions.
	if spec.IsAssignment {
		name, value, _ := strings.Cut(spec.Args[0], "=")
		if err := os.Setenv(name, value); err != nil {
			fmt.Fprintf(actualStderr, "%s: %s\n", name, OSReason(err))
			return 1, nil
		}
		return 0, nil
	}

	commandName := spec.Args[0]
	commandArgs := spec.Args[1:]

	// Check if it's a builtin command. ExecuteBuiltin's error return
	// carries only the ErrExit sentinel (propagated for the chain layer to
	// interpret) or internal failures; ordinary builtin failures print
	// their own message and surface as the exit status.
	if IsBuiltin(commandName) {
		return ExecuteBuiltin(commandName, commandArgs, actualStdin, actualStdout, actualStderr)
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
//
// While the child runs, SIGINT/SIGQUIT delivered to the shell are forwarded
// to the child and the shell survives (execution.md §Signals). This uses
// signal.Notify — never signal.Ignore, whose disposition would stick across
// commands and be inherited by later children — so it holds for EVERY
// command in the session, and each concurrent pipeline segment forwards to
// its own child independently.
func executeExternal(ctx context.Context, name string, args []string, stdin io.Reader, stdout, stderr io.Writer) (int, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	// Register BEFORE Start so a signal in the start window is absorbed by
	// the buffered channel instead of killing the shell; the forwarder
	// drains it once the child exists.
	sigCh := make(chan os.Signal, 4)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGQUIT)
	defer signal.Stop(sigCh)

	if err := cmd.Start(); err != nil {
		return classifyRunError(ctx, err, name, stderr)
	}

	// Forward signals to this child until Wait returns. The process handle
	// is captured after Start (never read cmd.Process concurrently).
	proc := cmd.Process
	forwardDone := make(chan struct{})
	defer close(forwardDone)
	go func() {
		for {
			select {
			case sig := <-sigCh:
				_ = proc.Signal(sig)
			case <-forwardDone:
				return
			}
		}
	}()

	err := cmd.Wait()
	if err == nil {
		return 0, nil
	}
	return classifyRunError(ctx, err, name, stderr)
}

// classifyRunError maps a Start/Wait error to the normative exit codes
// (execution.md §Standard Exit Codes) and prints the canonical message for
// spawn failures. Only status is reported — these are never chain-fatal.
func classifyRunError(ctx context.Context, err error, name string, stderr io.Writer) (int, error) {
	// Interrupt / cancellation: treated as SIGINT (130). Checked first —
	// a context kill surfaces as a SIGKILL death otherwise.
	if ctx.Err() != nil {
		return 130, nil
	}

	// Spawn failures — classified BEFORE the exit-status cases.
	if errors.Is(err, exec.ErrNotFound) || errors.Is(err, os.ErrNotExist) {
		fmt.Fprintf(stderr, "%s: command not found\n", name)
		return 127, nil
	}
	if errors.Is(err, os.ErrPermission) || errors.Is(err, syscall.ENOEXEC) || errors.Is(err, syscall.EISDIR) {
		fmt.Fprintf(stderr, "%s: %s\n", name, OSReason(err))
		return 126, nil
	}

	// Ran and exited non-zero, or was killed by a signal. Signaled() MUST
	// come first: ExitCode() is -1 for signal deaths, never 128+N.
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if ws, ok := exitErr.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
			return 128 + int(ws.Signal()), nil
		}
		return exitErr.ExitCode(), nil
	}

	// Producer terminated because its consumer exited (pipeline early
	// exit): success-equivalent and silent — never an error, never a
	// message (execution.md §Early Exit Terminates Producers).
	if errors.Is(err, io.ErrClosedPipe) || errors.Is(err, syscall.EPIPE) {
		return 0, nil
	}

	// Any other failure: generic status-1 with a bare-reason message.
	fmt.Fprintf(stderr, "%s: %s\n", name, OSReason(err))
	return 1, nil
}

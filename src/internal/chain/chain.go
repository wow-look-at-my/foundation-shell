// Package chain provides command chain execution with pipeline and logical operators.
package chain

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"sync"

	"foundation-shell/internal/command"
	"foundation-shell/internal/token"
	"foundation-shell/pkg/parser"
)

// Executor implements expander.SubshellExecutor for same-process subshell execution.
// It parses and executes commands using the internal parser and chain executor,
// capturing stdout for command substitution.
type Executor struct {
	ctx    context.Context
	stdin  io.Reader
	stderr io.Writer

	// LastStatus optionally reports the shell's last exit code so $? inside
	// substitution bodies expands to the outer shell's status. Nil means $?
	// expands to 0. Wired up by the shell layer.
	LastStatus func() int
}

// NewExecutor creates a new Executor for subshell execution.
func NewExecutor(ctx context.Context, stdin io.Reader, stderr io.Writer) *Executor {
	return &Executor{
		ctx:    ctx,
		stdin:  stdin,
		stderr: stderr,
	}
}

// Execute runs a command string and returns its output.
// This implements expander.SubshellExecutor interface.
func (e *Executor) Execute(cmd string) (output string, exitCode int, err error) {
	// Recursive parse: the executor hands ITSELF to the parser, so nested
	// substitutions inside the body expand through recursion -- never by
	// re-scanning spliced output -- and quoting inside the body is handled
	// by the body's own lexing.
	cmdChain, err := parser.ParseWithOptions(cmd, parser.Options{
		Executor:   e,
		LastStatus: e.LastStatus,
	})
	if err != nil {
		// A body that fails to PARSE fails the whole line: the caller
		// (parser) wraps this as "command substitution error: ...".
		return "", 1, err
	}

	// Capture stdout in a buffer
	var stdout bytes.Buffer

	// Execute using internal chain executor (same process, no external shell)
	exitCode, err = ExecuteWithIO(e.ctx, cmdChain, e.stdin, &stdout, e.stderr)
	if err != nil {
		// Runtime failures of the body never fail the outer line: the
		// chain layer already reported per-command failures to stderr, the
		// captured stdout is yielded, and the status is discarded by the
		// expansion layer (expansion.md §Failure Semantics). An ErrExit
		// sentinel stops only the body's own sequence (execution.md
		// §exit): the shell survives, so it too is absorbed here.
		var exitErr command.ErrExit
		if errors.As(err, &exitErr) {
			return stdout.String(), exitErr.Code, nil
		}
		return stdout.String(), exitCode, nil
	}

	return stdout.String(), exitCode, nil
}

// syncWriter wraps a writer with a mutex for thread-safe concurrent writes.
type syncWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func (sw *syncWriter) Write(p []byte) (n int, err error) {
	sw.mu.Lock()
	defer sw.mu.Unlock()
	return sw.w.Write(p)
}

// Execute runs a command chain, handling pipelines and logical operators.
// Returns the exit code of the last executed command.
func Execute(ctx context.Context, chain *parser.Chain) (exitCode int, err error) {
	return ExecuteWithIO(ctx, chain, os.Stdin, os.Stdout, os.Stderr)
}

// ExecuteWithIO runs a command chain with custom I/O streams.
//
// Segments (runs of |-connected commands) are evaluated left to right with
// full skip propagation (operators.md §Execution Algorithm): a skipped
// segment PRESERVES the current status, and the operator following a
// skipped segment is evaluated against that same preserved status — so
// `false && a && b` runs nothing and yields 1.
//
// The error return is reserved for the command.ErrExit sentinel (exit at
// top level stops the chain; remaining segments do not run) and internal
// failures. Ordinary command failures are exit statuses, already reported
// to stderr by the command layer, and the chain continues.
func ExecuteWithIO(ctx context.Context, chain *parser.Chain, stdin io.Reader, stdout, stderr io.Writer) (exitCode int, err error) {
	// Handle empty chain
	if chain == nil || len(chain.Commands) == 0 {
		return 0, nil
	}

	exitCode = 0
	i := 0
	for i < len(chain.Commands) {
		// Find the extent of the current pipeline segment
		pipelineEnd := i
		for pipelineEnd < len(chain.Operators) && chain.Operators[pipelineEnd] == token.Pipe {
			pipelineEnd++
		}

		// Apply the operator PRECEDING this segment to the propagated
		// status: skipping preserves exitCode, and the next iteration
		// re-tests the following operator against the same status.
		if i > 0 {
			switch chain.Operators[i-1] {
			case token.And:
				if exitCode != 0 {
					i = pipelineEnd + 1
					continue
				}
			case token.Or:
				if exitCode == 0 {
					i = pipelineEnd + 1
					continue
				}
			case token.Semicolon:
				// Unconditional.
			}
		}

		// Execute the pipeline (or single command if no pipes)
		pipelineCommands := chain.Commands[i : pipelineEnd+1]
		if len(pipelineCommands) == 1 {
			exitCode, err = command.Execute(ctx, pipelineCommands[0], stdin, stdout, stderr)
		} else {
			exitCode, err = executePipelineWithIO(ctx, pipelineCommands, stdin, stdout, stderr)
		}

		// ErrExit from a top-level segment or an internal failure stops
		// the chain: no further segments run.
		if err != nil {
			return exitCode, err
		}

		// Move past this pipeline
		i = pipelineEnd + 1
	}

	return exitCode, nil
}

// executePipelineWithIO runs multiple commands connected by pipes with custom I/O.
// Returns the exit code of the last (rightmost) command in the pipeline.
func executePipelineWithIO(ctx context.Context, commands []*parser.CommandSpec, defaultStdin io.Reader, defaultStdout, defaultStderr io.Writer) (int, error) {
	if len(commands) == 0 {
		return 0, nil
	}
	if len(commands) == 1 {
		return command.Execute(ctx, commands[0], defaultStdin, defaultStdout, defaultStderr)
	}

	// Create pipes between commands
	pipes := make([]*io.PipeReader, len(commands)-1)
	pipeWriters := make([]*io.PipeWriter, len(commands)-1)
	for i := range pipes {
		pipes[i], pipeWriters[i] = io.Pipe()
	}

	// Wrap stderr in a synchronized writer for concurrent access
	safeStderr := &syncWriter{w: defaultStderr}

	var wg sync.WaitGroup
	exitCodes := make([]int, len(commands))
	errs := make([]error, len(commands))

	// Launch all commands
	for i, cmd := range commands {
		wg.Add(1)

		// Determine stdin/stdout for this command
		var stdin io.Reader = defaultStdin
		var stdout io.Writer = defaultStdout

		if i > 0 {
			stdin = pipes[i-1]
			if cmd.InputFile != "" {
				// The command's stdin is redirected from a file, so its
				// incoming pipe is not connected: close the read end at
				// wiring time so the upstream producer terminates instead
				// of blocking forever (redirection.md §7.4).
				pipes[i-1].CloseWithError(io.ErrClosedPipe)
			}
		}
		if i < len(commands)-1 {
			stdout = pipeWriters[i]
		}

		go func(idx int, cmd *parser.CommandSpec, stdin io.Reader, stdout io.Writer) {
			defer wg.Done()

			// When this command finishes (or fails to start), close BOTH
			// of its pipe ends: the write end of its stdout pipe so the
			// downstream consumer sees EOF, AND the read end of its stdin
			// pipe so a blocked upstream producer's write fails (the
			// in-process equivalent of EPIPE) instead of deadlocking. The
			// producer treats that write failure as a silent, ordinary
			// termination (execution.md §Early Exit Terminates Producers).
			if idx < len(pipeWriters) {
				defer pipeWriters[idx].Close()
			}
			if idx > 0 {
				defer pipes[idx-1].CloseWithError(io.ErrClosedPipe)
			}

			exitCodes[idx], errs[idx] = command.Execute(ctx, cmd, stdin, stdout, safeStderr)
		}(i, cmd, stdin, stdout)
	}

	wg.Wait()

	// An exit builtin inside a multi-command pipeline sets only that
	// command's status; the shell survives (execution.md §exit). Any other
	// error is internal and propagates.
	var internalErr error
	for idx, err := range errs {
		if err == nil {
			continue
		}
		var exitErr command.ErrExit
		if errors.As(err, &exitErr) {
			exitCodes[idx] = exitErr.Code
			continue
		}
		if internalErr == nil {
			internalErr = err
		}
	}

	// Return exit code of last command
	lastIdx := len(commands) - 1
	return exitCodes[lastIdx], internalErr
}

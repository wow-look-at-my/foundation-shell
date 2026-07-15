// Package chain provides command chain execution with pipeline and logical operators.
package chain

import (
	"bytes"
	"context"
	"fmt"
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
	// expands to 0. Wired up by the shell layer (batch 2c).
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
func (e *Executor) Execute(command string) (output string, exitCode int, err error) {
	// Recursive parse: the executor hands ITSELF to the parser, so nested
	// substitutions inside the body expand through recursion -- never by
	// re-scanning spliced output -- and quoting inside the body is handled
	// by the body's own lexing.
	cmdChain, err := parser.ParseWithOptions(command, parser.Options{
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
		// RUNTIME failure of the body (command not found, redirection open
		// failure, ...): report it on stderr, keep whatever stdout was
		// captured, and DISCARD the failure so the outer line continues --
		// a substitution's exit code is discarded (expansion.md). This must
		// NOT propagate as an error: that would turn a runtime failure into
		// a parse-failing "command substitution error".
		//
		// TODO(batch 2c, audit issues 4/8): once the chain layer reports
		// per-command failures to stderr itself (with real 127/126 exit
		// codes) instead of returning them as chain-fatal errors, this
		// interim print becomes redundant and should be removed.
		fmt.Fprintln(e.stderr, err)
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
// Returns the exit code of the last executed command.
func ExecuteWithIO(ctx context.Context, chain *parser.Chain, stdin io.Reader, stdout, stderr io.Writer) (exitCode int, err error) {
	// Handle empty chain
	if chain == nil || len(chain.Commands) == 0 {
		return 0, nil
	}

	// Single command: just execute it
	if len(chain.Commands) == 1 {
		return command.Execute(ctx, chain.Commands[0], stdin, stdout, stderr)
	}

	// Multiple commands with operators
	// Process commands, grouping consecutive pipes into pipelines
	i := 0
	for i < len(chain.Commands) {
		// Find the extent of the current pipeline segment
		pipelineEnd := i
		for pipelineEnd < len(chain.Operators) && chain.Operators[pipelineEnd] == token.Pipe {
			pipelineEnd++
		}

		// Execute the pipeline (or single command if no pipes)
		pipelineCommands := chain.Commands[i : pipelineEnd+1]
		if len(pipelineCommands) == 1 {
			exitCode, err = command.Execute(ctx, pipelineCommands[0], stdin, stdout, stderr)
		} else {
			exitCode, err = executePipelineWithIO(ctx, pipelineCommands, stdin, stdout, stderr)
		}

		// If there's an error that's not just a non-zero exit, return it
		if err != nil {
			return exitCode, err
		}

		// Move past this pipeline
		i = pipelineEnd + 1

		// Check if there's a logical operator after this pipeline
		if pipelineEnd < len(chain.Operators) {
			op := chain.Operators[pipelineEnd]
			switch op {
			case token.And:
				// AND: skip next if previous failed (non-zero exit)
				if exitCode != 0 {
					// Skip next command/pipeline
					i++
					// Skip any pipes that follow
					for i < len(chain.Operators) && chain.Operators[i-1] == token.Pipe {
						i++
					}
				}
			case token.Or:
				// OR: skip next if previous succeeded (zero exit)
				if exitCode == 0 {
					// Skip next command/pipeline
					i++
					// Skip any pipes that follow
					for i < len(chain.Operators) && chain.Operators[i-1] == token.Pipe {
						i++
					}
				}
			case token.Semicolon:
				// Semicolon: unconditionally continue to next command
			}
		}
	}

	return exitCode, nil
}

// executePipeline runs multiple commands connected by pipes using default I/O.
// Returns the exit code of the last command in the pipeline.
func executePipeline(ctx context.Context, commands []*parser.CommandSpec) (int, error) {
	return executePipelineWithIO(ctx, commands, os.Stdin, os.Stdout, os.Stderr)
}

// executePipelineWithIO runs multiple commands connected by pipes with custom I/O.
// Returns the exit code of the last command in the pipeline.
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
	errors := make([]error, len(commands))

	// Launch all commands
	for i, cmd := range commands {
		wg.Add(1)

		// Determine stdin/stdout for this command
		var stdin io.Reader = defaultStdin
		var stdout io.Writer = defaultStdout

		if i > 0 {
			stdin = pipes[i-1]
		}
		if i < len(commands)-1 {
			stdout = pipeWriters[i]
		}

		go func(idx int, cmd *parser.CommandSpec, stdin io.Reader, stdout io.Writer) {
			defer wg.Done()

			// Close pipe writer when done (if this command writes to a pipe)
			if idx < len(pipeWriters) {
				defer pipeWriters[idx].Close()
			}

			exitCodes[idx], errors[idx] = command.Execute(ctx, cmd, stdin, stdout, safeStderr)
		}(i, cmd, stdin, stdout)
	}

	wg.Wait()

	// Return exit code of last command
	lastIdx := len(commands) - 1
	return exitCodes[lastIdx], errors[lastIdx]
}

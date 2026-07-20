// Package shell provides the main REPL (Read-Eval-Print Loop) for the foundation shell.
package shell

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"foundation-shell/internal/chain"
	"foundation-shell/internal/command"
	"foundation-shell/internal/expander"
	"foundation-shell/internal/syntax"
	"foundation-shell/pkg/parser"

	"github.com/chzyer/readline"
	"golang.org/x/term"
)

const (
	// ANSI escape codes for prompt styling
	promptSuccess = "\033[32m" // Green for successful command
	promptFailure = "\033[31m" // Red for failed command
	promptReset   = "\033[0m"

	welcomeMessage = "Welcome to Foundation Shell\n"
)

// Shell represents the shell state and I/O configuration.
type Shell struct {
	stdin         io.Reader
	stdout        io.Writer
	stderr        io.Writer
	isInteractive bool
	lastExitCode  int
	executor      expander.SubshellExecutor
}

// New creates a new Shell with standard I/O and the specified interactivity mode.
func New(interactive bool) *Shell {
	return &Shell{
		stdin:         os.Stdin,
		stdout:        os.Stdout,
		stderr:        os.Stderr,
		isInteractive: interactive,
		lastExitCode:  0,
	}
}

// NewWithIO creates a new Shell with custom I/O streams, useful for testing.
func NewWithIO(stdin io.Reader, stdout, stderr io.Writer, interactive bool) *Shell {
	return &Shell{
		stdin:         stdin,
		stdout:        stdout,
		stderr:        stderr,
		isInteractive: interactive,
		lastExitCode:  0,
	}
}

// IsTerminal checks if the given reader is connected to a TTY.
func (s *Shell) IsTerminal() bool {
	if f, ok := s.stdin.(*os.File); ok {
		return term.IsTerminal(int(f.Fd()))
	}
	return false
}

// IsTerminal checks if os.Stdin is connected to a TTY.
func IsTerminal() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// Run starts the main REPL loop.
// Returns the last exit code when the loop terminates.
func (s *Shell) Run(ctx context.Context) int {
	s.setupExecution(ctx)

	if s.isInteractive {
		fmt.Fprint(s.stdout, welcomeMessage)
		return s.runInteractive(ctx)
	}
	return s.runNonInteractive(ctx)
}

// setupExecution wires the subshell executor for command substitution and
// the last-status sources: $? (and the exit builtin's no-argument form)
// report s.lastExitCode everywhere, including inside substitution bodies.
func (s *Shell) setupExecution(ctx context.Context) {
	lastStatus := func() int { return s.lastExitCode }
	if s.executor == nil {
		// Create executor for subshell expansion (same-process execution)
		executor := chain.NewExecutor(ctx, s.stdin, s.stderr)
		executor.LastStatus = lastStatus
		s.executor = executor
	}
	command.LastStatus = lastStatus
}

// executeCommand parses and executes one input (a REPL line, or the whole
// text in non-interactive mode; unquoted newlines separate commands).
// Returns true if execution should continue, false when the shell must
// terminate (the exit builtin ran at top level).
func (s *Shell) executeCommand(ctx context.Context, input string) bool {
	cmdChain, err := parser.ParseWithOptions(input, parser.Options{
		Executor:   s.executor,
		LastStatus: func() int { return s.lastExitCode },
	})
	if err != nil {
		if errors.Is(err, parser.ErrEmptyInput) {
			return true
		}
		// Use syntax analyzer for better error display
		result := syntax.Analyze(input)
		if !result.Valid {
			fmt.Fprint(s.stderr, syntax.FormatDiagnostics(input, result.Errors))
		} else {
			fmt.Fprintf(s.stderr, "parse error: %v\n", err)
		}
		s.lastExitCode = 1
		return true
	}

	exitCode, err := chain.ExecuteWithIO(ctx, cmdChain, s.stdin, s.stdout, s.stderr)
	if err != nil {
		var exitErr command.ErrExit
		if errors.As(err, &exitErr) {
			// exit at top level: the chain already stopped; record the
			// code and terminate the shell. Remaining input does not run.
			s.lastExitCode = exitErr.Code
			return false
		}
		// Internal failures print bare to stderr — runtime command
		// failures were already reported by the command layer, and
		// messages carry no "execution error:" wrapper.
		fmt.Fprintln(s.stderr, err)
	}
	s.lastExitCode = exitCode
	return true
}

// syntaxPainter implements readline.Painter for syntax highlighting.
type syntaxPainter struct {
	highlighter *syntax.Highlighter
}

func (p *syntaxPainter) Paint(line []rune, pos int) []rune {
	return []rune(p.highlighter.Highlight(string(line)))
}

// runInteractive handles the REPL loop for interactive mode using readline.
func (s *Shell) runInteractive(ctx context.Context) int {
	painter := &syntaxPainter{highlighter: syntax.NewHighlighter(syntax.DefaultTheme)}

	cfg := &readline.Config{
		Prompt:          s.getPrompt(),
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
		Painter:         painter,
		Stdout:          s.stdout,
		Stderr:          s.stderr,
	}

	// Only set Stdin if it's a ReadCloser (required by readline)
	if rc, ok := s.stdin.(io.ReadCloser); ok {
		cfg.Stdin = rc
	}

	rl, err := readline.NewEx(cfg)
	if err != nil {
		fmt.Fprintf(s.stderr, "readline init error: %v\n", err)
		return 1
	}
	defer rl.Close()

	for {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return s.lastExitCode
		default:
		}

		// Update prompt based on last exit code
		rl.SetPrompt(s.getPrompt())

		// Read next line
		line, err := rl.Readline()
		if err != nil {
			if err == readline.ErrInterrupt {
				// Ctrl+C pressed, continue to next prompt
				continue
			}
			if err == io.EOF {
				// Ctrl+D or EOF, graceful exit
				break
			}
			fmt.Fprintf(s.stderr, "read error: %v\n", err)
			s.lastExitCode = 1
			break
		}

		// Skip empty lines
		if line == "" {
			continue
		}

		if !s.executeCommand(ctx, line) {
			break
		}
	}

	return s.lastExitCode
}

// runNonInteractive executes piped input in whole-input mode: ALL of stdin
// is read to EOF and parsed as ONE input (execution.md §Non-Interactive
// Mode). Unquoted newlines separate commands and comments — including a
// shebang line — are removed by the lexer, so quoted strings, substitution
// bodies, and operator continuations may span lines. Consequences, all
// specified: a parse error anywhere rejects the whole input (status 1,
// nothing executes); commands inherit the shell's stdin, which is at EOF;
// every $? expands to the pre-input status; exit stops the sequence.
func (s *Shell) runNonInteractive(ctx context.Context) int {
	data, err := io.ReadAll(s.stdin)
	if err != nil {
		fmt.Fprintf(s.stderr, "read error: %v\n", err)
		s.lastExitCode = 1
		return s.lastExitCode
	}

	if ctx.Err() != nil {
		return s.lastExitCode
	}

	s.executeCommand(ctx, string(data))
	return s.lastExitCode
}

// RunScript executes a script file: the file is read in full and executed
// exactly like non-interactive whole-input mode. The script file is NOT the
// shell's stdin — commands inherit the stdin the shell itself was started
// with, so `fsh script.fsh < data.txt` lets commands in the script read
// data.txt. The shebang line is an ordinary # comment.
func (s *Shell) RunScript(ctx context.Context, filename string) int {
	data, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(s.stderr, "cannot open script file %s: %s\n", filename, command.OSReason(err))
		s.lastExitCode = 1
		return s.lastExitCode
	}

	s.setupExecution(ctx)
	s.executeCommand(ctx, string(data))
	return s.lastExitCode
}

// RunCommand executes a single command string as ONE input with whole-input
// semantics (newlines inside the string separate commands). This is the
// entry point used by fsh-exec.
// Returns the exit code of the command.
func (s *Shell) RunCommand(ctx context.Context, cmdStr string) int {
	s.setupExecution(ctx)
	s.executeCommand(ctx, cmdStr)
	return s.lastExitCode
}

// getPrompt returns the shell prompt styled based on the last exit code.
func (s *Shell) getPrompt() string {
	if s.lastExitCode == 0 {
		return promptSuccess + "$ " + promptReset
	}
	return promptFailure + "$ " + promptReset
}

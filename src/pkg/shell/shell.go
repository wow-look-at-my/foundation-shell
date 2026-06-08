// Package shell provides the main REPL (Read-Eval-Print Loop) for the foundation shell.
package shell

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"foundation-shell/internal/chain"
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
	// Create executor for subshell expansion (same-process execution)
	s.executor = chain.NewExecutor(ctx, s.stdin, s.stderr)

	if s.isInteractive {
		fmt.Fprint(s.stdout, welcomeMessage)
		return s.runInteractive(ctx)
	}
	return s.runNonInteractive(ctx)
}

// executeCommand parses and executes a single command line.
// Returns true if execution should continue, false on fatal errors.
func (s *Shell) executeCommand(ctx context.Context, line string) bool {
	cmdChain, err := parser.ParseWithExecutor(line, s.executor)
	if err != nil {
		if errors.Is(err, parser.ErrEmptyInput) {
			return true
		}
		// Use syntax analyzer for better error display
		result := syntax.Analyze(line)
		if !result.Valid {
			fmt.Fprint(s.stderr, syntax.FormatDiagnostics(line, result.Errors))
		} else {
			fmt.Fprintf(s.stderr, "parse error: %v\n", err)
		}
		s.lastExitCode = 1
		return true
	}

	exitCode, err := chain.ExecuteWithIO(ctx, cmdChain, s.stdin, s.stdout, s.stderr)
	if err != nil {
		fmt.Fprintf(s.stderr, "execution error: %v\n", err)
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

		s.executeCommand(ctx, line)
	}

	return s.lastExitCode
}

// runNonInteractive handles the REPL loop for non-interactive mode (piped input).
func (s *Shell) runNonInteractive(ctx context.Context) int {
	scanner := bufio.NewScanner(s.stdin)

	for {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return s.lastExitCode
		default:
		}

		// Read next line
		if !scanner.Scan() {
			// EOF or error
			if err := scanner.Err(); err != nil {
				fmt.Fprintf(s.stderr, "read error: %v\n", err)
				s.lastExitCode = 1
			}
			// EOF is graceful exit
			break
		}

		line := scanner.Text()

		// Skip empty lines
		if line == "" {
			continue
		}

		s.executeCommand(ctx, line)
	}

	return s.lastExitCode
}

// RunScript executes commands from a script file.
// Returns the exit code of the last command executed.
func (s *Shell) RunScript(ctx context.Context, filename string) int {
	file, err := os.Open(filename)
	if err != nil {
		fmt.Fprintf(s.stderr, "cannot open script: %v\n", err)
		return 1
	}
	defer file.Close()

	// Create a non-interactive shell with the file as stdin
	scriptShell := NewWithIO(file, s.stdout, s.stderr, false)
	return scriptShell.Run(ctx)
}

// RunCommand executes a single command string.
// Returns the exit code of the command.
func (s *Shell) RunCommand(ctx context.Context, cmdStr string) int {
	if s.executor == nil {
		s.executor = chain.NewExecutor(ctx, s.stdin, s.stderr)
	}
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

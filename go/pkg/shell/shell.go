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
	"foundation-shell/pkg/parser"

	"golang.org/x/term"
)

const (
	// ANSI escape codes for colors
	colorGreen = "\033[32m"
	colorRed   = "\033[31m"
	colorReset = "\033[0m"

	welcomeMessage = "Welcome to Foundation Shell\n"
)

// Shell represents the shell state and I/O configuration.
type Shell struct {
	stdin         io.Reader
	stdout        io.Writer
	stderr        io.Writer
	isInteractive bool
	lastExitCode  int
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
	// Check if stdin is an *os.File with a valid file descriptor
	if f, ok := s.stdin.(*os.File); ok {
		return term.IsTerminal(int(f.Fd()))
	}
	return false
}

// IsTerminal checks if os.Stdin is connected to a TTY.
// This is a convenience function for use before creating a Shell.
func IsTerminal() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// Run starts the main REPL loop.
// Returns the last exit code when the loop terminates.
func (s *Shell) Run(ctx context.Context) int {
	if s.isInteractive {
		fmt.Fprint(s.stdout, welcomeMessage)
	}

	scanner := bufio.NewScanner(s.stdin)

	for {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			return s.lastExitCode
		default:
		}

		// Print prompt in interactive mode
		if s.isInteractive {
			fmt.Fprint(s.stdout, s.getPrompt())
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

		// Parse the input
		cmdChain, err := parser.Parse(line)
		if err != nil {
			// Handle empty input gracefully (not an error to user)
			if errors.Is(err, parser.ErrEmptyInput) {
				continue
			}
			fmt.Fprintf(s.stderr, "parse error: %v\n", err)
			s.lastExitCode = 1
			continue
		}

		// Execute the command chain
		exitCode, err := chain.ExecuteWithIO(ctx, cmdChain, s.stdin, s.stdout, s.stderr)
		if err != nil {
			fmt.Fprintf(s.stderr, "execution error: %v\n", err)
		}
		s.lastExitCode = exitCode
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
	// Parse the command
	cmdChain, err := parser.Parse(cmdStr)
	if err != nil {
		if errors.Is(err, parser.ErrEmptyInput) {
			return 0
		}
		fmt.Fprintf(s.stderr, "parse error: %v\n", err)
		return 1
	}

	// Execute the command chain
	exitCode, err := chain.ExecuteWithIO(ctx, cmdChain, s.stdin, s.stdout, s.stderr)
	if err != nil {
		fmt.Fprintf(s.stderr, "execution error: %v\n", err)
	}
	s.lastExitCode = exitCode
	return exitCode
}

// getPrompt returns the shell prompt with color based on the last exit code.
// Green for success (exit code 0), red for failure (non-zero exit code).
func (s *Shell) getPrompt() string {
	if s.lastExitCode == 0 {
		return colorGreen + "$ " + colorReset
	}
	return colorRed + "$ " + colorReset
}

// Package command provides shell command handling, including builtin commands.
package command

import (
	"fmt"
	"io"
	"os"
	"strconv"
)

// BuiltinFunc is the signature for builtin command functions.
// It takes arguments, standard I/O streams, and returns an error if the command fails.
type BuiltinFunc func(args []string, stdin io.Reader, stdout, stderr io.Writer) error

// Builtins maps command names to their implementation functions.
var Builtins = map[string]BuiltinFunc{
	"cd":    builtinCd,
	"pwd":   builtinPwd,
	"exit":  builtinExit,
	"clear": builtinClear,
	"help":  builtinHelp,
}

// IsBuiltin returns true if the given command name is a builtin command.
func IsBuiltin(name string) bool {
	_, ok := Builtins[name]
	return ok
}

// ExecuteBuiltin executes a builtin command by name with the given arguments and I/O streams.
// Returns an error if the command is not a builtin or if execution fails.
func ExecuteBuiltin(name string, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	fn, ok := Builtins[name]
	if !ok {
		return fmt.Errorf("not a builtin: %s", name)
	}
	return fn(args, stdin, stdout, stderr)
}

// builtinCd changes the current working directory.
// If no argument is provided, changes to $HOME.
// Returns an error if the directory doesn't exist or can't be changed to.
func builtinCd(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	var dir string

	if len(args) == 0 {
		// No argument: go to $HOME
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("cd: could not determine home directory: %w", err)
		}
		dir = home
	} else {
		dir = args[0]
	}

	if err := os.Chdir(dir); err != nil {
		return fmt.Errorf("cd: %w", err)
	}

	return nil
}

// builtinPwd prints the current working directory to stdout.
func builtinPwd(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("pwd: %w", err)
	}

	fmt.Fprintln(stdout, cwd)
	return nil
}

// builtinExit exits the shell with the given exit code (default 0).
func builtinExit(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	exitCode := 0

	if len(args) > 0 {
		code, err := strconv.Atoi(args[0])
		if err != nil {
			fmt.Fprintf(stderr, "exit: %s: numeric argument required\n", args[0])
			os.Exit(2)
		}
		exitCode = code
	}

	os.Exit(exitCode)
	return nil // unreachable, but needed for the compiler
}

// builtinClear clears the terminal screen using ANSI escape sequences.
func builtinClear(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	// \033[2J clears the entire screen
	// \033[H moves the cursor to the home position (top-left)
	fmt.Fprint(stdout, "\033[2J\033[H")
	return nil
}

// builtinHelp displays help information about available commands.
func builtinHelp(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	const headerColor = "\033[1;34m" // Bold blue
	const cmdColor = "\033[1;32m"    // Bold green
	const reset = "\033[0m"

	fmt.Fprintf(stdout, "%sFoundation Shell - Available Commands:%s\n", headerColor, reset)
	fmt.Fprintf(stdout, "%s  cd [dir]%s     - Change directory (no arg: go to $HOME)\n", cmdColor, reset)
	fmt.Fprintf(stdout, "%s  pwd%s          - Print working directory\n", cmdColor, reset)
	fmt.Fprintf(stdout, "%s  exit [code]%s  - Exit the shell (default code: 0)\n", cmdColor, reset)
	fmt.Fprintf(stdout, "%s  clear%s        - Clear the screen\n", cmdColor, reset)
	fmt.Fprintf(stdout, "%s  help%s         - Display this help message\n", cmdColor, reset)
	fmt.Fprintln(stdout)
	fmt.Fprintf(stdout, "%sSpecial Characters:%s\n", headerColor, reset)
	fmt.Fprintf(stdout, "%s  |%s            - Pipe output of one command to another\n", cmdColor, reset)
	fmt.Fprintf(stdout, "%s  &&%s           - Chain commands (execute next only if previous succeeds)\n", cmdColor, reset)
	fmt.Fprintf(stdout, "%s  > file%s       - Redirect output to file\n", cmdColor, reset)
	fmt.Fprintf(stdout, "%s  >> file%s      - Append output to file\n", cmdColor, reset)
	fmt.Fprintf(stdout, "%s  < file%s       - Redirect input from file\n", cmdColor, reset)
	fmt.Fprintf(stdout, "%s  2> file%s      - Redirect error output to file\n", cmdColor, reset)
	fmt.Fprintf(stdout, "%s  2>> file%s     - Append error output to file\n", cmdColor, reset)
	fmt.Fprintf(stdout, "%s  &%s            - Run command in background\n", cmdColor, reset)

	return nil
}

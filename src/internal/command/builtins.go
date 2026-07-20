// Package command provides shell command handling, including builtin commands.
package command

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// BuiltinFunc is the signature for builtin command functions.
// Builtins return their exit status directly. The error return is reserved
// for the ErrExit sentinel and internal failures: ordinary builtin failures
// (missing directory, invalid export name) print their own prefix-free
// message to stderr — `<builtin>: <reason>` — and return a non-zero status,
// so the chain continues per operator logic (execution.md §Builtin Commands).
type BuiltinFunc func(args []string, stdin io.Reader, stdout, stderr io.Writer) (int, error)

// Builtins maps command names to their implementation functions.
var Builtins = map[string]BuiltinFunc{
	"cd":     builtinCd,
	"pwd":    builtinPwd,
	"exit":   builtinExit,
	"clear":  builtinClear,
	"help":   builtinHelp,
	"export": builtinExport,
}

// IsBuiltin returns true if the given command name is a builtin command.
func IsBuiltin(name string) bool {
	_, ok := Builtins[name]
	return ok
}

// ExecuteBuiltin executes a builtin command by name with the given arguments and I/O streams.
// Returns the exit status; the error is the ErrExit sentinel or an internal
// failure (unknown builtin).
func ExecuteBuiltin(name string, args []string, stdin io.Reader, stdout, stderr io.Writer) (int, error) {
	fn, ok := Builtins[name]
	if !ok {
		return 1, fmt.Errorf("not a builtin: %s", name)
	}
	return fn(args, stdin, stdout, stderr)
}

// builtinCd changes the current working directory.
// If no argument is provided, changes to the home directory. Extra
// arguments are ignored (only the first is used).
func builtinCd(args []string, stdin io.Reader, stdout, stderr io.Writer) (int, error) {
	var dir string

	if len(args) == 0 {
		// No argument: go to the home directory
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintln(stderr, "cd: could not determine home directory")
			return 1, nil
		}
		dir = home
	} else {
		dir = args[0]
	}

	if err := os.Chdir(dir); err != nil {
		fmt.Fprintf(stderr, "cd: %s: %s\n", dir, OSReason(err))
		return 1, nil
	}

	return 0, nil
}

// builtinPwd prints the current working directory to stdout.
func builtinPwd(args []string, stdin io.Reader, stdout, stderr io.Writer) (int, error) {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(stderr, "pwd: %s\n", OSReason(err))
		return 1, nil
	}

	if _, err := fmt.Fprintln(stdout, cwd); err != nil {
		// A write failure (e.g. the downstream pipeline consumer already
		// exited) is silent: early-exit terminations produce no message.
		return 1, nil
	}
	return 0, nil
}

// builtinExit requests shell termination via the ErrExit sentinel — it
// NEVER calls os.Exit (that would bypass output capture, deferred file
// closes, and readline teardown). No argument exits with the shell's last
// recorded status; a numeric argument is taken modulo 256, non-negative;
// extra arguments are ignored. A non-numeric argument is a usage error:
// status 2, no sentinel, the shell keeps running.
func builtinExit(args []string, stdin io.Reader, stdout, stderr io.Writer) (int, error) {
	exitCode := lastStatusValue()

	if len(args) > 0 {
		code, err := strconv.Atoi(args[0])
		if err != nil {
			fmt.Fprintf(stderr, "exit: %s: numeric argument required\n", args[0])
			return 2, nil
		}
		exitCode = ((code % 256) + 256) % 256
	}

	return exitCode, ErrExit{Code: exitCode}
}

// builtinClear clears the terminal screen using ANSI escape sequences.
func builtinClear(args []string, stdin io.Reader, stdout, stderr io.Writer) (int, error) {
	// \033[2J clears the entire screen
	// \033[H moves the cursor to the home position (top-left)
	fmt.Fprint(stdout, "\033[2J\033[H")
	return 0, nil
}

// exportNamePattern matches a valid variable name (expansion.md §Variable
// Expansion).
var exportNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// builtinExport sets environment variables or prints the environment.
//   - No arguments: print the entire environment sorted by name, one
//     NAME=value per line.
//   - NAME=VALUE: set the variable (first '=' splits; the value may be
//     empty or contain '=').
//   - NAME (valid name, no '='): no-op success — every variable is already
//     an environment variable in the same-process model.
//   - Invalid name: `export: invalid name: <arg>` to stderr; processing
//     CONTINUES; the final status is 1 if any argument failed.
//
// Arguments arrive fully expanded (`export A=$B` assigns B's value).
func builtinExport(args []string, stdin io.Reader, stdout, stderr io.Writer) (int, error) {
	if len(args) == 0 {
		env := os.Environ()
		sort.Strings(env)
		for _, entry := range env {
			if _, err := fmt.Fprintln(stdout, entry); err != nil {
				// Early-exit pipeline consumer: silent (see builtinPwd).
				return 1, nil
			}
		}
		return 0, nil
	}

	status := 0
	for _, arg := range args {
		name, value, hasValue := strings.Cut(arg, "=")
		if !exportNamePattern.MatchString(name) {
			fmt.Fprintf(stderr, "export: invalid name: %s\n", arg)
			status = 1
			continue
		}
		if !hasValue {
			// Bare name: no-op success (§All Variables Are Environment
			// Variables).
			continue
		}
		if err := os.Setenv(name, value); err != nil {
			fmt.Fprintf(stderr, "export: %s: %s\n", name, OSReason(err))
			status = 1
		}
	}
	return status, nil
}

// builtinHelp displays help information about available commands.
func builtinHelp(args []string, stdin io.Reader, stdout, stderr io.Writer) (int, error) {
	const headerColor = "\033[1;34m" // Bold blue
	const cmdColor = "\033[1;32m"    // Bold green
	const reset = "\033[0m"

	fmt.Fprintf(stdout, "%sFoundation Shell - Available Commands:%s\n", headerColor, reset)
	fmt.Fprintf(stdout, "%s  cd [dir]%s      - Change directory (no arg: go to home)\n", cmdColor, reset)
	fmt.Fprintf(stdout, "%s  pwd%s           - Print working directory\n", cmdColor, reset)
	fmt.Fprintf(stdout, "%s  exit [code]%s   - Exit the shell (default: last command's status)\n", cmdColor, reset)
	fmt.Fprintf(stdout, "%s  clear%s         - Clear the screen\n", cmdColor, reset)
	fmt.Fprintf(stdout, "%s  help%s          - Display this help message\n", cmdColor, reset)
	fmt.Fprintf(stdout, "%s  export [N=V]%s  - Set environment variables (no args: print environment)\n", cmdColor, reset)
	fmt.Fprintf(stdout, "%s  NAME=VALUE%s    - Standalone assignment: set an environment variable\n", cmdColor, reset)
	fmt.Fprintln(stdout)
	fmt.Fprintf(stdout, "%sOperators:%s\n", headerColor, reset)
	fmt.Fprintf(stdout, "%s  |%s             - Pipe output of one command to another\n", cmdColor, reset)
	fmt.Fprintf(stdout, "%s  &&%s            - Run next command only if the previous succeeds\n", cmdColor, reset)
	fmt.Fprintf(stdout, "%s  ||%s            - Run next command only if the previous fails\n", cmdColor, reset)
	fmt.Fprintf(stdout, "%s  ;%s             - Run commands in sequence unconditionally\n", cmdColor, reset)
	fmt.Fprintln(stdout)
	fmt.Fprintf(stdout, "%sRedirections:%s\n", headerColor, reset)
	fmt.Fprintf(stdout, "%s  < file%s        - Redirect input from file\n", cmdColor, reset)
	fmt.Fprintf(stdout, "%s  > file%s        - Redirect output to file\n", cmdColor, reset)
	fmt.Fprintf(stdout, "%s  >> file%s       - Append output to file\n", cmdColor, reset)
	fmt.Fprintf(stdout, "%s  2> file%s       - Redirect error output to file\n", cmdColor, reset)
	fmt.Fprintf(stdout, "%s  2>> file%s      - Append error output to file\n", cmdColor, reset)
	fmt.Fprintln(stdout)
	fmt.Fprintf(stdout, "%sOther:%s\n", headerColor, reset)
	fmt.Fprintf(stdout, "%s  # comment%s     - Rest of the line is ignored\n", cmdColor, reset)
	fmt.Fprintf(stdout, "%s  $?%s            - Exit status of the previous command\n", cmdColor, reset)

	return 0, nil
}

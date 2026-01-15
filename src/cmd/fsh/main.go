// fsh is a bash-like shell that supports both interactive and non-interactive modes.
// - If a script file is provided as an argument, it executes the script
// - If stdin is a TTY, it runs in interactive REPL mode
// - If stdin is not a TTY (piped input), it reads commands from stdin
package main

import (
	"context"
	"os"

	"foundation-shell/pkg/shell"
)

func main() {
	ctx := context.Background()

	// Check for script file argument
	if len(os.Args) > 1 {
		sh := shell.New(false)
		os.Exit(sh.RunScript(ctx, os.Args[1]))
	}

	// No arguments - detect TTY for interactive mode
	isInteractive := shell.IsTerminal()
	sh := shell.New(isInteractive)
	os.Exit(sh.Run(ctx))
}

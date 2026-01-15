// fsh-exec is a single-shot command executor.
// All arguments are joined together and executed as a single command.
// This is like `bash -c "command"` but without the -c flag.
//
// Usage: fsh-exec echo hello world
// Equivalent to: bash -c "echo hello world"
package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"foundation-shell/pkg/shell"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: fsh-exec <command> [args...]")
		os.Exit(1)
	}

	// Join all arguments into a single command string
	cmdStr := strings.Join(os.Args[1:], " ")

	ctx := context.Background()
	sh := shell.New(false)
	os.Exit(sh.RunCommand(ctx, cmdStr))
}

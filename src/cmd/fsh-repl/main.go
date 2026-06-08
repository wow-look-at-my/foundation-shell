// fsh-repl is an interactive REPL-only shell.
// It always runs in interactive mode, displaying prompts and welcome messages.
package main

import (
	"context"
	"os"

	"foundation-shell/pkg/shell"
)

func main() {
	ctx := context.Background()
	sh := shell.New(true) // Always interactive
	os.Exit(sh.Run(ctx))
}

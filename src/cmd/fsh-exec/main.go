// fsh-exec is a single-shot command executor. Two forms are accepted:
//
//	fsh-exec echo hello world     # all arguments joined into one command line
//	fsh-exec -c 'echo hello'      # the next argument IS the command line
//
// The joined form is like `bash -c "command"` without the -c flag; the -c
// form matches the bash convention. -c without an argument is a usage error
// (exit status 2).
package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"foundation-shell/pkg/shell"
)

func main() {
	args := os.Args[1:]

	var cmdStr string
	switch {
	case len(args) == 0:
		fmt.Fprintln(os.Stderr, "Usage: fsh-exec <command> [args...]")
		fmt.Fprintln(os.Stderr, "       fsh-exec -c <command>")
		os.Exit(1)
	case args[0] == "-c":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "fsh-exec: -c requires an argument")
			fmt.Fprintln(os.Stderr, "Usage: fsh-exec -c <command>")
			os.Exit(2)
		}
		// The -c argument is the whole command line; further arguments are
		// ignored (there are no positional parameters).
		cmdStr = args[1]
	default:
		// Join all arguments into a single command string
		cmdStr = strings.Join(args, " ")
	}

	ctx := context.Background()
	sh := shell.New(false)
	os.Exit(sh.RunCommand(ctx, cmdStr))
}

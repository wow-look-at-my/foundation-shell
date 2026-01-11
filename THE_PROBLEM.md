# The Problem: Bash's `set -e` Is a Lie

## What We Want

When running commands via Claude Code (or any automated tool), we want **fail-fast behavior**: if any command fails, stop execution immediately. Don't continue running subsequent commands that may depend on the failed command's success.

```bash
echo "Starting deployment"
false                        # This fails
echo "Deployment complete"   # This should NOT run
```

## What Bash Promises

Bash has `set -e` (errexit) which supposedly does exactly this:

> Exit immediately if a pipeline, which may consist of a single simple command, a list, or a compound command returns a non-zero status.

## What Bash Actually Does

Bash has a massive exception carved out:

> The shell does not exit if the command that fails is part of the command list immediately following a `while` or `until` keyword, part of the test in an `if` statement, part of any command executed in a `&&` or `||` list **except the command following the final `&&` or `||`**, any command in a pipeline but the last, or if the command's return status is being inverted with `!`.

Translation: **`set -e` is disabled for any command in a `&&` chain except the last one.**

## Why This Kills Us

Claude Code wraps every command like this:

```bash
bash -c 'shopt -u extglob 2>/dev/null || true && eval '\''USER_COMMAND'\'' && pwd -P'
```

The user's command is inside an `eval` which is part of a `&&` list, and it's NOT the final command (`pwd -P` is). Therefore:

**`set -e` will NEVER trigger for the user's command, no matter what we do.**

## What We Tried

1. **Inject `set -e` before the command** - Doesn't work because the `&&` exception applies
2. **Inject `set -e` inside the eval** - Same problem, eval is not the final command
3. **Use `trap 'exit $?' ERR`** - Same limitation, ERR trap has identical `&&` exceptions
4. **Wrap in subshell `( set -e; cmd )`** - Subshells inherit the parent's "errexit disabled" state
5. **Use `bash -c` inside eval** - Would work but creates quoting hell (nested single quotes inside single quotes)

## The Root Cause

Bash was designed for interactive use where you might write:

```bash
make && make install
```

And you don't want `set -e` to exit your shell if `make` fails - you want the `&&` to handle it. This is reasonable for interactive use.

But for **automated command execution**, this behavior is catastrophic. There's no way to opt out of it.

## The Solution

Build a shell that:

1. Implements basic POSIX shell semantics (pipes, redirects, variables, etc.)
2. Has **real** fail-fast behavior with no `&&` exception
3. Can be used as `CLAUDE_CODE_SHELL` for Claude Code

We don't need a full-featured shell. We need a shell that:
- Executes commands
- Handles basic redirects (`>`, `<`, `>>`, `2>&1`)
- Handles pipes (`|`)
- Handles `&&` and `||` (but with proper errexit!)
- Handles `eval` (Claude Code uses it)
- Supports `-c 'command'` invocation
- **Actually stops when a command fails**

## Prior Art

- **Oil Shell / Oils**: Has `strict_errexit` option that fixes this
- **fish**: Different syntax but saner error handling
- **dash**: Same POSIX bugs as bash

None of these are ideal for our use case (minimal, embeddable, Go-based for easy distribution).

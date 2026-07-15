# Foundation Shell

A from-scratch shell implemented in Go, built against a rigorous external
specification. It ships an interactive shell with live syntax highlighting, a
one-shot command executor, and a REPL-only binary.

## Features

- Pipelines (`cmd1 | cmd2 | cmd3`) that terminate when a consumer exits
  early (`yes | head -1` prints `y` and returns immediately)
- Command chaining with `&&`, `||`, and `;`, with full short-circuit
  propagation (`false && a && b` runs nothing)
- Operators work with or without surrounding whitespace (`a|b`, `cmd>file`);
  quoted or escaped operator characters (`'|'`, `\;`, `grep '>' file`) stay
  literal
- I/O redirection (`<`, `>`, `>>`, `2>`, `2>>`); fd duplication (`2>&1`) is
  rejected with a clear error rather than misread as a filename
- Quoting (single and double) with depth-tracked nesting
- Command substitution — both `$(...)` (nestable) and backticks; bodies are
  expanded recursively and output is spliced as data, never re-executed
- `#` comments and newlines as command separators
- Environment variable and tilde expansion, `$?`, `export`, and standalone
  `NAME=VALUE` assignments (every variable is an environment variable)
- Builtins: `cd`, `pwd`, `exit`, `clear`, `help`, `export`; command failures
  never abort a chain (`nosuchcmd || echo fallback` recovers)
- Normative exit codes: 127 not found, 126 not executable, 128+N signal
  deaths, 130 interrupts; `exit` uses a sentinel, never `os.Exit`
- SIGINT/SIGQUIT are forwarded to the running child for every command in
  the session; the shell survives Ctrl+C
- Non-interactive input (piped stdin, script files, `fsh-exec`) is read in
  full and parsed as ONE input: quotes, substitution bodies, and operator
  continuations may span lines, and a parse error rejects the whole input
- Syntax highlighting, both live in the REPL and as a standalone analyzer
- Interactive REPL with readline line editing

## Building

Requires [just](https://github.com/casey/just) >= 1.38 and Go (the correct Go
toolchain is downloaded automatically per `src/go.mod`).

```bash
just build
```

This produces three binaries in `build/`:

| Binary | Purpose |
|--------|---------|
| `fsh` | The shell. Runs a script file argument, reads commands from piped stdin, or starts an interactive REPL on a TTY. |
| `fsh-exec` | One-shot executor. Two forms: `fsh-exec echo hi` (argv joined into one command line) and `fsh-exec -c 'echo hi'` (the next argument is the command line). |
| `fsh-repl` | REPL-only mode: always interactive, with prompt and syntax highlighting. |

## Testing

Requires [bats](https://github.com/bats-core/bats-core) in addition to the
build prerequisites.

```bash
just test   # Go unit tests + BATS integration tests
```

## Specification

The authoritative Foundation Shell specification lives in
[wow-look-at-my/foundation-shell-spec](https://github.com/wow-look-at-my/foundation-shell-spec)
and is mounted here as a git submodule at `spec/`, pinned to the exact spec
commit this implementation targets (`.gitmodules` tracks branch `master`).
`tests/` in this repository is the executable conformance suite (BATS) run by
`just test`.

The submodule is **optional** for building and testing — nothing in the build,
`just test`, or CI reads it. To get the spec content, clone with
`git clone --recurse-submodules`, or after a plain clone run:

```bash
git submodule update --init
```

To bump the pin to a newer spec commit:

```bash
cd spec && git fetch origin && git checkout <new-sha> && cd .. && git add spec && git commit
```

## Project layout

| Path | Contents |
|------|----------|
| `src/cmd/fsh` | Interactive shell entry point |
| `src/cmd/fsh-exec` | One-shot command executor |
| `src/cmd/fsh-repl` | REPL-only entry point |
| `src/internal/lexer` | Tokenizer |
| `src/internal/token` | Token types |
| `src/internal/expander` | Expansion: tilde, environment variables, `$?`, command substitution |
| `src/internal/chain` | Chain (pipeline/operator) execution |
| `src/internal/command` | Command execution and builtins |
| `src/internal/syntax` | Syntax analyzer, highlighter, diagnostics |
| `src/pkg/parser` | Parser (tokens -> command chains) |
| `src/pkg/shell` | Shell orchestration and REPL |
| `tests/` | BATS conformance tests |
| `spec/` | The spec repo as a pinned git submodule (optional; see Specification) |
| `build/` | Build output (`just build`) |

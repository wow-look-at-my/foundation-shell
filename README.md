# Foundation Shell

A from-scratch shell implemented in Go, built against a rigorous external
specification. It ships an interactive shell with live syntax highlighting, a
one-shot command executor, and a REPL-only binary.

## Features

- Pipelines (`cmd1 | cmd2 | cmd3`)
- Command chaining with `&&`, `||`, and `;`
- Operators work with or without surrounding whitespace (`a|b`, `cmd>file`);
  quoted or escaped operator characters (`'|'`, `\;`, `grep '>' file`) stay
  literal
- I/O redirection (`<`, `>`, `>>`, `2>`, `2>>`); fd duplication (`2>&1`) is
  rejected with a clear error rather than misread as a filename
- Quoting (single and double) with depth-tracked nesting
- Command substitution — both `$(...)` (nestable) and backticks; bodies are
  expanded recursively and output is spliced as data, never re-executed
- `#` comments and newlines as command separators
- Environment variable and tilde expansion
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
| `fsh-exec` | One-shot executor: joins its argv into a single command line and runs it (like `bash -c "..."` without the `-c`). |
| `fsh-repl` | REPL-only mode: always interactive, with prompt and syntax highlighting. |

## Testing

Requires [bats](https://github.com/bats-core/bats-core) in addition to the
build prerequisites.

```bash
just test   # Go unit tests + BATS integration tests
```

## Specification

The authoritative Foundation Shell specification lives in
[wow-look-at-my/foundation-shell-spec](https://github.com/wow-look-at-my/foundation-shell-spec).
`spec/tests/` in this repository is the executable conformance suite (BATS)
run by `just test`; the spec prose itself is deliberately not vendored here.

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
| `spec/tests/` | BATS conformance tests |
| `build/` | Build output (`just build`) |

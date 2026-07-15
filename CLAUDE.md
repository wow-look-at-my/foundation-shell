# Foundation Shell

A shell implementation in Go with syntax highlighting, command substitution, and pipelines.

## Build

```bash
just build
```

## Test

```bash
just test  # runs Go unit tests + BATS integration tests
```

- Go unit tests: `src/` directory
- BATS integration tests: `tests/`

## Specification

The authoritative spec is the external repo
[wow-look-at-my/foundation-shell-spec](https://github.com/wow-look-at-my/foundation-shell-spec).
Spec prose is not vendored here; `tests/` is the executable conformance
suite (BATS) run by `just test`.

## Project Structure

- `src/` - Go source code (the Go module root)
  - `cmd/fsh/` - Main shell executable (script file, piped stdin, or interactive REPL)
  - `cmd/fsh-exec/` - One-shot executor. Two forms: `fsh-exec echo hi` (argv joined with spaces into one command line) and `fsh-exec -c 'echo hi'` (the argument after `-c` IS the command line; `-c` without an argument is a usage error, exit 2)
  - `cmd/fsh-repl/` - REPL-only mode
  - `internal/` - Internal packages (lexer, expander, chain, command, syntax, token)
  - `pkg/` - Public packages (parser, shell)
- `tests/` - BATS conformance tests
- `build/` - Build output (`just build`; go-toolchain builds into `src/build/`)

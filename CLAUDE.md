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

- Go unit tests: `go/` directory
- BATS integration tests: `spec/tests/`

## Project Structure

- `go/` - Go source code
  - `cmd/fsh/` - Main shell executable
  - `cmd/fsh-exec/` - Execute single command (`-c` flag)
  - `cmd/fsh-repl/` - REPL-only mode
  - `internal/` - Internal packages (lexer, expander, chain, command, syntax, token)
  - `pkg/` - Public packages (parser, shell)
- `spec/` - Specifications (source of truth)
  - `tests/` - BATS integration tests
- `build/` - Build output

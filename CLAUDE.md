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

The authoritative spec is
[wow-look-at-my/foundation-shell-spec](https://github.com/wow-look-at-my/foundation-shell-spec),
mounted as a git submodule at `spec/` and pinned to the exact spec commit this
implementation targets (`.gitmodules` tracks branch `master`). `tests/` is the
executable conformance suite (BATS) run by `just test`.

The submodule is optional for building/testing — nothing in the build or CI
reads it. Initialize it with `git submodule update --init` (or clone with
`git clone --recurse-submodules`). Bump the pin with:
`cd spec && git fetch origin && git checkout <new-sha> && cd .. && git add spec && git commit`.

**Warnings:**

- CI does NOT fetch the submodule (`actions/checkout` default). Keep it that
  way unless CI starts reading spec content.
- Run `go-toolchain` from `src/` ONLY — an initialized `spec/` submodule
  contains `generator/go.mod`, and a repo-root run would walk into it.

## Project Structure

- `src/` - Go source code (the Go module root)
  - `cmd/fsh/` - Main shell executable (script file, piped stdin, or interactive REPL)
  - `cmd/fsh-exec/` - One-shot executor. Two forms: `fsh-exec echo hi` (argv joined with spaces into one command line) and `fsh-exec -c 'echo hi'` (the argument after `-c` IS the command line; `-c` without an argument is a usage error, exit 2)
  - `cmd/fsh-repl/` - REPL-only mode
  - `internal/` - Internal packages (lexer, expander, chain, command, syntax, token)
  - `pkg/` - Public packages (parser, shell)
- `tests/` - BATS conformance tests
- `docs/ci.md` - CI jobs: why `working-directory: src`, what each permission is for, the one-line comment limit in workflow YAML
- `spec/` - The spec repo as a pinned git submodule (optional; see Specification)
- `build/` - Build output (`just build`; go-toolchain builds into `src/build/`)

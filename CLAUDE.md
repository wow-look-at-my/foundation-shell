# Foundation Shell

A shell implementation in Go with syntax highlighting, command substitution, and pipelines.

## Build

```bash
just build
```

## Test

`just build` and `just test` are the same command: `go-toolchain` in `src/`.
It vets, formats, runs the Go unit tests with coverage, builds into
`src/build/`, and then runs every `src/dats/*.dats` suite against those
binaries. A suite failure fails the build.

- Go unit tests: alongside the code in `src/`
- Conformance suite: `src/dats/*.dats`
  ([dats](https://github.com/wow-look-at-my/dats)), one file per spec area

## Specification

The authoritative spec is
[wow-look-at-my/foundation-shell-spec](https://github.com/wow-look-at-my/foundation-shell-spec),
mounted as a git submodule at `spec/` and pinned to the exact spec commit this
implementation targets (`.gitmodules` tracks branch `master`). `src/dats/` is
the executable conformance suite, run by every build.

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
    - `lexer.Scan` is the ONLY tokenizer. `Project` is the execution view, `Validate` the one implementation of the structural rules, and `syntax.Analyze` classifies the same scan for highlighting — it does not tokenize
  - `pkg/` - Public packages (parser, shell)
  - `dats/` - Conformance suite, one `.dats` file per spec area
- `docs/conformance-suite.md` - dats suite conventions: `$GO_TOOLCHAIN_DATS_BUILD_DIR`, read-only cwd, exact-block assertions, the heredoc guard
- `docs/ci.md` - CI job: why `working-directory: src`, why it installs bubblewrap, what each permission is for, the one-line comment limit in workflow YAML
- `spec/` - The spec repo as a pinned git submodule (optional; see Specification)
- `src/build/` - Build output (`just build` runs go-toolchain, which writes it)

# Foundation Shell Go Conversion Plan

## Overview
Convert foundation-shell from C++ to Go, creating three executables with full feature parity using goroutines for concurrent pipeline execution (~30 lines vs 150+ in C++).

## ⚠️ CRITICAL: Fix Bugs, Don't Replicate Them

**The C++ version never worked properly and failed tests.** Use it as a reference for architecture and design patterns, but:

- **DO NOT blindly copy bugs from the C++ implementation**
- **DO fix any issues you identify in the logic**
- **DO use simpler, more correct Go idioms instead of complex C++ patterns**
- **DO write comprehensive tests to catch bugs the C++ version had**
- **DO verify behavior independently, not just by copying C++**

The C++ version had concurrency issues that prevented it from passing tests. The Go version should be simpler and MORE correct, not a faithful reproduction of broken code.

## Three Executables
1. **fsh-repl**: REPL-only mode (always interactive)
2. **fsh**: Bash-like mode (script file arg OR TTY detection)
3. **fsh-exec**: Single-shot executor (all args become command, no `-c` flag needed)

## Project Structure

First, reorganize the existing C++ code into a `cpp/` subdirectory, then create the Go version alongside it:

```
/mnt/ssdpool/projects/foundation-shell/
├── README.md                  # Top-level readme
├── cpp/                       # C++ version (moved here)
│   ├── src/
│   ├── tests/
│   ├── CMakeLists.txt
│   ├── justfile
│   └── ...
└── go/                        # NEW Go version
    ├── go.mod, go.sum
    ├── justfile               # Go build system (just build, just test)
    ├── cmd/
    │   ├── fsh-repl/main.go       # REPL-only entry point
    │   ├── fsh/main.go            # Bash-like entry point
    │   └── fsh-exec/main.go       # Single-shot entry point
    ├── pkg/
    │   ├── shell/                 # Public Shell API
    │   │   ├── shell.go           # Shell struct, Run(), RunScript(), RunCommand()
    │   │   └── shell_test.go
    │   ├── parser/                # Public Parser API
    │   │   ├── parser.go          # Parse() - string → CommandChain
    │   │   └── parser_test.go
    │   └── executor/              # Public Executor API (if needed)
    └── internal/
        ├── token/                 # Token system
        │   ├── tokentype.go       # TokenType enum (Command, Pipe, And, Or, redirections)
        │   ├── token.go           # Token, ValueToken, OperatorToken
        │   └── token_test.go
        ├── lexer/                 # String tokenization
        │   ├── lexer.go           # Tokenize() with quote/escape handling
        │   └── lexer_test.go
        ├── expander/              # Variable expansion
        │   ├── expander.go        # ExpandTilde(), ExpandEnvironment()
        │   └── expander_test.go
        ├── command/               # Command execution
        │   ├── command.go         # Command struct, Execute()
        │   ├── builtins.go        # cd, pwd, exit, clear, help
        │   └── command_test.go
        ├── chain/                 # Command chain execution
        │   ├── chain.go           # Chain struct, Execute(), executePipeline()
        │   └── chain_test.go
        └── ioutils/               # I/O helpers
            ├── redirect.go        # File redirection handling
            └── pipe.go            # Pipeline utilities
```

## Core Components

### 1. Token System (`internal/token/`)
- **TokenType enum**: Command, CommandArgument, Pipe (|), And (&&), Or (||), RedirectStdIn (<), RedirectStdOut (>), RedirectStdOutAppend (>>), RedirectStdErr (2>), RedirectStdErrAppend (2>>)
- **Token struct**: Type + Value (empty for operators)
- Methods: `IsOperator()`, `IsValue()`

### 2. Lexer (`internal/lexer/`)
- **Tokenize()**: Split input respecting quotes and escapes
- Handle single quotes (literal), double quotes (allow expansion), backslash escapes
- Return `[]TokenContext` with content and quote metadata
- Reference: C++ `tokenizeInput()` in Command.cpp:356-429

### 3. Expander (`internal/expander/`)
- **ExpandTilde()**: `~` → `$HOME`, `~/path` → `$HOME/path`
- **ExpandEnvironment()**: `$VAR` → value, skip in single-quoted strings
- Use `\x01` marker for escaped dollars
- Reference: C++ expansion in Command.cpp:264-322

### 4. Parser (`pkg/parser/`)
- **Parse()**: String → CommandChain
  1. Tokenize with lexer
  2. Expand (tilde + env vars, skip if single-quoted)
  3. Parse tokens (string → Token with type)
  4. Build chain (Token[] → Commands + Operators)
- Track command boundaries for Command vs CommandArgument
- Reference: C++ `parseFromTokens()` in CommandChain.cpp:111-292

### 5. Command (`internal/command/`)
- **Fields**: Args, InputFile, OutputFile, ErrorFile, AppendOutput, AppendError
- **Execute()**: Check builtins → setup I/O → execute external or builtin
- **Builtins**: cd, pwd, exit, clear, help (map-based dispatch)
- **I/O Redirection**: Open files with proper flags (append/truncate)
- Reference: C++ Command.cpp:150-262

### 6. CommandChain (`internal/chain/`)
- **Fields**: Commands, Operators (invariant: len(Operators) == len(Commands) - 1)
- **Execute()**:
  - Single command: direct execution
  - Pipeline: concurrent with goroutines + io.Pipe
  - AND (&&): skip next if previous failed
  - OR (||): skip next if previous succeeded
- **executePipeline()**: ~30 lines with goroutines + WaitGroup
  ```go
  // Create pipes
  pipes := make([]*io.PipeReader, n-1)
  writers := make([]*io.PipeWriter, n-1)
  for i := 0; i < n-1; i++ {
      pipes[i], writers[i] = io.Pipe()
  }

  // Launch commands
  var wg sync.WaitGroup
  for i, cmd := range commands {
      wg.Add(1)
      go func(idx int, c *Command) {
          defer wg.Done()
          // Connect stdin/stdout via pipes
          c.Execute(ctx, stdin, stdout, stderr)
      }(i, cmd)
  }
  wg.Wait()
  ```
- Reference: C++ pipeline logic in CommandChain.cpp:338-408

### 7. Shell (`pkg/shell/`)
- **Fields**: stdin, stdout, stderr, isInteractive, lastExitCode
- **Run()**: REPL loop
  - Prompt with color (green/red based on exit code)
  - Read line → Parse → Execute → update exit code
  - Handle EOF (Ctrl+D), errors
- **RunScript()**: Execute from file
- **RunCommand()**: Execute single command string
- **IsTerminal()**: Check if stdin is TTY (syscall.Tcgetattr)
- Reference: C++ Shell.cpp:76-179

### 8. Entry Points (`cmd/`)

**fsh-repl/main.go**:
```go
sh := shell.New(true)  // Always interactive
os.Exit(sh.Run(ctx))
```

**fsh/main.go**:
```go
if len(os.Args) > 1 {
    sh := shell.New(false)
    os.Exit(sh.RunScript(ctx, os.Args[1]))
}
isInteractive := shell.IsTerminal()
sh := shell.New(isInteractive)
os.Exit(sh.Run(ctx))
```

**fsh-exec/main.go**:
```go
cmdStr := strings.Join(os.Args[1:], " ")
sh := shell.New(false)
os.Exit(sh.RunCommand(ctx, cmdStr))
```

## Implementation Order

### Phase 1: Core Infrastructure (PARALLEL - 4 agents)

These components are completely independent and can be implemented simultaneously:

1. **Agent 1 - Token system** (internal/token/)
   - tokentype.go: TokenType enum with all operators and value types
   - token.go: Token struct, NewValueToken(), NewOperatorToken()
   - token_test.go: Unit tests for validation
   - Reference: src/Token.hpp, src/TokenType.hpp

2. **Agent 2 - Lexer** (internal/lexer/)
   - lexer.go: Tokenize() with quote/escape handling
   - lexer_test.go: Tests for quotes, escapes, whitespace
   - Reference: src/Command.cpp:356-429 (tokenizeInput)

3. **Agent 3 - Expander** (internal/expander/)
   - expander.go: ExpandTilde(), ExpandEnvironment()
   - expander_test.go: Tests for tilde and env var expansion
   - Reference: src/Command.cpp:264-322

4. **Agent 4 - Builtins** (internal/command/builtins.go)
   - Map-based dispatch for cd, pwd, exit, clear, help
   - Each builtin function implementation
   - Reference: src/Command.cpp:29-116 (handleBuiltins)

### Phase 2: Integration Layer (SEQUENTIAL - depends on Phase 1)

5. **Parser** (pkg/parser/) - depends on token system + lexer + expander
   - parser.go: Parse() orchestrates tokenization → expansion → chain building
   - parser_test.go: Tests for full parsing pipeline
   - Reference: src/CommandChain.cpp:111-292

6. **Command** (internal/command/) - depends on builtins
   - command.go: Command struct, Execute(), I/O redirection
   - command_test.go: Tests for execution and redirection
   - Reference: src/Command.cpp:150-262

7. **CommandChain** (internal/chain/) - depends on command + parser
   - chain.go: Chain struct, Execute(), executePipeline() with goroutines
   - chain_test.go: Tests for pipelines, operators
   - Reference: src/CommandChain.cpp:338-408

8. **Shell** (pkg/shell/) - depends on parser + chain
   - shell.go: Shell struct, Run(), RunScript(), RunCommand()
   - shell_test.go: Integration tests
   - Reference: src/Shell.cpp:76-179

### Phase 3: Entry Points & Polish (PARALLEL - 3 agents)

9. **Three main.go files** (cmd/) - can be done in parallel once shell is ready
   - cmd/fsh-repl/main.go
   - cmd/fsh/main.go
   - cmd/fsh-exec/main.go

10. **Tests** - unit tests for each component, integration tests for full shell

## Feature Parity Checklist

- [ ] Operators: `|`, `&&`, `||`
- [ ] Redirections: `<`, `>`, `>>`, `2>`, `2>>`
- [ ] Builtins: cd, pwd, exit, clear, help
- [ ] Quote handling: single (`'`), double (`"`), escape (`\`)
- [ ] Tilde expansion: `~`, `~/path`
- [ ] Environment variables: `$VAR`, `\$VAR` (escaped)
- [ ] Concurrent pipelines with goroutines
- [ ] Interactive vs non-interactive mode
- [ ] Colored prompts based on exit code
- [ ] EOF handling (Ctrl+D)

## Critical C++ Reference Files (after reorganization)

- **cpp/src/Command.cpp** (lines 264-464): bashSplitString logic, expansion algorithms, external process execution
- **cpp/src/CommandChain.cpp** (lines 111-292, 338-408): parseFromTokens, pipeline execution, operator handling
- **cpp/src/Shell.cpp** (lines 76-179): REPL loop, TTY detection, prompt generation
- **cpp/src/Token.hpp, cpp/src/TokenType.hpp**: Token type definitions
- **cpp/tests/ShellTest.cpp, cpp/tests/FeaturesTest.cpp**: Expected behavior and test scenarios

## Build & Test

Create a `justfile` with two commands:

```justfile
# Build all three executables
build:
    @mkdir -p bin
    go build -o bin/fsh-repl ./cmd/fsh-repl
    go build -o bin/fsh ./cmd/fsh
    go build -o bin/fsh-exec ./cmd/fsh-exec
    @echo "Built: bin/fsh-repl, bin/fsh, bin/fsh-exec"

# Run all tests
test:
    go test -v -race ./...
```

Usage:
```bash
just build   # Build all three executables
just test    # Run all tests with race detector
```

## Verification Steps

1. **Basic commands**: `fsh-repl` → `echo hello` → verify output
2. **Pipelines**: `echo test | grep test` → verify concurrent execution
3. **Redirections**: `echo test > out.txt`, `cat < out.txt` → verify file I/O
4. **Operators**: `true && echo yes`, `false || echo no` → verify conditionals
5. **Builtins**: `cd /tmp`, `pwd`, `clear`, `help`, `exit` → verify all work
6. **Quotes**: `echo "hello $USER"`, `echo 'no $expansion'` → verify handling
7. **Expansion**: `echo ~`, `echo $HOME` → verify expansion
8. **fsh modes**:
   - No args + TTY → REPL
   - No args + pipe → read from stdin
   - With script arg → execute script
9. **fsh-exec**: `fsh-exec echo hello world` → verify args become command

## Parallel Agent Execution Strategy

### Initial Setup

**Step 0: Copy this plan to workspace**:
```bash
cp /home/claude/.claude/plans/goofy-finding-sloth.md /mnt/ssdpool/projects/foundation-shell/GO_IMPLEMENTATION_PLAN.md
```

**Step 1: Reorganize C++ code** (move everything into `cpp/` subdirectory):
```bash
cd /mnt/ssdpool/projects/foundation-shell
mkdir cpp
# Move all C++ files into cpp/ (except README.md)
mv src tests cmake extern scripts CMakeLists.txt CMakePresets.json justfile CLAUDE.md TODO.md .clang-format .editorconfig cpp/
# Keep .git, .gitignore, .gitmodules, README.md at root
```

**Step 2: Create Go project structure**:
```bash
cd /mnt/ssdpool/projects/foundation-shell
mkdir -p go/{cmd/{fsh-repl,fsh,fsh-exec},pkg/{shell,parser},internal/{token,lexer,expander,command,chain,ioutils}}
cd go
go mod init foundation-shell
```

**Step 3: Create Go justfile**:
Create `/mnt/ssdpool/projects/foundation-shell/go/justfile` with build and test commands.

### Launch 4 Agents in Parallel (Phase 1)

All 4 agents should be launched **in a single message** to maximize parallelism.

**IMPORTANT FOR ALL AGENTS**: The C++ version had bugs and failed tests. Use it as a reference for concepts, but fix any logic issues you identify. Write correct, idiomatic Go code, not a direct translation of broken C++.

**Agent 1 Prompt (Token System):**
> Implement the token system in Go at `go/internal/token/`. Create:
> 1. tokentype.go: TokenType enum (Command, CommandArgument, Pipe, And, Or, RedirectStdIn, RedirectStdOut, RedirectStdOutAppend, RedirectStdErr, RedirectStdErrAppend) with IsOperator(), IsValue() methods
> 2. token.go: Token struct with Type and Value fields, NewValueToken() and NewOperatorToken() constructors with validation
> 3. token_test.go: Unit tests for all token types and validation
> Reference: /mnt/ssdpool/projects/foundation-shell/cpp/src/Token.hpp and TokenType.hpp

**Agent 2 Prompt (Lexer):**
> Implement the lexer in Go at `go/internal/lexer/`. Create:
> 1. lexer.go: Tokenize() function that splits input respecting quotes (single/double) and backslash escapes, returns []TokenContext with content and quote metadata
> 2. lexer_test.go: Tests for quote handling, escapes, whitespace splitting
> Reference: /mnt/ssdpool/projects/foundation-shell/cpp/src/Command.cpp lines 356-429 (tokenizeInput function)

**Agent 3 Prompt (Expander):**
> Implement the expander in Go at `go/internal/expander/`. Create:
> 1. expander.go: ExpandTilde() for ~ expansion to $HOME, ExpandEnvironment() for $VAR expansion with \x01 marker for escaped dollars
> 2. expander_test.go: Tests for tilde and environment variable expansion, including edge cases
> Reference: /mnt/ssdpool/projects/foundation-shell/cpp/src/Command.cpp lines 264-322 (expansion functions)

**Agent 4 Prompt (Builtins):**
> Implement shell builtins in Go at `go/internal/command/builtins.go`. Create:
> 1. Map-based dispatch for builtins: cd, pwd, exit, clear, help
> 2. Each builtin function with signature: func(cmd *Command, stdin io.Reader, stdout, stderr io.Writer) error
> 3. cd uses os.Chdir, pwd uses os.Getwd, clear prints ANSI escape, help prints comprehensive help
> Reference: /mnt/ssdpool/projects/foundation-shell/cpp/src/Command.cpp lines 29-116 (handleBuiltins)

After these 4 agents complete, proceed sequentially with Phase 2.

## Success Criteria

- ✅ All three executables build and run
- ✅ Full feature parity with C++ version (all operators, redirections, builtins, expansions)
- ✅ Tests pass (unit + integration)
- ✅ Pipelines execute concurrently with goroutines
- ✅ Code is idiomatic Go (passes `go vet`, follows conventions)
- ✅ Simpler than C++ version (no coroutines, no event loops, no complex async)

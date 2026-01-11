# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Foundation Shell is a stateless shell implementation in C++. It doesn't preserve variables between commands but supports environment variables and built-in command handling. The executable is `foundation_shell`.

## Build and Test Commands

```sh
just build                      # Build the project
just run                        # Build and run the shell
just test                       # Run all tests
just test-name "Test name"      # Run specific test by name
just test-group "groupname"     # Run tests by group/tag
just profile "Test name"        # Profile a test (macOS sample / Linux valgrind)
just clean                      # Clean build directory
```

## Architecture

### Core Components

1. **Shell** (`src/Shell.cpp`): Main shell controller. Uses C++20 coroutines via mh_stuff library for async execution. Runs the command loop and manages user interaction.

2. **CommandChain** (`src/CommandChain.cpp`): Parses token sequences into executable command chains. Handles operators (`&&`, `||`, `|`) for conditional execution and piping.

3. **Command** (`src/Command.cpp`): Represents a single command with arguments and I/O redirection. Handles built-in commands (cd, pwd, exit, help) and external process execution.

4. **Token System** (`src/Token.hpp`, `src/TokenType.hpp`): Lexical analysis infrastructure. TokenType enum defines operators (Pipe, And, Or, redirections) and value types (Command, CommandArgument).

### I/O Abstraction Layer

The shell uses mh_stuff's platform-agnostic I/O abstractions:
- `mh::io::source_ptr` / `mh::io::sink_ptr`: Abstract input/output streams
- `mh::io::pipe`: Creates connected source/sink pairs for inter-process communication
- `mh::process`: Process spawning with async wait support

### Async Execution Model

- `mh::task<T>`: Coroutine task type for async operations
- `mh::dispatcher`: Event loop that runs async tasks to completion
- All async functions must have names ending in `Async`
- Never pass by reference to coroutine functions (use value or pointer)

## Code Style Requirements

- **C++23 required** (uses `std::print`, `std::format`)
- Use `std::print()` and `std::format()` instead of iostreams
- `cout`/`cerr`/`cin` are forbidden (poisoned via `LastCppInclude.hpp`)
- **LastCppInclude.hpp must be the last include in every .cpp file**
- Include paths must be relative to project root (no `../` paths)
- Same-directory includes use just the filename
- PascalCase for file names
- TODOs must use `static_assert(false, "TODO: ...")` or throw `std::runtime_error`

## Platform Abstraction

- Keep platform-specific code (file descriptors, etc.) in platform-specific directories
- Main codebase must remain platform-agnostic
- Use mh_stuff abstractions for I/O and process management

## Design Principles

- Stateless by design - no variable preservation between commands
- On any error in the pipeline, throw an exception to return to the steady-state prompt
- Delete backwards-compatibility code; use `[[deprecated]]` only when breaking changes are necessary
- Constructors must throw for invalid states

## Testing

- Uses Catch2 v3.8.0+
- `runShellCommand("command")` returns `ShellOutput` with `stdout_output`, `stderr_output`, `combined_output`
- Do not use `std::string::find()` or `contains()` in tests - use Catch2 matchers like `Catch::Matchers::ContainsSubstring()`
- Tests run with 30-second timeout
- Catch2 docs at `build/_deps/catch2-src/docs/`

## Build System Notes

- Pre-build checks run automatically: shebang fixing, static_assert validation, PascalCase naming, include path validation, LastCppInclude verification
- Scripts are in C# (dotnet run) in `scripts/` directory
- Uses Ninja generator for fast builds
- mh_stuff requires C++20 minimum; main project uses C++23

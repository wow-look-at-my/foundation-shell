# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This repository contains a stateless shell implementation in C++ called the "Foundation Shell" (previously "Base Shell"). It's a simple, clean shell that doesn't preserve variables between commands but does support environment variables and includes built-in command handling.

## Build and Test Commands

This project uses [Justfiles](https://github.com/casey/just) for task automation. All commands should be run from the project root.

```sh
just build                      # Build the project
just run                        # Build and run the shell
just test                       # Run all tests
just test-name "Test name"      # Run specific test by name
just test-group "groupname"     # Run tests in a specific group/tag
just profile "Test name"        # Profile a specific test (macOS sample / Linux valgrind)
just clean                      # Clean build directory
```

## Architecture

### Core Components

1. **Shell**: The main shell controller that runs the command loop and manages user interaction.
   - Implemented in `src/Shell.cpp` and `src/Shell.hpp`
   - Uses C++20 coroutines for asynchronous execution via mh_stuff library
   - Handles input reading, timeout management, and error handling

2. **CommandChain**: Orchestrates execution of command sequences with operators.
   - Implemented in `src/CommandChain.cpp` and `src/CommandChain.hpp`
   - Parses token sequences into executable command chains
   - Handles operators like `&&`, `||`, `|` for conditional execution and piping
   - Returns to Shell's main loop via coroutines

3. **Command**: Represents a single command with arguments and I/O redirection.
   - Implemented in `src/Command.cpp` and `src/Command.hpp`
   - Handles built-in commands (cd, pwd, exit, help, etc.)
   - Manages process execution and I/O redirection
   - Supports background execution with `&`

4. **Token System**: Lexical analysis and parsing infrastructure.
   - `src/Token.hpp`: Base token types (Token, ValueToken)
   - `src/TokenType.hpp`: Enumeration of all token types (operators, values, etc.)
   - Supports parsing of complex command lines with operators and redirections

5. **Configuration and Environment**:
   - `src/Config.hpp`: Configuration management
   - `src/EnvManager.hpp`: Environment variable handling

### Data Flow

1. **Shell** reads user input using async input handling
2. Input string is tokenized into a vector of string tokens
3. **CommandChain** parses tokens into structured command sequences with operators
4. Each **Command** within the chain handles:
   - Built-in command detection and execution
   - External process spawning and management
   - I/O redirection setup
5. Commands execute asynchronously using mh_stuff coroutines
6. Results flow through pipes/redirections as configured
7. Shell displays output and returns to prompt

### Dependencies

- **mh_stuff**: Provides C++20 coroutine infrastructure, async utilities, and cross-platform abstractions
- **nlohmann_json**: JSON handling for configuration
- **Catch2**: Testing framework (v3.4.0+)

**Important**: The project requires C++23 features but mh_stuff uses C++20. Ensure mh_stuff doesn't force explicit `-std=c++20` flags that override the parent project's C++23 requirement.

### I/O Handling

The shell supports:
- Input redirection (`<`)
- Output redirection (`>` and `>>`)
- Error redirection (`2>` and `2>>`)
- Pipes for connecting commands (`|`)
- Background processes (`&`)
- Command chaining (`&&` and `||`)

## Implementation Notes

- The project uses C++23 features (like `std::print`) built on C++20 coroutines for async execution
- Files are stored in the user's home directory with `.foundation_shell_` prefix
- This is a stateless shell by design, so there's no variable preservation
- `foundation_shell` is the executable name (CMakeLists.txt uses this, not base_shell)
- The build system uses Ninja generator for fast incremental builds
- Extensive pre-commit checks: shebang fixing, static_assert validation, naming convention checks, include path validation

## Code Maintenance Guidelines

- Get rid of backwards compatibility. We do not want old dead code hanging around in this project. If you must break something, mark the old version with [[deprecated]].
- Make sure all async functions' names end with Async
- Never pass by reference to coroutine/async functions, its way too dangerous.
- Keep this project platform agnostic. There shouldn't be any mention of file descriptors outside of unix/ directories
- Always use std::print() and std::format() instead of stringstreams or iostreams (note: std::print requires C++23)
- Do not write code that encourages or easily allows the creation of invalid states. For example, an "index" value cannot logically be negative for an array type, so you would use an unsigned integer. For a class constructor, it should throw for any invalid states.
- Format your code properly. If you do not format it to match the codebase style, expect it to get randomly autoformatted out from under you at some point in the future. Then you'll have reread the file and reorient yourself.
- Always use include paths that are relative to the project root. Avoid "../" in include paths at all costs. For files that are in the same directory, just #include the filename with no relative path.
- If you're going to add todos, use `static_assert(false, "TODO: <the todo>");` or `throw new std::runtime_error("TODO: <the todo>");`
- **LastCppInclude.hpp must be the last include in every .cpp file** - This header uses `#pragma GCC poison` to forbid `cout`/`cerr`/`cin` (use std::print instead) and `find`/`contains` in tests (use Catch2 matchers instead)

## Design Principles

- On any kind of error, anywhere along the pipe, we should throw an exception to bail out and return back to our normal steady state prompt (easy to do because we don't have any state!)

## Coding Practices

- We always use exact equality in string comparison checks in tests.

## Scripting

- For scripts, use C# with dotnet run.

## Testing

- Uses Catch2 v3.4.0+ for unit testing
- Tests are located in `tests/` directory
- Review Catch2 documentation: `@build/_deps/catch2-src/docs/matchers.md` and `@build/_deps/catch2-src/docs/assertions.md`
- All tests run with 30-second timeout
- Profiling support available on macOS (sample) and Linux (Valgrind callgrind)
- Use `runShellCommand("command")` from `TestUtils.hpp` to execute commands and get a `ShellOutput` with `stdout_output`, `stderr_output`, and `combined_output` fields
- Do not use `std::string::find()` or `contains()` in tests - use Catch2 matchers like `Catch::Matchers::ContainsSubstring()`

## C++ Standard Requirements

- Main project requires C++23 (for std::print and other features)
- mh_stuff dependency requires C++20 minimum
- Use `target_compile_features(cxx_std_XX)` instead of explicit `-std=` flags to allow proper inheritance
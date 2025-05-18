# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This repository contains a stateless shell implementation in C++ called the "Foundation Shell" (previously "Base Shell"). It's a simple, clean shell that doesn't preserve variables between commands but does support environment variables and includes built-in command handling.

## Build and Test Commands

### Building the Project

```bash
mkdir -p build
cd build
cmake ..
make
```

### Running the Shell

```bash
# From the build directory
./foundation_shell
```

### Running Tests

```bash
# From the build directory
ctest                        # Run all tests
./shell_test                 # Run just the shell tests
./features_test              # Run just the feature tests
```

## Architecture

### Core Components

1. **Shell**: The main shell controller that runs the command loop and manages user interaction.
   - Implemented in `src/shell.cpp` and `include/shell.hpp`
   - Uses C++20 coroutines for asynchronous execution

2. **Command**: Represents a single command with arguments and I/O redirection options.
   - Implemented in `src/command.cpp`, `src/command_async.cpp`, and `include/command.hpp`
   - Handles command execution, built-in commands, and piping

3. **Task**: A C++20 coroutine implementation for asynchronous operations.
   - Implemented in `include/task.hpp`
   - Provides a simple way to write asynchronous code with coroutines

4. **Config**: Manages shell configuration including themes and prompt formatting.
   - Implemented in `src/config.cpp` and `include/config.hpp`
   - Loads/saves configuration from/to a file in the user's home directory

5. **History**: Manages command history.
   - Implemented in `src/history.cpp` and `include/history.hpp`
   - Saves commands to history file and supports history retrieval

6. **Job**: Handles background processes and job control.
   - Implemented in `src/job.cpp` and `include/job.hpp`
   - Supports listing jobs, bringing jobs to foreground/background

7. **Alias**: Manages command aliases.
   - Implemented in `src/alias.cpp` and `include/alias.hpp`
   - Loads aliases from file and expands aliases in commands

### Data Flow

1. User enters command → Shell reads input
2. Input is tokenized and parsed into Command objects with redirections/pipes
3. Commands are executed through the async Task system
4. Output is displayed to user
5. Shell returns to prompt for next command

### I/O Handling

The shell supports:
- Input redirection (`<`)
- Output redirection (`>` and `>>`)
- Error redirection (`2>` and `2>>`) 
- Pipes for connecting commands (`|`)
- Background processes (`&`)
- Command chaining (`&&` and `||`)

## Key Features

- **Stateless Design**: No variable persistence between commands (by design)
- **Built-in Commands**: Includes cd, exit, pwd, clear, history, jobs, fg, bg, etc.
- **Command History**: Stores command history in a file
- **Job Control**: Background processes, jobs listing, foreground/background control
- **Aliases**: Support for command aliases stored in config file
- **Colorful Output**: Theme support and colored prompts
- **Command Suggestions**: Suggests similar commands when a command is not found
- **I/O Redirection**: Full support for input/output/error redirection
- **Pipe Support**: Connect commands with pipes
- **Command Chaining**: Execute commands conditionally with && and ||

## Implementation Notes

- The project uses C++20 features, particularly coroutines for async execution
- Files are stored in the user's home directory with `.foundation_shell_` prefix
- This is a stateless shell by design, so there's no variable preservation
- `foundation_shell` is the executable name (CMakeLists.txt uses this, not base_shell)
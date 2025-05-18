# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This repository contains a stateless shell implementation in C++ called the "Foundation Shell" (previously "Base Shell"). It's a simple, clean shell that doesn't preserve variables between commands but does support environment variables and includes built-in command handling.

## Build and Test Commands

This project uses [Justfiles](https://github.com/casey/just) for simple frequently-used commands.

### Building the Project

```sh
just build
```

### Running the Shell

```sh
just run
```

### Running Tests

```sh
just test
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

## Code Maintenance Guidelines

- Get rid of backwards compatibility. We do not want old dead code hanging around in this project. If you must break something, mark the old version with [[deprecated]].
- Make sure all async functions' names end with Async
- Never pass by reference to coroutine/async functions, its way too dangerous.
- Keep this project platform agnostic. There shouldn't be any mention of file descriptors outside of unix/ directories
- Always use std::print() and std::format() instead of stringstreams
- Do not write code that encourages or easily allows the creation of invalid states. For example, an "index" value cannot logically be negative for an array type, so you would use an unsigned integer. For a class constructor, it should throw for any invalid states.
- Format your code properly. If you do not format it to match the codebase style, expect it to get randomly autoformatted out from under you at some point in the future. Then you'll have reread the file and reorient yourself.
- Always use include paths that are relative to the project root. Avoid "../" in include paths at all costs. For files that are in the same directory, just #include the filename with no relative path.

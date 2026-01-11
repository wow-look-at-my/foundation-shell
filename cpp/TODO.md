# TODO List for Base Shell

This document outlines planned features for the Base Shell project while maintaining its stateless design principles.

## Core Features

- [x] Basic command execution
- [x] Environment variable support
- [x] Built-in `cd` command
- [x] Built-in `exit` command
- [x] Bash-compatible command-line parsing

## Planned Features

### Input/Output Operations
- [x] Input redirection `<`
- [x] Output redirection `>` and `>>`
- [x] Pipe support `|`
- [x] Error redirection `2>` and `2>>`

### UI Improvements
- [x] Colorful output (syntax highlighting, colorized prompt)
- [x] Improved prompt with current directory and exit code
- [x] Command history with file storage (to maintain statelessness)
- [ ] Tab completion for commands and files
- [ ] Line editing capabilities (arrow keys, delete, etc.)
- [x] Clear screen command `clear`

### Command Enhancements
- [x] Additional built-in commands (`pwd`, `help`, etc.)
- [ ] Wildcards and filename expansion
- [x] Aliases loaded from config file (maintaining statelessness)
- [x] Command suggestions for mistyped commands

### Process Management
- [x] Background processes with `&`
- [x] Jobs control (fg, bg, jobs commands)
- [x] Signal handling (Ctrl+C, Ctrl+Z, etc.)
- [x] Process groups management

### Configuration
- [x] Config file for customization
- [x] Themes support
- [x] Custom prompt definition

### Error Handling
- [x] Improved error messages
- [x] Command not found suggestions
- [x] Debug mode with verbose output

## Implementation Notes

All features must maintain the stateless nature of the shell:
- No variable persistence between commands
- Configuration should be loaded from files
- History should be stored in a file
- Command completion data should be derived at runtime, not stored in memory between commands

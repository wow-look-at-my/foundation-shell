# Base Shell

A simple stateless shell implemented in C++. This shell is designed to be stateless, meaning it doesn't preserve variables between commands, though it still handles environment variables.

## Features

- Basic command execution
- Environment variable access
- Built-in `cd` command
- Built-in `exit` command
- Clean, minimal implementation

## Building

This project uses CMake. To build:

```bash
mkdir build
cd build
cmake ..
make
```

## Usage

After building, run the shell:

```bash
./base_shell
```

Enter commands at the prompt:

```
base-shell$ ls -la
base-shell$ echo $HOME
base-shell$ cd /tmp
base-shell$ pwd
base-shell$ exit
```

## Limitations

- This is a stateless shell, so variables are not preserved between commands
- No command history or line editing
- No command completion
- Limited built-in commands

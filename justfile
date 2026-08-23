[private]
help:
    @just --list

# Vet, format, test, build, and run the dats conformance suites.
# go-toolchain does all of it in one pass, into src/build/.
[working-directory: 'src']
build:
    go-toolchain

# The same pipeline: go-toolchain runs the Go unit tests and src/dats/*.dats.
test: build

# Run the shell
run: build
    ./src/build/fsh

# Run the REPL
repl: build
    ./src/build/fsh-repl

# Install to ~/.local/bin (symlinks to src/build/)
install: build
    @mkdir -p ~/.local/bin
    ln -sf "{{justfile_directory()}}/src/build/fsh" ~/.local/bin/fsh
    ln -sf "{{justfile_directory()}}/src/build/fsh-exec" ~/.local/bin/fsh-exec
    ln -sf "{{justfile_directory()}}/src/build/fsh-repl" ~/.local/bin/fsh-repl
    @echo "Installed symlinks to ~/.local/bin"

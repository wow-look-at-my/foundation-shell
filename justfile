# Build all three executables

[private]
help:
    @just --list

setup:
    @mkdir -p build

[working-directory: 'src']
build: setup
    go build -o ../build/fsh-repl ./cmd/fsh-repl
    go build -o ../build/fsh ./cmd/fsh
    go build -o ../build/fsh-exec ./cmd/fsh-exec
    @echo "Built: build/fsh-repl, build/fsh, build/fsh-exec"

# Run all tests (Go unit tests + BATS integration tests)
test: build
    cd src && go test -v -race -cover ./...
    bats tests/

# Run the shell
run: build
    ./build/fsh

# Run the REPL
repl: build
    ./build/fsh-repl

# Format code
[working-directory: 'src']
fmt:
    go fmt ./...

# Lint code
[working-directory: 'src']
lint:
    go vet ./...

# Update dependencies
[working-directory: 'src']
deps:
    go mod tidy

# Install to ~/.local/bin (symlinks to build/)
install: build
    @mkdir -p ~/.local/bin
    ln -sf "{{justfile_directory()}}/build/fsh" ~/.local/bin/fsh
    ln -sf "{{justfile_directory()}}/build/fsh-exec" ~/.local/bin/fsh-exec
    ln -sf "{{justfile_directory()}}/build/fsh-repl" ~/.local/bin/fsh-repl
    @echo "Installed symlinks to ~/.local/bin"

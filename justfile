# Justfile for Foundation Shell

# Build the project
build:
    mkdir -p build
    cd build && cmake -G Ninja .. && cmake --build .

# Run the shell
run: build
    cd build && ./foundation_shell

# Run all tests
test: build
    cd build && ./unit_tests
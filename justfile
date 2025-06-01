# Justfile for Foundation Shell

default:
	@just --list

# Clean build directory
clean:
    rm -rf build

# Build the project
build:
    mkdir -p build
    cd build && cmake -G Ninja -DBUILD_TESTING=ON .. && cmake --build .

# Run the shell
run: build
    cd build && ./foundation_shell

# Run all tests
test: build
    cd build && timeout 30s ./tests/unit_tests

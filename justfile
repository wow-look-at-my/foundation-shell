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

# Run all tests or optionally filter by pattern
test *args="": build
    #!/usr/bin/env bash
    cd build
    timeout --preserve-status 30s ./tests/unit_tests {{args}}

# Run tests by name: just test-name "Multiple pipes work"
test-name name: build
    cd build && timeout --preserve-status 30s ./tests/unit_tests "{{name}}"

# Run tests by group: just test-group piping
test-group group: build
    cd build && timeout --preserve-status 30s ./tests/unit_tests "[{{group}}]"

# Profile a specific test (cross-platform): just profile "Multiple pipes work"
profile name: build
    #!/usr/bin/env bash
    cd build
    PROFILE_FILE="profile_$(echo '{{name}}' | tr ' ' '_')"
    
    if [ "{{os()}}" = "linux" ]; then
        echo "Using Valgrind callgrind on Linux..."
        timeout 60s valgrind --tool=callgrind --dump-instr=yes --collect-jumps=yes \
            --callgrind-out-file="$PROFILE_FILE.callgrind" ./tests/unit_tests "{{name}}" || true
        echo "Profile saved to $PROFILE_FILE.callgrind"
        echo "Analyze with: callgrind_annotate $PROFILE_FILE.callgrind"
        if command -v kcachegrind >/dev/null 2>&1; then
            echo "Or visualize with: kcachegrind $PROFILE_FILE.callgrind &"
        fi
    elif [ "{{os()}}" = "macos" ]; then
        echo "Using macOS sample profiler..."
        echo "Starting test and looking for foundation_shell process..."
        ./tests/unit_tests "{{name}}" &
        TEST_PID=$!
        sleep 2
        
        # Find the foundation_shell process spawned by the test
        SHELL_PID=$(pgrep -f "foundation_shell" | head -1)
        if [ -n "$SHELL_PID" ]; then
            echo "Found foundation_shell process: $SHELL_PID"
            echo "Sampling foundation_shell process for 30 seconds..."
            sample $SHELL_PID 30 -file "$PROFILE_FILE.txt" || true
        else
            echo "foundation_shell process not found, profiling test instead..."
            sample $TEST_PID 30 -file "$PROFILE_FILE.txt" || true
        fi
        
        kill $TEST_PID 2>/dev/null || true
        echo "Profile saved to $PROFILE_FILE.txt"
        echo "View with: cat $PROFILE_FILE.txt"
    else
        echo "Unsupported OS: {{os()}}"
        echo "Falling back to basic timeout test..."
        timeout 30s ./tests/unit_tests "{{name}}" || echo "Test timed out"
    fi

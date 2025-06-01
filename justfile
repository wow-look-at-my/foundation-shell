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
        echo "Starting test and looking for processes..."
        ./tests/unit_tests "{{name}}" &
        TEST_PID=$!
        sleep 2
        
        # Find the foundation_shell process spawned by the test
        SHELL_PID=$(pgrep -f "foundation_shell" | head -1)
        if [ -n "$SHELL_PID" ]; then
            echo "Found foundation_shell process: $SHELL_PID"
            
            # Find all child processes (echo, grep, etc.)
            echo "Looking for child processes..."
            CHILD_PIDS=$(pgrep -P $SHELL_PID 2>/dev/null || true)
            echo "Child processes: $CHILD_PIDS"
            
            # Also look for grep and echo processes that might be related
            GREP_PIDS=$(pgrep -f "grep" 2>/dev/null || true)
            ECHO_PIDS=$(pgrep -f "echo" 2>/dev/null || true)
            
            echo "Found grep processes: $GREP_PIDS"
            echo "Found echo processes: $ECHO_PIDS"
            
            # Sample the shell process
            echo "Sampling foundation_shell process for 30 seconds..."
            sample $SHELL_PID 30 -file "$PROFILE_FILE-shell.txt" || true
            
            # Sample any child processes if they exist
            for child in $CHILD_PIDS $GREP_PIDS $ECHO_PIDS; do
                if [ -n "$child" ] && [ "$child" != "$SHELL_PID" ]; then
                    echo "Sampling child process $child..."
                    sample $child 5 -file "$PROFILE_FILE-child-$child.txt" 2>/dev/null || true
                fi
            done
            
            echo "Profile files saved:"
            ls -la $PROFILE_FILE*.txt 2>/dev/null || echo "No profile files found"
        else
            echo "foundation_shell process not found, profiling test instead..."
            sample $TEST_PID 30 -file "$PROFILE_FILE.txt" || true
        fi
        
        kill $TEST_PID 2>/dev/null || true
        echo "View shell profile with: cat $PROFILE_FILE-shell.txt"
    else
        echo "Unsupported OS: {{os()}}"
        echo "Falling back to basic timeout test..."
        timeout 30s ./tests/unit_tests "{{name}}" || echo "Test timed out"
    fi

#!/usr/bin/env bats

# Pipeline execution conformance tests.
#
# Spec: foundation-shell-spec src/execution.md (§Pipeline Execution,
# §Early Exit Terminates Producers, §Spawn Failures, §Builtins in
# Pipelines) and src/operators.md (§Pipe Operator).
#
# Potentially-hanging cases run under `timeout 10`; exit code 124 would
# mean the pipeline wedged, which the spec forbids.

bats_require_minimum_version 1.5.0

setup() {
    FSH="${FSH:-$BATS_TEST_DIRNAME/../../build/fsh-exec}"

    if [[ ! -x "$FSH" ]]; then
        skip "fsh-exec not found at $FSH - run 'just build' first"
    fi

    cd "$BATS_TEST_TMPDIR"
}

run_fsh() {
    run "$FSH" "$1"
}

# execution.md §Early Exit Terminates Producers (normative): yes | head -1
# prints y and exits immediately, status 0 — it MUST NOT hang, and the
# terminated producer is silent.
@test "yes | head -1 terminates with status 0" {
    run --separate-stderr timeout 10 "$FSH" 'yes | head -1'
    [ "$status" -ne 124 ]
    [ "$status" -eq 0 ]
    [ "$output" = "y" ]
    [ -z "$stderr" ]
}

# execution.md §Early Exit: a finite producer bigger than the pipe buffer.
@test "seq 1 100000 | head -2 terminates" {
    run timeout 10 "$FSH" 'seq 1 100000 | head -2'
    [ "$status" -ne 124 ]
    [ "$status" -eq 0 ]
    [ "${lines[0]}" = "1" ]
    [ "${lines[1]}" = "2" ]
}

# execution.md §Early Exit: a consumer that never reads stdin must not
# wedge the producer.
@test "echo x | pwd does not hang" {
    run timeout 10 "$FSH" 'echo x | pwd'
    [ "$status" -ne 124 ]
    [ "$status" -eq 0 ]
    [ "$output" = "$(pwd)" ]
}

# execution.md §Spawn Failures: the failing consumer reports 127 and the
# pipeline completes.
@test "echo hi | nosuchcmd reports 127" {
    run -127 --separate-stderr timeout 10 "$FSH" 'echo hi | nosuchcmd_fsh_test'
    [[ "$stderr" == *"nosuchcmd_fsh_test: command not found"* ]]
}

# execution.md §Spawn Failures: a failing producer is reported on stderr
# but the exit status is still the rightmost command's (cat's 0).
@test "nosuchcmd | cat reports the error but exits 0" {
    run --separate-stderr timeout 10 "$FSH" 'nosuchcmd_fsh_test | cat'
    [ "$status" -ne 124 ]
    [ "$status" -eq 0 ]
    [[ "$stderr" == *"nosuchcmd_fsh_test: command not found"* ]]
}

# execution.md §Builtins in Pipelines: builtins execute IN-PROCESS even as
# pipeline members — cd mutates the parent shell (deliberate POSIX
# divergence; POSIX shells print the original directory).
@test "builtin cd in a pipeline mutates the shell: cd / | cat ; pwd" {
    run timeout 10 "$FSH" 'cd / | cat ; pwd'
    [ "$status" -ne 124 ]
    [ "$status" -eq 0 ]
    [ "$output" = "/" ]
}

# operators.md §Pipe Operator: the pipeline's exit code is the RIGHTMOST
# command's.
@test "pipeline exit status is the rightmost command's" {
    run_fsh "true | false | sh -c 'exit 5'"
    [ "$status" -eq 5 ]

    run_fsh 'false | true'
    [ "$status" -eq 0 ]

    run_fsh 'true | false'
    [ "$status" -eq 1 ]
}

# operators.md §Pipe Operator: data flows between concurrent commands.
@test "data flows through a pipeline" {
    run_fsh 'echo hello world | tr a-z A-Z'
    [ "$status" -eq 0 ]
    [ "$output" = "HELLO WORLD" ]
}

# operators.md §Multi-Stage Pipelines: longer pipelines work end to end.
@test "multi-stage pipeline" {
    run_fsh 'printf "b\na\nb\n" | sort | uniq'
    [ "$status" -eq 0 ]
    [ "${lines[0]}" = "a" ]
    [ "${lines[1]}" = "b" ]
}

# redirection.md §7.1: explicit output redirection overrides the pipe —
# the downstream command reads EOF, and the file gets the data. The
# disconnected pipe must not hang (execution.md §Early Exit).
@test "output redirection overrides the pipe" {
    run timeout 10 "$FSH" 'echo hi > piped.txt | cat'
    [ "$status" -ne 124 ]
    [ "$status" -eq 0 ]
    [ -z "$output" ]
    [ "$(cat piped.txt)" = "hi" ]
}

# redirection.md §7.4: a mid-pipeline input redirection disconnects that
# command from the pipe; the upstream producer terminates instead of
# blocking.
@test "input redirection inside a pipeline does not wedge the producer" {
    printf 'file-data\n' > side.txt
    run timeout 10 "$FSH" 'yes | cat < side.txt'
    [ "$status" -ne 124 ]
    [ "$status" -eq 0 ]
    [ "$output" = "file-data" ]
}

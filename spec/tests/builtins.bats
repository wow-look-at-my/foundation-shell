#!/usr/bin/env bats

# Builtin command and exit-code conformance tests.
#
# Spec: foundation-shell-spec src/execution.md (§Builtin Commands,
# §Standard Exit Codes, §exit sentinel semantics).

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

# execution.md §cd, §Builtin Commands: builtins run in-process, so cd
# persists for later commands in the same input.
@test "cd changes the directory for the rest of the chain" {
    run_fsh 'cd / ; pwd'
    [ "$status" -eq 0 ]
    [ "$output" = "/" ]
}

# execution.md §cd: failure message and status 1; the chain continues.
@test "cd failure message and status" {
    run --separate-stderr "$FSH" 'cd /nonexistent_fsh_dir'
    [ "$status" -eq 1 ]
    [ "$stderr" = "cd: /nonexistent_fsh_dir: no such file or directory" ]
}

# execution.md §pwd: prints the working directory, status 0.
@test "pwd prints the current directory" {
    run_fsh 'pwd'
    [ "$status" -eq 0 ]
    [ "$output" = "$(pwd)" ]
}

# execution.md §exit: numeric argument becomes the shell's exit status.
@test "exit 7 exits with status 7" {
    run_fsh 'exit 7'
    [ "$status" -eq 7 ]
}

# execution.md §exit: the value is taken modulo 256, non-negative.
@test "exit codes wrap modulo 256" {
    run_fsh 'exit 300'
    [ "$status" -eq 44 ]

    run_fsh 'exit -1'
    [ "$status" -eq 255 ]
}

# execution.md §exit: at top level the chain STOPS — later commands never
# run.
@test "exit stops the sequence: exit 7 ; echo after" {
    run_fsh 'exit 7 ; echo after'
    [ "$status" -eq 7 ]
    [ -z "$output" ]
}

# execution.md §exit: inside a pipeline segment, exit only sets that
# command's status; the shell survives and the chain continues.
@test "exit inside a pipeline does not stop the shell" {
    run_fsh 'exit 7 | cat ; echo after'
    [ "$status" -eq 0 ]
    [ "$output" = "after" ]
}

# execution.md §exit: a non-numeric argument is a usage error (status 2)
# and the shell does NOT exit.
@test "exit with non-numeric argument is a usage error, shell survives" {
    run --separate-stderr "$FSH" 'exit abc'
    [ "$status" -eq 2 ]
    [ "$stderr" = "exit: abc: numeric argument required" ]

    run --separate-stderr "$FSH" 'exit abc ; echo survived'
    [ "$status" -eq 0 ]
    [ "$output" = "survived" ]
}

# execution.md §Standard Exit Codes: command not found is 127.
@test "command not found is 127 with the canonical message" {
    run -127 --separate-stderr "$FSH" 'nosuchcmd_fsh_test'
    [ "$stderr" = "nosuchcmd_fsh_test: command not found" ]
}

# execution.md §Standard Exit Codes: found but not executable is 126.
@test "non-executable file is 126" {
    printf '#!/bin/sh\necho no\n' > notexec.sh
    chmod 644 notexec.sh
    run --separate-stderr "$FSH" './notexec.sh'
    [ "$status" -eq 126 ]
    [ "$stderr" = "./notexec.sh: permission denied" ]
}

# execution.md §Standard Exit Codes: a child killed by signal N reports
# 128+N (SIGKILL = 9 -> 137), never -1 or 255.
@test "signal death reports 128+N" {
    run_fsh "sh -c 'kill -9 \$\$'"
    [ "$status" -eq 137 ]
}

# execution.md §export: sets the variable; children see it (all variables
# are environment variables).
@test "export sets a variable that children see" {
    run_fsh "export FSH_TEST_EXPORT=bar ; sh -c 'echo \$FSH_TEST_EXPORT'"
    [ "$status" -eq 0 ]
    [ "$output" = "bar" ]
}

# execution.md §export: no arguments prints the environment as NAME=value
# lines.
@test "export with no arguments prints the environment" {
    run_fsh 'export FSH_TEST_PRINTED=hello ; export'
    [ "$status" -eq 0 ]
    [[ "$output" == *"FSH_TEST_PRINTED=hello"* ]]
}

# execution.md §export: invalid names error but processing CONTINUES with
# the remaining arguments; final status 1.
@test "export continues past invalid names" {
    run --separate-stderr "$FSH" "export FSH_A=1 1BAD=2 FSH_B=3 ; sh -c 'echo \$FSH_A\$FSH_B'"
    [ "$status" -eq 0 ]
    [ "$output" = "13" ]
    [[ "$stderr" == *"export: invalid name: 1BAD=2"* ]]

    run_fsh 'export 1BAD=2'
    [ "$status" -eq 1 ]
}

# execution.md §Standalone Assignment: NAME=VALUE alone sets the variable;
# children started later in the chain see it.
@test "standalone assignment is visible to children" {
    run_fsh "FSH_TEST_ASSIGN=baz ; sh -c 'echo \$FSH_TEST_ASSIGN'"
    [ "$status" -eq 0 ]
    [ "$output" = "baz" ]
}

# execution.md §Standalone Assignment: a single-quoted word is NOT an
# assignment — it runs a command literally named FOO=bar (127).
@test "single-quoted 'FOO=bar' is a command, not an assignment" {
    run -127 --separate-stderr "$FSH" "'FSH_Q=bar'"
    [ "$stderr" = "FSH_Q=bar: command not found" ]
}

# execution.md §Standalone Assignment: prefix assignments are NOT
# supported — two words make it a command named VAR=x.
@test "prefix assignment VAR=x cmd is not supported" {
    run -127 --separate-stderr "$FSH" 'FSH_P=x true'
    [ "$stderr" = "FSH_P=x: command not found" ]
}

# execution.md §help: must list every builtin (export included), the $?
# parameter, and must NOT advertise a background & operator.
@test "help lists export and \$? and no background operator" {
    run_fsh 'help'
    [ "$status" -eq 0 ]
    [[ "$output" == *"export"* ]]
    [[ "$output" == *'$?'* ]]
    [[ "$output" != *"background"* ]]
}

# execution.md §clear: writes the ANSI clear-screen sequence.
@test "clear writes the ANSI clear sequence" {
    run_fsh 'clear'
    [ "$status" -eq 0 ]
    [ "$output" = $'\033[2J\033[H' ]
}

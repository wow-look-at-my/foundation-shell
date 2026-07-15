#!/usr/bin/env bats

# Whole-input (batch) mode conformance tests: piped stdin, script files,
# and fsh-exec -c.
#
# Spec: foundation-shell-spec src/execution.md (§Non-Interactive Mode,
# §Script Execution, §Single Command Execution) and src/lexer.md (§3.4
# newline separators, §8 comments).

bats_require_minimum_version 1.5.0

setup() {
    FSH="${FSH:-$BATS_TEST_DIRNAME/../../build/fsh-exec}"
    FSH_BIN="${FSH_BIN:-$BATS_TEST_DIRNAME/../../build/fsh}"

    if [[ ! -x "$FSH" || ! -x "$FSH_BIN" ]]; then
        skip "fsh binaries not found - run 'just build' first"
    fi

    cd "$BATS_TEST_TMPDIR"
}

run_fsh() {
    run "$FSH" "$1"
}

# run_fsh_batch pipes a script into the fsh binary (non-interactive
# whole-input mode). Positional args keep the script byte-exact.
run_fsh_batch() {
    run bash -c 'printf %s "$1" | "$2"' _ "$1" "$FSH_BIN"
}

# execution.md §Non-Interactive Mode: ALL of stdin is one input; unquoted
# newlines separate commands.
@test "multi-line stdin executes command by command" {
    run_fsh_batch $'echo a\necho b\n'
    [ "$status" -eq 0 ]
    [ "${lines[0]}" = "a" ]
    [ "${lines[1]}" = "b" ]
}

# lexer.md §8: comments — including the shebang line — are removed by the
# lexer; a script file needs no special first-line handling.
@test "script file with shebang and comments executes" {
    printf '#!/usr/bin/env fsh\n# a comment\necho from-script # trailing note\n' > script.fsh
    run "$FSH_BIN" script.fsh
    [ "$status" -eq 0 ]
    [ "$output" = "from-script" ]
}

# execution.md §Non-Interactive Mode consequence 2: a parse error ANYWHERE
# rejects the whole input — the first line must NOT have run.
@test "parse error anywhere rejects the whole input" {
    run --separate-stderr bash -c 'printf "echo ran > marker.txt\necho \"unclosed\n" | "$1"' _ "$FSH_BIN"
    [ "$status" -eq 1 ]
    [ ! -e marker.txt ]
    [[ "$stderr" == *"error: unclosed double quote"* ]]
}

# lexer.md §3.4: after a chain operator the newline is a CONTINUATION —
# false && <newline> echo behaves exactly like false && echo.
@test "newline after && continues the line" {
    run_fsh_batch $'echo a &&\necho b\n'
    [ "$status" -eq 0 ]
    [ "${lines[0]}" = "a" ]
    [ "${lines[1]}" = "b" ]

    run_fsh_batch $'false &&\necho skipped\n'
    [ "$status" -eq 1 ]
    [ -z "$output" ]
}

# expansion.md §Special Parameters (whole-input consequence): the entire
# input is ONE parse, so every $? expands BEFORE anything runs — it sees
# the pre-input status (0), NOT the false's 1. This is a documented
# deviation from POSIX shells, where the same script prints 1.
@test "whole-input \$? sees the pre-input status" {
    run_fsh_batch $'false\necho [$?]\n'
    [ "$status" -eq 0 ]
    [ "$output" = "[0]" ]
}

# execution.md §Non-Interactive Mode consequence 1: the shell has already
# consumed stdin to EOF, so $(cat) reads nothing — a command can never
# steal script text, and the following line still executes.
@test "commands inherit stdin at EOF: \$(cat) is empty" {
    run --separate-stderr bash -c 'printf "echo got:\$(cat)\necho second\n" | "$1"' _ "$FSH_BIN"
    [ "$status" -eq 0 ]
    [ "${lines[0]}" = "got:" ]
    [ "${lines[1]}" = "second" ]
}

# execution.md §exit: exit stops the sequence; the remaining commands do
# not run and the shell exits with the given status.
@test "exit stops a batch sequence" {
    run_fsh_batch $'echo one\nexit 3\necho two\n'
    [ "$status" -eq 3 ]
    [ "$output" = "one" ]
}

# lexer.md §3.4: blank lines and comment-only lines collapse into a single
# separator.
@test "blank lines and comment lines between commands are harmless" {
    run_fsh_batch $'\n# leading comment\necho a\n\n\n# middle\necho b\n\n'
    [ "$status" -eq 0 ]
    [ "${lines[0]}" = "a" ]
    [ "${lines[1]}" = "b" ]
}

# execution.md §Non-Interactive Mode: quoted strings may span lines in
# whole-input mode (the newline is token content).
@test "quoted newlines are content in batch mode" {
    run_fsh_batch $'echo "a\nb"\n'
    [ "$status" -eq 0 ]
    [ "${lines[0]}" = "a" ]
    [ "${lines[1]}" = "b" ]
}

# CLAUDE.md / execution.md §Single Command Execution: fsh-exec -c takes
# the next argument as THE command line.
@test "fsh-exec -c executes its argument" {
    run "$FSH" -c 'echo hi'
    [ "$status" -eq 0 ]
    [ "$output" = "hi" ]
}

# execution.md §Single Command Execution: the -c string is one input with
# whole-input semantics — embedded newlines separate commands.
@test "fsh-exec -c string may contain newlines" {
    run "$FSH" -c $'echo a\necho b'
    [ "$status" -eq 0 ]
    [ "${lines[0]}" = "a" ]
    [ "${lines[1]}" = "b" ]
}

# fsh-exec usage: -c without an argument is a usage error, exit status 2.
@test "fsh-exec -c without an argument is a usage error" {
    run "$FSH" -c
    [ "$status" -eq 2 ]
}

# execution.md §Script Execution: a missing script file reports the
# canonical message with status 1.
@test "missing script file message" {
    run --separate-stderr "$FSH_BIN" no-such-script.fsh
    [ "$status" -eq 1 ]
    [ "$stderr" = "cannot open script file no-such-script.fsh: no such file or directory" ]
}

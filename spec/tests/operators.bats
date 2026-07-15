#!/usr/bin/env bats

# Chain operator conformance tests.
#
# Spec: foundation-shell-spec src/operators.md (semantics, skip
# propagation, error cases) and src/lexer.md §3.3 (tokenization).

bats_require_minimum_version 1.5.0

setup() {
    FSH="${FSH:-$BATS_TEST_DIRNAME/../../build/fsh-exec}"

    if [[ ! -x "$FSH" ]]; then
        skip "fsh-exec not found at $FSH - run 'just build' first"
    fi
}

run_fsh() {
    run "$FSH" "$1"
}

# operators.md §Execution Algorithm: full skip propagation — a skipped
# segment preserves the status, so BOTH echos are skipped and the chain
# yields false's 1, silently.
@test "skip propagation: false && echo a && echo b is silent, rc 1" {
    run_fsh 'false && echo a && echo b'
    [ "$status" -eq 1 ]
    [ -z "$output" ]
}

# operators.md §Execution Algorithm: the mirror case for ||.
@test "skip propagation: true || echo a || echo b is silent, rc 0" {
    run_fsh 'true || echo a || echo b'
    [ "$status" -eq 0 ]
    [ -z "$output" ]
}

# operators.md §Execution Algorithm: the preserved failure status reaches
# the || after the skipped segment.
@test "false && echo a || echo b runs only b" {
    run_fsh 'false && echo a || echo b'
    [ "$status" -eq 0 ]
    [ "$output" = "b" ]
}

# operators.md §Combining Operators: false || one && two runs both.
@test "false || echo one && echo two prints both" {
    run_fsh 'false || echo one && echo two'
    [ "$status" -eq 0 ]
    [ "${lines[0]}" = "one" ]
    [ "${lines[1]}" = "two" ]
}

# operators.md §Combining Operators: true && false || fallback.
@test "true && false || echo fallback prints fallback" {
    run_fsh 'true && false || echo fallback'
    [ "$status" -eq 0 ]
    [ "$output" = "fallback" ]
}

# operators.md §Exit Code Definition / execution.md §Runtime Failures
# Never Abort the Chain: a spawn failure (127) feeds operator logic like
# any other non-zero exit.
@test "nosuchcmd || echo fallback recovers, rc 0" {
    run --separate-stderr "$FSH" 'nosuchcmd_fsh_test || echo fallback'
    [ "$status" -eq 0 ]
    [ "$output" = "fallback" ]
    [[ "$stderr" == *"nosuchcmd_fsh_test: command not found"* ]]
}

# execution.md §Runtime Failures Never Abort the Chain.
@test "nosuchcmd ; echo next continues" {
    run --separate-stderr "$FSH" 'nosuchcmd_fsh_test ; echo next'
    [ "$status" -eq 0 ]
    [ "$output" = "next" ]
}

# operators.md §Semicolon Operator: the exit code is the LAST command's.
@test "semicolon chain status is the last command's" {
    run_fsh 'true ; false'
    [ "$status" -eq 1 ]

    run_fsh 'false ; true'
    [ "$status" -eq 0 ]
}

# operators.md §Error Cases: a trailing semicolon is VALID.
@test "trailing semicolon is valid" {
    run_fsh 'echo hi ;'
    [ "$status" -eq 0 ]
    [ "$output" = "hi" ]
}

# operators.md §Error Cases: consecutive operators are a parse error;
# nothing on the line executes.
@test "echo a ; ; echo b is a consecutive-operators error" {
    run --separate-stderr "$FSH" 'echo a ; ; echo b'
    [ "$status" -eq 1 ]
    [ -z "$output" ]
    [[ "$stderr" == *"consecutive operators: ; followed by ;"* ]]
}

# operators.md §Error Cases: an operator cannot start the line.
@test "leading operator is a parse error" {
    run --separate-stderr "$FSH" '| cat'
    [ "$status" -eq 1 ]
    [[ "$stderr" == *"unexpected operator at start: |"* ]]
}

# operators.md §Error Cases: chain operators cannot end the line.
@test "trailing pipe is a parse error" {
    run --separate-stderr "$FSH" 'echo hi |'
    [ "$status" -eq 1 ]
    [ -z "$output" ]
    [[ "$stderr" == *"unexpected operator at end"* ]]
}

# lexer.md §3.3.4: quoted operator characters are literal data, never
# operators.
@test "quoted pipes are data" {
    run_fsh "echo '|'"
    [ "$status" -eq 0 ]
    [ "$output" = "|" ]

    run_fsh 'echo "|"'
    [ "$status" -eq 0 ]
    [ "$output" = "|" ]
}

# lexer.md §3.3.4: escaped operator characters are literal.
@test "escaped pipe is data" {
    run_fsh 'echo \|'
    [ "$status" -eq 0 ]
    [ "$output" = "|" ]
}

# lexer.md §3.3.4: a quoted '>' as a grep pattern must NOT become a
# redirection — the input file survives untouched.
@test "grep '>' does not truncate its input file" {
    printf 'a>b\nplain\n' > "$BATS_TEST_TMPDIR/gfile"
    run "$FSH" "grep '>' $BATS_TEST_TMPDIR/gfile"
    [ "$status" -eq 0 ]
    [ "$output" = "a>b" ]
    [ "$(cat "$BATS_TEST_TMPDIR/gfile")" = "$(printf 'a>b\nplain')" ]
}

# lexer.md §3.3: operators do not need whitespace.
@test "operators work without surrounding whitespace" {
    run_fsh 'echo hello|tr a-z A-Z'
    [ "$status" -eq 0 ]
    [ "$output" = "HELLO" ]

    run_fsh 'echo a;echo b'
    [ "$status" -eq 0 ]
    [ "${lines[0]}" = "a" ]
    [ "${lines[1]}" = "b" ]
}

# lexer.md §3.3.2: a lone & is a literal word character, not an operator.
@test "single ampersand is a literal character" {
    run_fsh 'echo a&b'
    [ "$status" -eq 0 ]
    [ "$output" = "a&b" ]
}

# operators.md §Precedence: pipes bind tighter than logical operators.
@test "pipe binds tighter than &&" {
    run_fsh 'echo test | grep test && echo found'
    [ "$status" -eq 0 ]
    [ "${lines[0]}" = "test" ]
    [ "${lines[1]}" = "found" ]
}

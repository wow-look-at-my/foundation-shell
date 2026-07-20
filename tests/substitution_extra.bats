#!/usr/bin/env bats

# Command substitution conformance tests beyond the basics in
# command_substitution.bats: the single-pass security property, quote
# handling in bodies, nested backticks, and failure semantics.
#
# Spec: foundation-shell-spec src/expansion.md (§Command Substitution) and
# src/execution.md (§Command Substitution Executor, §exit).

bats_require_minimum_version 1.5.0

setup() {
    FSH="${FSH:-$BATS_TEST_DIRNAME/../build/fsh-exec}"

    if [[ ! -x "$FSH" ]]; then
        skip "fsh-exec not found at $FSH - run 'just build' first"
    fi

    cd "$BATS_TEST_TMPDIR"
}

run_fsh() {
    run "$FSH" "$1"
}

# expansion.md §Single-Pass Expansion: substitution OUTPUT is never
# re-scanned — data from a file is printed verbatim, NOT executed. This is
# a security property.
@test "substitution output is data: injection payload stays literal" {
    printf '$(echo pwned)\n' > payload.txt
    run_fsh 'echo $(cat payload.txt)'
    [ "$status" -eq 0 ]
    [ "$output" = '$(echo pwned)' ]
}

# expansion.md §Single-Pass Expansion: backtick payloads too.
@test "substitution output is data: backtick payload stays literal" {
    printf '`date`\n' > payload.txt
    run_fsh 'echo $(cat payload.txt)'
    [ "$status" -eq 0 ]
    [ "$output" = '`date`' ]
}

# lexer.md §7.2 / expansion.md §Recursive Execution: quotes in the body
# are honored by the recursive parse — the two spaces survive.
@test "double quotes inside a body preserve spacing" {
    run_fsh 'echo $(echo "a  b")'
    [ "$status" -eq 0 ]
    [ "$output" = "a  b" ]
}

# lexer.md §7.1 rule 4: a quoted ) is body content, not the closing
# delimiter.
@test "quoted close-paren does not close the substitution" {
    run_fsh "echo \$(echo ')')"
    [ "$status" -eq 0 ]
    [ "$output" = ")" ]

    run_fsh 'echo $(echo ")")'
    [ "$status" -eq 0 ]
    [ "$output" = ")" ]
}

# quoting.md §6.3: nested backticks NEST by the quote rule and execute
# recursively — no escaping needed (a deliberate POSIX divergence).
@test "nested backticks execute recursively" {
    run_fsh 'echo `echo `echo hi``'
    [ "$status" -eq 0 ]
    [ "$output" = "hi" ]
}

# expansion.md §Recursive Execution: single quotes INSIDE the body
# suppress expansion within it.
@test "single quotes inside a body suppress nested substitution" {
    run_fsh "echo \$(echo '\$(pwd)')"
    [ "$status" -eq 0 ]
    [ "$output" = '$(pwd)' ]
}

# expansion.md §Failure Semantics: a runtime failure inside the body
# prints to stderr, the substitution yields the captured stdout (empty
# here), and the line CONTINUES — status comes from the outer command.
@test "failing substitution yields empty output, line continues, rc 0" {
    run --separate-stderr "$FSH" 'echo $(nosuchcmd_fsh_test)'
    [ "$status" -eq 0 ]
    [ -z "$output" ]
    [[ "$stderr" == *"nosuchcmd_fsh_test: command not found"* ]]
}

# expansion.md §Failure Semantics: the substituted command's exit status
# is DISCARDED — it never aborts or fails the surrounding line.
@test "substitution exit status is discarded" {
    run_fsh "echo x\$(sh -c 'exit 3')y"
    [ "$status" -eq 0 ]
    [ "$output" = "xy" ]
}

# execution.md §exit: exit inside a substitution stops only the
# substitution's own sequence; the shell survives and both echos run.
@test "exit inside a substitution does not stop the shell" {
    run_fsh 'echo before $(exit 7) ; echo after'
    [ "$status" -eq 0 ]
    [ "${lines[0]}" = "before " ]
    [ "${lines[1]}" = "after" ]
}

# expansion.md §Output Handling: trailing newlines are trimmed, interior
# newlines are preserved.
@test "trailing newlines trimmed, interior newlines kept" {
    run_fsh 'echo x$(printf "a\nb\n\n")y'
    [ "$status" -eq 0 ]
    [ "$output" = "$(printf 'xa\nby')" ]
}

# expansion.md §Quoting and Expansion: double quotes do not suppress
# command substitution.
@test "substitution runs inside double quotes" {
    run_fsh 'echo "count: $(echo 5)"'
    [ "$status" -eq 0 ]
    [ "$output" = "count: 5" ]
}

# expansion.md §Failure Semantics: a body that fails to PARSE fails the
# whole line as a parse error — nothing executes.
@test "substitution body parse error rejects the whole line" {
    run --separate-stderr "$FSH" 'echo $(echo |) ; echo after'
    [ "$status" -eq 1 ]
    [ -z "$output" ]
}

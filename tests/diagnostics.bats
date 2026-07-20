#!/usr/bin/env bats

# Diagnostic output conformance tests: the three-line caret block format
# on the real binaries.
#
# Spec: foundation-shell-spec src/diagnostics.md (§2 format, §5 canonical
# strings, §6 multi-line input, §7 multiple errors, §8 examples).

bats_require_minimum_version 1.5.0

setup() {
    FSH="${FSH:-$BATS_TEST_DIRNAME/../build/fsh-exec}"
    FSH_BIN="${FSH_BIN:-$BATS_TEST_DIRNAME/../build/fsh}"

    if [[ ! -x "$FSH" || ! -x "$FSH_BIN" ]]; then
        skip "fsh binaries not found - run 'just build' first"
    fi
}

# diagnostics.md §2, §8.3: each error is EXACTLY three lines — the input
# line, the caret line, and "error: <message>". The unclosed token spans
# positions 5..11, so six carets start at column 5.
@test "unclosed single quote produces the three-line caret block" {
    run --separate-stderr "$FSH" "echo 'hello"
    [ "$status" -eq 1 ]
    [ -z "$output" ]
    [ "${#stderr_lines[@]}" -eq 3 ]
    [ "${stderr_lines[0]}" = "echo 'hello" ]
    [ "${stderr_lines[1]}" = "     ^^^^^^" ]
    [ "${stderr_lines[2]}" = "error: unclosed single quote" ]
}

# diagnostics.md §8.4: the double-quote variant.
@test "unclosed double quote caret block" {
    run --separate-stderr "$FSH" 'echo "hello'
    [ "$status" -eq 1 ]
    [ "${stderr_lines[0]}" = 'echo "hello' ]
    [ "${stderr_lines[1]}" = "     ^^^^^^" ]
    [ "${stderr_lines[2]}" = "error: unclosed double quote" ]
}

# diagnostics.md §7: multiple errors produce blank-line-separated blocks —
# the spec's normative two-error example: an unclosed substitution
# containing an unclosed quote reports the quote first (innermost), then
# the substitution.
@test "two-error input produces two blank-line-separated blocks" {
    run --separate-stderr "$FSH" -c 'echo $(foo "bar'
    [ "$status" -eq 1 ]
    [ -z "$output" ]
    expected='echo $(foo "bar
     ^^^^^^^^^^
error: unclosed double quote

echo $(foo "bar
     ^^^^^^^^^^
error: unclosed command substitution $(...)'
    [ "$stderr" = "$expected" ]
}

# diagnostics.md §8.1: trailing pipe — one caret under the operator, and
# the analyzer's message carries no operator suffix.
@test "trailing pipe caret block" {
    run --separate-stderr "$FSH" 'echo hello |'
    [ "$status" -eq 1 ]
    [ "${stderr_lines[0]}" = "echo hello |" ]
    [ "${stderr_lines[1]}" = "           ^" ]
    [ "${stderr_lines[2]}" = "error: unexpected operator at end" ]
}

# diagnostics.md §8.1: a two-character trailing operator gets two carets.
@test "trailing && caret block spans the operator" {
    run --separate-stderr "$FSH" 'echo hello &&'
    [ "$status" -eq 1 ]
    [ "${stderr_lines[1]}" = "           ^^" ]
    [ "${stderr_lines[2]}" = "error: unexpected operator at end" ]
}

# diagnostics.md §8.2: missing redirection target points at the operator.
@test "missing redirection target caret block" {
    run --separate-stderr "$FSH" 'echo >'
    [ "$status" -eq 1 ]
    [ "${stderr_lines[0]}" = "echo >" ]
    [ "${stderr_lines[1]}" = "     ^" ]
    [ "${stderr_lines[2]}" = "error: missing redirection target" ]
}

# diagnostics.md §5.1: leading-operator detection is REQUIRED of the
# analyzer, with the operator in the message.
@test "leading operator caret block" {
    run --separate-stderr "$FSH" '| foo'
    [ "$status" -eq 1 ]
    [ "${stderr_lines[0]}" = "| foo" ]
    [ "${stderr_lines[1]}" = "^" ]
    [ "${stderr_lines[2]}" = "error: unexpected operator at start: |" ]
}

# diagnostics.md §6.4: for multi-line input the block shows the LINE
# containing the error, with carets relative to that line's start.
@test "multi-line input reports the offending line" {
    run --separate-stderr bash -c 'printf "echo hello\ncat \"unclosed\n" | "$1"' _ "$FSH_BIN"
    [ "$status" -eq 1 ]
    [ "${stderr_lines[0]}" = 'cat "unclosed' ]
    [ "${stderr_lines[1]}" = "    ^^^^^^^^^" ]
    [ "${stderr_lines[2]}" = "error: unclosed double quote" ]
}

# diagnostics.md §5.1 note / quoting.md §5.5: an EVEN quote count can be
# unclosed — the canonical strings carry no count-based suffix.
@test "even quote count still reports unclosed (no suffix)" {
    run --separate-stderr "$FSH" "echo 'a 'b"
    [ "$status" -eq 1 ]
    [ "${stderr_lines[2]}" = "error: unclosed single quote" ]
}

#!/usr/bin/env bats

# Conformance tests for constructs this shell deliberately does not
# implement. Each one must FAIL LOUDLY at parse time instead of being
# absorbed as ordinary words, which is what makes them safe to type.
#
# Spec: foundation-shell-spec src/lexer.md (§3.3.2 lone &),
# src/redirection.md (§9.5 fd duplication, §9.6 here-documents,
# §9.7 background execution).

bats_require_minimum_version 1.5.0

setup() {
    FSH="${FSH:-$BATS_TEST_DIRNAME/../build/fsh-exec}"

    if [[ ! -x "$FSH" ]]; then
        skip "fsh binaries not found - run 'just build' first"
    fi
}

# The regression this file exists for. Before the guard, `&` became an
# argument, so this ran `bash -c 'sleep 30' &` in the FOREGROUND and the
# shell blocked for the full 30 seconds. A timeout status of 124 means the
# hang is back.
@test "background & fails fast instead of hanging the shell" {
    run --separate-stderr timeout 5 "$FSH" -c "bash -c 'sleep 30' &"
    [ "$status" -eq 1 ]
    [ "${stderr_lines[2]}" = "error: background execution is not supported" ]
}

# The other half of the trap: everything after the & was swallowed too, so
# the second command silently never ran.
@test "background & does not swallow the rest of the line" {
    run --separate-stderr "$FSH" -c 'echo a & echo b'
    [ "$status" -eq 1 ]
    [ -z "$output" ]
    [ "${#stderr_lines[@]}" -eq 3 ]
    [ "${stderr_lines[0]}" = 'echo a & echo b' ]
    [ "${stderr_lines[1]}" = '       ^' ]
    [ "${stderr_lines[2]}" = "error: background execution is not supported" ]
}

@test "here-document is named as unsupported, not as a missing target" {
    run --separate-stderr "$FSH" -c 'cat <<EOF'
    [ "$status" -eq 1 ]
    [ "${stderr_lines[0]}" = 'cat <<EOF' ]
    [ "${stderr_lines[1]}" = '    ^' ]
    [ "${stderr_lines[2]}" = "error: here-documents are not supported" ]
}

# A here-string is three `<`, forming two adjacent pairs. It must report
# once: a doubled diagnostic trains the reader to skim.
@test "here-string reports exactly one error" {
    run --separate-stderr "$FSH" -c 'cat <<<word'
    [ "$status" -eq 1 ]
    [ "${#stderr_lines[@]}" -eq 3 ]
    [ "${stderr_lines[2]}" = "error: here-documents are not supported" ]
}

@test "fd duplication is still rejected" {
    run --separate-stderr "$FSH" -c 'echo hi 2>&1'
    [ "$status" -eq 1 ]
    [[ "$stderr" == *"file descriptor duplication is not supported"* ]]
}

# No false positives: & is a perfectly ordinary word character everywhere
# except as a word of its own.
@test "ampersand inside a word is still literal" {
    run "$FSH" -c 'echo a&b'
    [ "$status" -eq 0 ]
    [ "$output" = "a&b" ]
}

@test "quoted ampersand is still literal" {
    run "$FSH" -c 'echo "a & b"'
    [ "$status" -eq 0 ]
    [ "$output" = "a & b" ]
}

@test "escaped ampersand is still literal" {
    run "$FSH" -c 'echo x\&'
    [ "$status" -eq 0 ]
    [ "$output" = "x&" ]
}

@test "url query string survives" {
    run "$FSH" -c 'echo "http://h/p?a=1&b=2"'
    [ "$status" -eq 0 ]
    [ "$output" = "http://h/p?a=1&b=2" ]
}

@test "&& is unaffected" {
    run "$FSH" -c 'true && echo ok'
    [ "$status" -eq 0 ]
    [ "$output" = "ok" ]
}

@test "single < input redirection is unaffected" {
    printf 'payload\n' > "$BATS_TEST_TMPDIR/in.txt"
    run "$FSH" -c "cat < $BATS_TEST_TMPDIR/in.txt"
    [ "$status" -eq 0 ]
    [ "$output" = "payload" ]
}

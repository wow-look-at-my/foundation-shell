#!/usr/bin/env bats

# Quoting conformance tests.
#
# Spec: foundation-shell-spec src/quoting.md (depth-tracked nesting,
# suppression flags) and src/lexer.md (tokenization). Each test cites the
# section its expectation comes from.

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

# quoting.md §6.1: whitespace-delimited same-type quotes NEST; the nested
# pair stays literally in the argument, only the outermost pair is stripped.
@test "nested single quotes stay literal: 'outer 'inner' end'" {
    run_fsh "echo 'outer 'inner' end'"
    [ "$status" -eq 0 ]
    [ "$output" = "outer 'inner' end" ]
}

# quoting.md §6.1: nesting is not limited to one level.
@test "multi-level single-quote nesting" {
    run_fsh "echo 'l1 'l2 'l3' l2' l1'"
    [ "$status" -eq 0 ]
    [ "$output" = "l1 'l2 'l3' l2' l1" ]
}

# quoting.md §6.2: the same rule for double quotes.
@test "nested double quotes stay literal" {
    run_fsh 'echo "outer "inner" end"'
    [ "$status" -eq 0 ]
    [ "$output" = 'outer "inner" end' ]
}

# quoting.md §6.2: double-quote semantics inside a nested region are
# unchanged — $VAR still expands; the nested quotes print.
@test "expansion still runs inside nested double quotes" {
    run env HOME=/home/testuser "$FSH" 'echo "path "$HOME" here"'
    [ "$status" -eq 0 ]
    [ "$output" = 'path "/home/testuser" here' ]
}

# quoting.md §6.1.1 (POSIX-identical anchors): attached closers close, so
# 'a' 'b' is two arguments.
@test "two adjacent quoted words are two arguments" {
    run_fsh "printf '[%s]' 'a' 'b'"
    [ "$status" -eq 0 ]
    [ "$output" = "[a][b]" ]
}

# quoting.md §6.1.1: 'a'b concatenates, 'a''b' concatenates.
@test "POSIX-style quote concatenation is unchanged" {
    run_fsh "echo 'a'b"
    [ "$status" -eq 0 ]
    [ "$output" = "ab" ]

    run_fsh "echo 'a''b'"
    [ "$status" -eq 0 ]
    [ "$output" = "ab" ]
}

# quoting.md §6.1.1 / §2.3: the close-escape-reopen dance for attached
# apostrophes works exactly as in POSIX.
@test "escaped-apostrophe dance: 'don'\\''t'" {
    run_fsh "echo 'don'\\''t'"
    [ "$status" -eq 0 ]
    [ "$output" = "don't" ]
}

# quoting.md §12.6 / lexer.md §12.6 (flagship interleave): every interior
# quote closes, the segments concatenate, and WasSingleQuoted suppresses
# expansion — the argument is exactly:  Hello, "$USER"!  (UNexpanded).
@test "flagship interleave stays unexpanded" {
    cmd=$(printf 'echo "Hello, "%s"$USER"%s"!"' \'\"\' \'\"\')
    run "$FSH" "$cmd"
    [ "$status" -eq 0 ]
    [ "$output" = 'Hello, "$USER"!' ]
}

# quoting.md §13.3: when the nesting rule reads "nest" and depth never
# returns to 0, the input fails LOUDLY with the canonical unclosed error.
@test "nesting cost is a loud error: 'hello 'world" {
    run --separate-stderr "$FSH" "echo 'hello 'world"
    [ "$status" -eq 1 ]
    [ -z "$output" ]
    [[ "$stderr" == *"error: unclosed single quote"* ]]
}

# quoting.md §13.3: the POSIX close-then-concatenate idiom is an unclosed
# double quote here (the second " NESTS).
@test "nesting cost is a loud error: \"Total: \"\$N" {
    run --separate-stderr "$FSH" 'echo "Total: "$N'
    [ "$status" -eq 1 ]
    [ -z "$output" ]
    [[ "$stderr" == *"error: unclosed double quote"* ]]
}

# lexer.md §4.5: empty quoted strings ARE arguments.
@test "empty quotes produce an empty argument" {
    run_fsh "printf '[%s]' ''"
    [ "$status" -eq 0 ]
    [ "$output" = "[]" ]

    run_fsh 'printf "[%s]" a "" b'
    [ "$status" -eq 0 ]
    [ "$output" = "[a][][b]" ]
}

# quoting.md §2.5: a single-quoted part ANYWHERE suppresses ALL expansion
# for the whole token (whole-token granularity).
@test "whole-token suppression: 'a'\$HOME stays literal" {
    run_fsh "echo 'a'\$HOME"
    [ "$status" -eq 0 ]
    [ "$output" = 'a$HOME' ]
}

# lexer.md §3.2: the per-word flags reset at every boundary — an empty
# single-quoted word must not suppress the NEXT word's expansion.
@test "quote flags do not leak across words" {
    run env HOME=/home/testuser "$FSH" "echo '' \$HOME"
    [ "$status" -eq 0 ]
    [ "$output" = " /home/testuser" ]
}

# expansion.md §Tilde Expansion: any quoting suppresses tilde expansion.
@test "quoted tilde is literal" {
    run_fsh 'echo "~"'
    [ "$status" -eq 0 ]
    [ "$output" = "~" ]

    run_fsh "echo '~'"
    [ "$status" -eq 0 ]
    [ "$output" = "~" ]
}

# expansion.md §Tilde Expansion: bare unquoted ~ expands to $HOME.
@test "unquoted tilde expands to HOME" {
    run env HOME=/home/testuser "$FSH" 'echo ~'
    [ "$status" -eq 0 ]
    [ "$output" = "/home/testuser" ]

    run env HOME=/home/testuser "$FSH" 'echo ~/docs'
    [ "$status" -eq 0 ]
    [ "$output" = "/home/testuser/docs" ]
}

# quoting.md §6.4 / §5.3: different-type quote characters inside an open
# region are literal content.
@test "mixed quote types are literal inside each other" {
    run_fsh "echo 'say \"hello\"'"
    [ "$status" -eq 0 ]
    [ "$output" = 'say "hello"' ]

    run_fsh 'echo "it'"'"'s fine"'
    [ "$status" -eq 0 ]
    [ "$output" = "it's fine" ]
}

# quoting.md §3.4: escaped double quotes inside double quotes.
@test "escaped quotes inside double quotes" {
    run_fsh 'echo "say \"hi\""'
    [ "$status" -eq 0 ]
    [ "$output" = 'say "hi"' ]
}

# quoting.md §2.1: no escape processing inside single quotes.
@test "backslash is literal inside single quotes" {
    run_fsh "echo 'a\\nb'"
    [ "$status" -eq 0 ]
    [ "$output" = 'a\nb' ]
}

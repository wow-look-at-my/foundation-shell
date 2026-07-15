#!/usr/bin/env bats

# Expansion pipeline conformance tests.
#
# Spec: foundation-shell-spec src/expansion.md.

bats_require_minimum_version 1.5.0

setup() {
    FSH="${FSH:-$BATS_TEST_DIRNAME/../build/fsh-exec}"

    if [[ ! -x "$FSH" ]]; then
        skip "fsh-exec not found at $FSH - run 'just build' first"
    fi
}

run_fsh() {
    run "$FSH" "$1"
}

# expansion.md §Word Splitting: an unquoted token that expands to empty
# stays in the argument list as an EMPTY argument (POSIX drops it).
@test "unset variable stays as an empty argument" {
    run env -u FSH_UNSET_VAR "$FSH" 'printf [%s] a $FSH_UNSET_VAR b'
    [ "$status" -eq 0 ]
    [ "$output" = "[a][][b]" ]
}

# expansion.md §Word Splitting: NO post-expansion word splitting — a value
# containing whitespace remains ONE argument.
@test "no post-expansion word splitting" {
    run env X='a b' "$FSH" 'printf [%s] $X'
    [ "$status" -eq 0 ]
    [ "$output" = "[a b]" ]
}

# expansion.md §Special Parameters: $? starts at 0; the whole input is one
# parse, so a fresh fsh-exec sees the pre-input status.
@test "\$? expands to 0 in a fresh shell" {
    run_fsh 'echo $?'
    [ "$status" -eq 0 ]
    [ "$output" = "0" ]
}

# expansion.md §Variable Expansion rule 4: the braced body is a VERBATIM
# environment lookup, not the special parameter — ${?} is normally empty.
@test "\${?} is a verbatim environment lookup (empty)" {
    run_fsh 'printf [%s] ${?}'
    [ "$status" -eq 0 ]
    [ "$output" = "[]" ]
}

# expansion.md §NOT Supported: other POSIX special parameters stay
# literal.
@test "\$\$ and \$1 are literal" {
    run_fsh 'echo $$'
    [ "$status" -eq 0 ]
    [ "$output" = '$$' ]

    run_fsh 'echo $1'
    [ "$status" -eq 0 ]
    [ "$output" = '$1' ]
}

# expansion.md §Escaped Dollar Signs: \$ suppresses expansion via the
# escape marker; the literal $ survives.
@test "escaped dollar is literal" {
    run_fsh 'echo \$HOME'
    [ "$status" -eq 0 ]
    [ "$output" = '$HOME' ]
}

# expansion.md §Expansion Rules rule 1: name scanning is byte-wise ASCII —
# a non-ASCII character ends the name instead of extending it.
@test "variable names are ASCII-only" {
    run env FSH_A=x "$FSH" 'echo $FSH_Aé'
    [ "$status" -eq 0 ]
    [ "$output" = "xé" ]
}

# expansion.md §Expansion Rules rule 2: greedy matching takes the longest
# valid name.
@test "greedy name matching" {
    run env A=a AB=ab ABC=abc "$FSH" 'echo $ABC'
    [ "$status" -eq 0 ]
    [ "$output" = "abc" ]
}

# expansion.md §Examples: braces limit the name explicitly.
@test "braces limit the variable name" {
    run env A=a "$FSH" 'echo ${A}BC'
    [ "$status" -eq 0 ]
    [ "$output" = "aBC" ]
}

# expansion.md §Expansion Rules rules 5-6: ${} stays literal; an unclosed
# ${ is literal text.
@test "empty and unclosed braces stay literal" {
    run_fsh 'echo ${}'
    [ "$status" -eq 0 ]
    [ "$output" = '${}' ]

    run_fsh 'echo ${FSH_UNCLOSED'
    [ "$status" -eq 0 ]
    [ "$output" = '${FSH_UNCLOSED' ]
}

# expansion.md §Spliced Values Are Protected: a variable value containing
# substitution syntax is DATA — never executed, never re-expanded.
@test "variable values are data, not code" {
    run env X='$(echo pwned)' "$FSH" 'echo $X'
    [ "$status" -eq 0 ]
    [ "$output" = '$(echo pwned)' ]

    run env Y='`date`' "$FSH" 'echo $Y'
    [ "$status" -eq 0 ]
    [ "$output" = '`date`' ]

    run env Z='$HOME' "$FSH" 'echo $Z'
    [ "$status" -eq 0 ]
    [ "$output" = '$HOME' ]
}

# expansion.md §Non-Existent Variables: missing variables vanish inside
# larger words.
@test "missing braced variable expands to nothing inside a word" {
    run env -u MISSING_FSH_VAR "$FSH" 'echo prefix${MISSING_FSH_VAR}end'
    [ "$status" -eq 0 ]
    [ "$output" = "prefixend" ]
}

# expansion.md §Tilde Expansion: ~user is NOT supported; a tilde not at
# the start of the token does not expand.
@test "tilde expansion shape rules" {
    run_fsh 'echo ~user'
    [ "$status" -eq 0 ]
    [ "$output" = "~user" ]

    run_fsh 'echo /path/~'
    [ "$status" -eq 0 ]
    [ "$output" = "/path/~" ]
}

# expansion.md §Tilde Expansion (whole-token granularity): quoting ANY
# part of the token suppresses tilde expansion for the whole token.
@test "tilde suppression is whole-token" {
    run env HOME=/home/testuser "$FSH" 'echo ~/"docs"'
    [ "$status" -eq 0 ]
    [ "$output" = "~/docs" ]
}

# expansion.md §Quoting and Expansion: double quotes allow variable
# expansion; single quotes suppress it.
@test "quote flags gate variable expansion" {
    run env FSH_V=hello "$FSH" 'echo "$FSH_V"'
    [ "$status" -eq 0 ]
    [ "$output" = "hello" ]

    run env FSH_V=hello "$FSH" "echo '\$FSH_V'"
    [ "$status" -eq 0 ]
    [ "$output" = '$FSH_V' ]
}

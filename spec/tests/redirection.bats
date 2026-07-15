#!/usr/bin/env bats

# I/O redirection conformance tests.
#
# Spec: foundation-shell-spec src/redirection.md. Each test cites its
# section.

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

# redirection.md §3.2: > truncates.
@test "> truncates the target file" {
    run_fsh 'echo first > out.txt'
    [ "$status" -eq 0 ]
    run_fsh 'echo second > out.txt'
    [ "$status" -eq 0 ]
    [ "$(cat out.txt)" = "second" ]
}

# redirection.md §3.3: >> appends.
@test ">> appends to the target file" {
    run_fsh 'echo line1 > log.txt'
    run_fsh 'echo line2 >> log.txt'
    [ "$status" -eq 0 ]
    [ "$(cat log.txt)" = "$(printf 'line1\nline2')" ]
}

# redirection.md §3.1: < replaces stdin with the file.
@test "< redirects input" {
    printf 'from-file\n' > in.txt
    run_fsh 'cat < in.txt'
    [ "$status" -eq 0 ]
    [ "$output" = "from-file" ]
}

# redirection.md §3.4: 2> captures stderr, leaving stdout alone.
@test "2> captures stderr" {
    run_fsh 'ls /nonexistent_fsh_dir 2> err.txt'
    [ "$status" -ne 0 ]
    [ -z "$output" ]
    [ -s err.txt ]
}

# redirection.md §3.5: 2>> appends stderr.
@test "2>> appends stderr" {
    run_fsh 'ls /nonexistent_fsh_dir 2> err.txt'
    run_fsh 'ls /nonexistent_fsh_dir 2>> err.txt'
    [ "$(wc -l < err.txt)" -eq 2 ]
}

# redirection.md §5.3: last-wins at PARSE time — the earlier target is
# never opened, so file1 is NOT created (a documented POSIX deviation;
# bash would create and truncate it).
@test "last redirection wins; earlier target never created" {
    run_fsh 'echo test > f1.txt > f2.txt'
    [ "$status" -eq 0 ]
    [ ! -e f1.txt ]
    [ "$(cat f2.txt)" = "test" ]
}

# redirection.md §9.4: a target that expands to empty is a loud parse
# error; nothing on the line executes.
@test "empty redirection target is a parse error" {
    run --separate-stderr env -u UNSET_FSH_VAR "$FSH" 'echo hi > $UNSET_FSH_VAR'
    [ "$status" -eq 1 ]
    [ -z "$output" ]
    [[ "$stderr" == *"empty redirection target"* ]]
}

# redirection.md §9.5: fd duplication is guarded with a dedicated error.
@test "2>&1 is rejected with the fd-duplication error" {
    run --separate-stderr "$FSH" 'echo hi 2>&1'
    [ "$status" -eq 1 ]
    [ -z "$output" ]
    [[ "$stderr" == *"file descriptor duplication is not supported: &1"* ]]
}

# redirection.md §9.5: >&2 is the same guard.
@test ">&2 is rejected with the fd-duplication error" {
    run --separate-stderr "$FSH" 'echo hi >&2'
    [ "$status" -eq 1 ]
    [[ "$stderr" == *"file descriptor duplication is not supported: &2"* ]]
}

# redirection.md §9.5: a QUOTED target is a legitimate filename.
@test "quoted '&1' target writes a file literally named &1" {
    run_fsh "echo hi > '&1'"
    [ "$status" -eq 0 ]
    [ "$(cat '&1')" = "hi" ]
}

# redirection.md §9.5: the guard inspects the PRE-expansion token — a
# target that becomes &1 only through expansion is a legal filename.
@test "expansion-produced &1 target is a legal filename" {
    run env X='&1' "$FSH" 'echo hi > $X'
    [ "$status" -eq 0 ]
    [ "$(cat '&1')" = "hi" ]
}

# redirection.md §9.1: runtime open failures print EXACTLY this message
# (no wrapper prefix, filename exactly once) and fail only the command.
@test "input open failure message is canonical" {
    run --separate-stderr "$FSH" 'cat < nonexistent.txt'
    [ "$status" -eq 1 ]
    [ "$stderr" = "cannot open input file nonexistent.txt: no such file or directory" ]
}

# redirection.md §9.1: the chain CONTINUES after an open failure.
@test "chain continues after input open failure" {
    run --separate-stderr "$FSH" 'cat < nonexistent.txt || echo recovered'
    [ "$status" -eq 0 ]
    [ "$output" = "recovered" ]
    [[ "$stderr" == *"cannot open input file nonexistent.txt: no such file or directory"* ]]
}

# redirection.md §8.3 + §9.1: missing parent directory is an output open
# failure with the canonical message.
@test "output open failure message is canonical" {
    run --separate-stderr "$FSH" 'echo x > missingdir/file.txt'
    [ "$status" -eq 1 ]
    [ "$stderr" = "cannot open output file missingdir/file.txt: no such file or directory" ]
}

# redirection.md §9.3: a redirection with no target is a parse error.
@test "missing redirection target is a parse error" {
    run --separate-stderr "$FSH" 'echo >'
    [ "$status" -eq 1 ]
    [[ "$stderr" == *"missing redirection target"* ]]
}

# redirection.md §9.3: an operator instead of a filename.
@test "redirection followed by an operator is a parse error" {
    run --separate-stderr "$FSH" 'echo > | cat'
    [ "$status" -eq 1 ]
    [[ "$stderr" == *"missing redirection target: > followed by operator |"* ]]
}

# lexer.md §3.4: a redirection cannot be continued across a newline — the
# separator is NOT suppressed after redirection operators, so the parser
# rejects the dangling redirection instead of silently redirecting into
# the next line's first word.
@test "newline after a redirection is a parse error, not continuation" {
    run --separate-stderr "$FSH" -c $'echo hi >\nout.txt'
    [ "$status" -eq 1 ]
    [[ "$stderr" == *"missing redirection target: > followed by operator ;"* ]]
    [ ! -e out.txt ]
}

# redirection.md §5.1: redirections may appear anywhere in the command.
@test "redirection position is independent" {
    printf 'data\n' > in.txt
    run_fsh '< in.txt cat'
    [ "$status" -eq 0 ]
    [ "$output" = "data" ]
}

# redirection.md §12.3: builtins respect redirections.
@test "builtin pwd respects output redirection" {
    run_fsh 'pwd > where.txt'
    [ "$status" -eq 0 ]
    [ "$(cat where.txt)" = "$(pwd)" ]
}

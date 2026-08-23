# The dats conformance suite

`src/dats/*.dats` is the executable conformance suite. One file covers one
spec area, and each test's comment cites the section it comes from.
[dats](https://github.com/wow-look-at-my/dats) is the runner;
`go-toolchain` fetches it and runs every suite after each build, so a suite
failure fails the build.

Run the whole thing with `just test`, or one file with `dats src/dats/x.dats`
from `src/`. `dats syntax src/dats/*.dats` parses without executing.

## The working directory is read-only

Every command runs through `bash -c` in the working directory of the `dats`
invocation, which is `src/` — so the binary under test is `build/fsh-exec` (or
`build/fsh`). The sandbox mounts that directory READ-ONLY. A test whose shell
must create a file writes to a `{outputs.NAME}` path, which resolves inside the
sandbox's writable temp directory, and asserts on it under `outputs.files`.

Two redirection tests need a writable *relative* target, because the target is
the thing under test (`> '&1'`). Those capture the binary's absolute path first
and `cd "$(mktemp -d)"` — the private `/tmp` is writable:

```yaml
cmd: "F=\"$PWD/build/fsh-exec\"; cd \"$(mktemp -d)\" && \"$F\" \"echo hi > '&1'\" && cat '&1'"
```

## Writing assertions

A pattern LIST is literal substrings. A line-number MAP is RE2 regex,
0-indexed, searched unanchored within that line. The two forms cannot be mixed
in one block.

Pin a whole line with `^\Q...\E$`. `\Q...\E` quotes the text between it, so an
expected value full of `$`, `[`, `|` and `(` needs no escaping at all. In a
double-quoted YAML scalar write `\\Q` and `\\E`, since YAML eats one backslash.

Assert that a stream is EMPTY with a negated line 0: `!stdout: {0: "."}`. A
line that does not exist cannot match, so the assertion passes only when there
is no line 0.

Assert an exact line COUNT — the diagnostics spec fixes several blocks at
exactly three lines — by capturing the stream to a file and anchoring the whole
match. `\A` and `\z` bracket the file, a YAML `\n` is a real newline, and each
line is a `\Q...\E` literal:

```yaml
cmd: "build/fsh-exec -c 'echo a & echo b' 2> {outputs.err.txt}"
outputs:
    files:
        err.txt:
            match:
                - "\\A\\Qecho a & echo b\\E\n\\Q       ^\\E\n\\Qerror: background execution is not supported\\E\n\\z"
```

Go's `$` matches end of text, NOT before a trailing newline. `^hi$` therefore
does not match a file holding `hi\n`; write `\A\Qhi\E\n\z`.

## Two things dats will not let you write

`.dats` files are parsed by yaml-fixed: indentation is TABS only. Spaces align
past a tab, which is what the two spaces after a sequence item's `- ` are for.

dats rejects a shell heredoc (`<<WORD`) or herestring (`<<<`) anywhere in `cmd`
at parse time — including one quoted as an argument to another program. The
tests that feed those to `fsh` keep the line in an `inputs.files` fixture and
run `build/fsh-exec -c "$(cat {inputs.line.fsh})"`. The flagship interleave
test in `quoting.dats` uses the same fixture trick for a different reason: its
own quoting leaves no readable way to spell it inline.

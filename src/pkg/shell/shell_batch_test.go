package shell

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runBatch runs input through a non-interactive shell (whole-input mode).
func runBatch(t *testing.T, input string) (exitCode int, stdout, stderr string) {
	t.Helper()
	var out, errBuf bytes.Buffer
	sh := NewWithIO(strings.NewReader(input), &out, &errBuf, false)
	exitCode = sh.Run(context.Background())
	return exitCode, out.String(), errBuf.String()
}

// Newlines separate commands within the single whole-input parse.
func TestBatch_NewlineSeparatedCommands(t *testing.T) {
	code, stdout, stderr := runBatch(t, "echo a\necho b\n")
	assert.Equal(t, 0, code)
	assert.Equal(t, "a\nb\n", stdout)
	assert.Empty(t, stderr)
}

// A failing command mid-input reports and the sequence continues; the
// shell's status is the LAST command's.
func TestBatch_FailureMidSequenceContinues(t *testing.T) {
	code, stdout, stderr := runBatch(t, "echo a\ndefinitely-not-a-cmd-xyz\necho c\n")
	assert.Equal(t, 0, code)
	assert.Equal(t, "a\nc\n", stdout)
	assert.Equal(t, "definitely-not-a-cmd-xyz: command not found\n", stderr)
}

// A trailing operator continues onto the next line (the lexer swallows the
// newline after && || | ; and redirections): `false &&\necho x` is ONE
// chain whose && skips echo — nothing prints, status 1.
func TestBatch_OperatorLineContinuation(t *testing.T) {
	code, stdout, _ := runBatch(t, "false &&\necho x\n")
	assert.Equal(t, 1, code)
	assert.Empty(t, stdout)
}

// The whole input is ONE parse, so every $? expands BEFORE anything runs:
// both echoes see the pre-input status 0 (execution.md §Non-Interactive
// Mode consequence 4). Per-line $? semantics apply to the interactive
// line-by-line path (see TestInteractive_LastStatusPerLine).
func TestBatch_DollarQuestionSeesPreInputStatus(t *testing.T) {
	code, stdout, _ := runBatch(t, "false\necho $?\ntrue\necho $?\n")
	assert.Equal(t, 0, code)
	assert.Equal(t, "0\n0\n", stdout)
}

// The interactive path parses (and expands) each line as it is entered, so
// $? sees the previous LINE's status. RunCommand calls model REPL lines.
func TestInteractive_LastStatusPerLine(t *testing.T) {
	var out, errBuf bytes.Buffer
	sh := NewWithIO(strings.NewReader(""), &out, &errBuf, false)
	ctx := context.Background()

	assert.Equal(t, 1, sh.RunCommand(ctx, "false"))
	sh.RunCommand(ctx, "echo $?")
	assert.Equal(t, 0, sh.RunCommand(ctx, "true"))
	sh.RunCommand(ctx, "echo $?")

	assert.Equal(t, "1\n0\n", out.String())
}

// exit stops the sequence: later commands do not run and the shell exits
// with the code.
func TestBatch_ExitStopsSequence(t *testing.T) {
	code, stdout, _ := runBatch(t, "echo one\nexit 3\necho two\n")
	assert.Equal(t, 3, code)
	assert.Equal(t, "one\n", stdout)
}

// exit with no argument exits with the shell's last recorded status — the
// pre-input status in whole-input mode (0 for a fresh shell).
func TestBatch_ExitNoArg(t *testing.T) {
	code, _, _ := runBatch(t, "false\nexit\n")
	assert.Equal(t, 0, code)
}

// A standalone assignment is observable by CHILD processes later in the
// input (env inheritance) — not by $VAR in the same input, which expanded
// at parse time.
func TestBatch_AssignmentVisibleToChildren(t *testing.T) {
	t.Setenv("FSH_BATCH_ASSIGN", "old")

	code, stdout, _ := runBatch(t, "FSH_BATCH_ASSIGN=new\nprintenv FSH_BATCH_ASSIGN\n")
	assert.Equal(t, 0, code)
	assert.Equal(t, "new\n", stdout)
}

// export affects child $PATH lookup: a shim directory prepended via export
// changes which binary a later command resolves to.
func TestBatch_ExportPathAffectsLookup(t *testing.T) {
	shimDir := t.TempDir()
	shim := filepath.Join(shimDir, "fsh-shim-cmd-xyz")
	require.NoError(t, os.WriteFile(shim, []byte("#!/bin/sh\necho shimmed\n"), 0o755))
	t.Setenv("PATH", os.Getenv("PATH")) // restore PATH after the test

	code, stdout, stderr := runBatch(t,
		"export PATH="+shimDir+":"+os.Getenv("PATH")+"\nfsh-shim-cmd-xyz\n")
	assert.Equal(t, 0, code, "stderr: %s", stderr)
	assert.Equal(t, "shimmed\n", stdout)
}

// Comments — including a shebang line — are removed by the lexer.
func TestBatch_CommentsAndShebang(t *testing.T) {
	code, stdout, stderr := runBatch(t, "#!/usr/bin/env fsh\n# a comment\necho ran\n")
	assert.Equal(t, 0, code)
	assert.Equal(t, "ran\n", stdout)
	assert.Empty(t, stderr)
}

// Commands inherit the shell's stdin, which is at EOF after the whole-input
// read: $(cat) yields nothing and can never steal script text.
func TestBatch_StdinAtEOFForCommands(t *testing.T) {
	code, stdout, _ := runBatch(t, "echo got:$(cat)\necho second\n")
	assert.Equal(t, 0, code)
	assert.Equal(t, "got:\nsecond\n", stdout)
}

// Input larger than bufio.Scanner's old 64KB line limit parses fine.
func TestBatch_LargeInput(t *testing.T) {
	long := strings.Repeat("x", 100_000)
	code, stdout, stderr := runBatch(t, "echo "+long+"\n")
	assert.Equal(t, 0, code, "stderr: %s", stderr)
	assert.Equal(t, long+"\n", stdout)
}

// RunScript reads the file in full; the script file is NOT the shell's
// stdin, and the shebang is an ordinary comment.
func TestRunScript_WholeInput(t *testing.T) {
	script := filepath.Join(t.TempDir(), "script.fsh")
	require.NoError(t, os.WriteFile(script,
		[]byte("#!/usr/bin/env fsh\necho from-script\nexit 5\n"), 0o644))

	var out, errBuf bytes.Buffer
	sh := NewWithIO(strings.NewReader("shell stdin, not script text\n"), &out, &errBuf, false)
	code := sh.RunScript(context.Background(), script)

	assert.Equal(t, 5, code)
	assert.Equal(t, "from-script\n", out.String())
}

// RunScript on an unopenable file: canonical message, status 1.
func TestRunScript_CannotOpenMessage(t *testing.T) {
	var out, errBuf bytes.Buffer
	sh := NewWithIO(strings.NewReader(""), &out, &errBuf, false)
	code := sh.RunScript(context.Background(), "/nonexistent/script.fsh")

	assert.Equal(t, 1, code)
	assert.Equal(t,
		"cannot open script file /nonexistent/script.fsh: no such file or directory\n",
		errBuf.String())
}

// Scripts can read the SHELL's stdin: commands inherit the stdin fsh was
// started with (spec: `fsh script.fsh < data.txt`).
func TestRunScript_CommandsReadShellStdin(t *testing.T) {
	script := filepath.Join(t.TempDir(), "reader.fsh")
	require.NoError(t, os.WriteFile(script, []byte("cat\n"), 0o644))

	var out, errBuf bytes.Buffer
	sh := NewWithIO(strings.NewReader("data for cat\n"), &out, &errBuf, false)
	code := sh.RunScript(context.Background(), script)

	assert.Equal(t, 0, code)
	assert.Equal(t, "data for cat\n", out.String())
}

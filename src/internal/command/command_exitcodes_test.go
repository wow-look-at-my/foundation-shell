package command

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"foundation-shell/pkg/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Normative exit codes (execution.md §Standard Exit Codes): 126 for a file
// that exists but cannot be executed, with `<name>: <os reason>`.
func TestExitCode126_NotExecutable(t *testing.T) {
	notExec := filepath.Join(t.TempDir(), "not-executable")
	require.NoError(t, os.WriteFile(notExec, []byte("data"), 0o644))

	spec := &parser.CommandSpec{Args: []string{notExec}}
	var stdout, stderr bytes.Buffer
	code, err := Execute(context.Background(), spec, nil, &stdout, &stderr)

	require.Nil(t, err)
	assert.Equal(t, 126, code)
	assert.Equal(t, notExec+": permission denied\n", stderr.String())
}

// A directory is found but not executable: 126 with `<name>: <os reason>`
// (Linux execve reports EACCES for directories, so the reason is
// "permission denied" here; the code is what is normative).
func TestExitCode126_IsADirectory(t *testing.T) {
	dir := t.TempDir()

	spec := &parser.CommandSpec{Args: []string{dir}}
	var stdout, stderr bytes.Buffer
	code, err := Execute(context.Background(), spec, nil, &stdout, &stderr)

	require.Nil(t, err)
	assert.Equal(t, 126, code)
	assert.True(t, strings.HasPrefix(stderr.String(), dir+": "),
		"message must be `<name>: <os reason>`, got %q", stderr.String())
}

// An explicit path that does not exist is still "command not found" (127).
func TestExitCode127_PathNotFound(t *testing.T) {
	spec := &parser.CommandSpec{Args: []string{"/nonexistent/dir/cmd12345"}}
	var stdout, stderr bytes.Buffer
	code, err := Execute(context.Background(), spec, nil, &stdout, &stderr)

	require.Nil(t, err)
	assert.Equal(t, 127, code)
	assert.Equal(t, "/nonexistent/dir/cmd12345: command not found\n", stderr.String())
}

// A child killed by signal N reports 128+N — never -1 or 255. SIGKILL=9.
func TestExitCode128PlusN_SignalDeath(t *testing.T) {
	spec := &parser.CommandSpec{Args: []string{"sh", "-c", "kill -9 $$"}}
	var stdout, stderr bytes.Buffer
	code, err := Execute(context.Background(), spec, nil, &stdout, &stderr)

	require.Nil(t, err)
	assert.Equal(t, 137, code)
	// Signal deaths are silent — no message.
	assert.Empty(t, stderr.String())
}

// External exit statuses 1-255 pass through unchanged.
func TestExitCodePassThrough(t *testing.T) {
	spec := &parser.CommandSpec{Args: []string{"sh", "-c", "exit 42"}}
	var stdout, stderr bytes.Buffer
	code, err := Execute(context.Background(), spec, nil, &stdout, &stderr)
	require.Nil(t, err)
	assert.Equal(t, 42, code)
}

// Output-redirection open failures use the canonical message with the
// unwrapped os reason.
func TestOutputRedirectOpenFailureMessage(t *testing.T) {
	target := "/nonexistent-dir-12345/out.txt"
	spec := &parser.CommandSpec{Args: []string{"echo", "hi"}, OutputFile: target}
	var stdout, stderr bytes.Buffer
	code, err := Execute(context.Background(), spec, nil, &stdout, &stderr)

	require.Nil(t, err)
	assert.Equal(t, 1, code)
	assert.Equal(t, "cannot open output file "+target+": no such file or directory\n", stderr.String())
	assert.Empty(t, stdout.String()) // the command did NOT execute
}

// OSReason unwraps PathError/exec wrapping to the bare OS error text.
func TestOSReason(t *testing.T) {
	pathErr := &fs.PathError{Op: "open", Path: "/x", Err: errors.New("some reason")}
	assert.Equal(t, "some reason", OSReason(pathErr))
	assert.Equal(t, "plain", OSReason(errors.New("plain")))
}

// A standalone assignment (marked by the parser) sets the variable
// in-process: status 0, nothing executed, no output.
func TestStandaloneAssignment(t *testing.T) {
	t.Setenv("FSH_TEST_ASSIGN", "old")

	spec := &parser.CommandSpec{Args: []string{"FSH_TEST_ASSIGN=new value"}, IsAssignment: true}
	var stdout, stderr bytes.Buffer
	code, err := Execute(context.Background(), spec, nil, &stdout, &stderr)

	require.Nil(t, err)
	assert.Equal(t, 0, code)
	assert.Equal(t, "new value", os.Getenv("FSH_TEST_ASSIGN"))
	assert.Empty(t, stdout.String())
	assert.Empty(t, stderr.String())
}

// The first '=' splits: the value may contain '='.
func TestStandaloneAssignment_FirstEqualsSplits(t *testing.T) {
	t.Setenv("FSH_TEST_ASSIGN2", "old")

	spec := &parser.CommandSpec{Args: []string{"FSH_TEST_ASSIGN2=B=C"}, IsAssignment: true}
	var stdout, stderr bytes.Buffer
	code, err := Execute(context.Background(), spec, nil, &stdout, &stderr)

	require.Nil(t, err)
	assert.Equal(t, 0, code)
	assert.Equal(t, "B=C", os.Getenv("FSH_TEST_ASSIGN2"))
}

// Without the parser's marking, a NAME=VALUE word is an ordinary command
// (quoted forms, multi-word commands): typically 127.
func TestUnmarkedAssignmentWordRunsAsCommand(t *testing.T) {
	spec := &parser.CommandSpec{Args: []string{"FSH_TEST_NOASSIGN=x"}}
	var stdout, stderr bytes.Buffer
	code, err := Execute(context.Background(), spec, nil, &stdout, &stderr)

	require.Nil(t, err)
	assert.Equal(t, 127, code)
	assert.Equal(t, "FSH_TEST_NOASSIGN=x: command not found\n", stderr.String())
}

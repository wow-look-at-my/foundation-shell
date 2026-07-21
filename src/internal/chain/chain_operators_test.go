package chain

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"foundation-shell/internal/command"
	"foundation-shell/pkg/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runChain parses src (no substitution executor) and executes it with
// buffered I/O.
func runChain(t *testing.T, src string) (code int, stdout, stderr string, err error) {
	t.Helper()
	cmdChain, parseErr := parser.Parse(src)
	require.NoError(t, parseErr, "parse %q", src)

	var out, errBuf bytes.Buffer
	code, err = ExecuteWithIO(context.Background(), cmdChain, strings.NewReader(""), &out, &errBuf)
	return code, out.String(), errBuf.String(), err
}

// Skip propagation: a skipped segment preserves the current status and the
// NEXT operator is evaluated against that same status. `false && a && b`
// runs nothing and yields 1 (the old executor ran b unconditionally).
func TestSkipPropagation_AndChain(t *testing.T) {
	code, stdout, _, err := runChain(t, "false && echo a && echo b")
	require.NoError(t, err)
	assert.Equal(t, 1, code)
	assert.Empty(t, stdout)
}

func TestSkipPropagation_OrChain(t *testing.T) {
	code, stdout, _, err := runChain(t, "true || echo a || echo b")
	require.NoError(t, err)
	assert.Equal(t, 0, code)
	assert.Empty(t, stdout)
}

func TestSkipPropagation_AndThenOrRecovers(t *testing.T) {
	code, stdout, _, err := runChain(t, "false && echo a || echo b")
	require.NoError(t, err)
	assert.Equal(t, 0, code)
	assert.Equal(t, "b\n", stdout)
}

func TestSkipPropagation_SemicolonResumes(t *testing.T) {
	code, stdout, _, err := runChain(t, "false && echo a ; echo b")
	require.NoError(t, err)
	assert.Equal(t, 0, code)
	assert.Equal(t, "b\n", stdout)
}

// A command that fails to spawn is an ordinary status-127 failure: ONE
// message on stderr, and ||, ;, && all see the status — the chain
// continues.
func TestExecErrorContinuation_OrFallback(t *testing.T) {
	code, stdout, stderr, err := runChain(t, "definitely-not-a-cmd-xyz || echo fallback")
	require.NoError(t, err)
	assert.Equal(t, 0, code)
	assert.Equal(t, "fallback\n", stdout)
	assert.Equal(t, "definitely-not-a-cmd-xyz: command not found\n", stderr)
}

func TestExecErrorContinuation_Semicolon(t *testing.T) {
	code, stdout, stderr, err := runChain(t, "definitely-not-a-cmd-xyz ; echo next")
	require.NoError(t, err)
	assert.Equal(t, 0, code)
	assert.Equal(t, "next\n", stdout)
	assert.Contains(t, stderr, "command not found")
}

func TestExecErrorContinuation_AndSkips(t *testing.T) {
	code, stdout, _, err := runChain(t, "definitely-not-a-cmd-xyz && echo nope")
	require.NoError(t, err)
	assert.Equal(t, 127, code)
	assert.Empty(t, stdout)
}

func TestRedirectFailureContinuation(t *testing.T) {
	code, stdout, stderr, err := runChain(t, "cat < /nonexistent-file-xyz || echo recovered")
	require.NoError(t, err)
	assert.Equal(t, 0, code)
	assert.Equal(t, "recovered\n", stdout)
	assert.Equal(t, "cannot open input file /nonexistent-file-xyz: no such file or directory\n", stderr)
}

// exit at top level stops the chain: later `;` segments do not run and the
// ErrExit sentinel propagates with the code.
func TestExitStopsChain(t *testing.T) {
	code, stdout, _, err := runChain(t, "echo one ; exit 7 ; echo two")

	var exitErr command.ErrExit
	require.ErrorAs(t, err, &exitErr)
	assert.Equal(t, 7, exitErr.Code)
	assert.Equal(t, 7, code)
	assert.Equal(t, "one\n", stdout)
}

// exit inside a multi-command pipeline sets only that segment's status; the
// chain (and shell) survive. Rightmost cat wins: status 0, `after` runs.
func TestExitInPipelineDoesNotStopChain(t *testing.T) {
	code, stdout, _, err := runChain(t, "exit 7 | cat ; echo after")
	require.NoError(t, err)
	assert.Equal(t, 0, code)
	assert.Equal(t, "after\n", stdout)
}

// Builtins in pipelines execute IN-PROCESS and mutate the parent shell — a
// deliberate, documented deviation from POSIX subshell semantics
// (execution.md §Builtins in Pipelines): `cd / | cat ; pwd` prints /.
func TestBuiltinInPipelineMutatesParent(t *testing.T) {
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	code, stdout, _, execErr := runChain(t, "cd / | cat ; pwd")
	require.NoError(t, execErr)
	assert.Equal(t, 0, code)
	assert.Equal(t, "/\n", stdout)
}

// Substitution executor: exit inside a body stops only the body's own
// sequence; the outer line continues and the body's status is discarded.
func TestExecutor_ExitInsideBodyDoesNotKillShell(t *testing.T) {
	var stderr bytes.Buffer
	e := NewExecutor(context.Background(), strings.NewReader(""), &stderr)

	output, code, err := e.Execute("echo captured ; exit 7 ; echo never")
	require.NoError(t, err)
	assert.Equal(t, 7, code)
	assert.Equal(t, "captured\n", output)
}

// The chain layer reports runtime failures itself: the executor must NOT
// print a second message (the old interim print is gone).
func TestExecutor_SingleFailureMessage(t *testing.T) {
	var stderr bytes.Buffer
	e := NewExecutor(context.Background(), strings.NewReader(""), &stderr)

	_, code, err := e.Execute("definitely-not-a-cmd-xyz")
	require.NoError(t, err)
	assert.Equal(t, 127, code)
	assert.Equal(t, "definitely-not-a-cmd-xyz: command not found\n", stderr.String())
}

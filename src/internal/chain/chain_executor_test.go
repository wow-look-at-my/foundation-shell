package chain

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"foundation-shell/pkg/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestExecutor() (*Executor, *bytes.Buffer) {
	var stderr bytes.Buffer
	return NewExecutor(context.Background(), strings.NewReader(""), &stderr), &stderr
}

func TestExecutor_SimpleCommand(t *testing.T) {
	e, stderr := newTestExecutor()

	output, exitCode, err := e.Execute("echo hi")
	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	assert.Equal(t, "hi\n", output)
	assert.Empty(t, stderr.String())
}

// The executor parses with ITSELF as the substitution executor, so nested
// substitutions in the body expand recursively.
func TestExecutor_RecursesForNestedSubstitution(t *testing.T) {
	e, _ := newTestExecutor()

	output, exitCode, err := e.Execute("echo $(echo $(echo deep))")
	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	assert.Equal(t, "deep\n", output)
}

// A body that fails to PARSE returns an error (the whole line fails).
func TestExecutor_ParseErrorPropagates(t *testing.T) {
	e, _ := newTestExecutor()

	_, exitCode, err := e.Execute("echo 'unclosed")
	require.Error(t, err)
	assert.Equal(t, 1, exitCode)
}

// A RUNTIME failure of the body is NOT an error: the captured stdout (here
// empty) is returned, the message goes to stderr, and the caller continues.
func TestExecutor_RuntimeFailureYieldsCapturedOutput(t *testing.T) {
	e, stderr := newTestExecutor()

	output, _, err := e.Execute("definitely-not-a-command-xyz")
	require.NoError(t, err)
	assert.Equal(t, "", output)
	assert.Contains(t, stderr.String(), "definitely-not-a-command-xyz")
}

// End-to-end through the parser: substitution output containing
// substitution syntax is spliced as DATA, never executed.
func TestExecutor_SubstitutionOutputNotReExecuted(t *testing.T) {
	payload := filepath.Join(t.TempDir(), "payload.txt")
	require.NoError(t, os.WriteFile(payload, []byte("$(echo pwned)"), 0o644))

	e, _ := newTestExecutor()
	cmdChain, err := parser.ParseWithExecutor("echo $(cat "+payload+")", e)
	require.NoError(t, err)

	var stdout, stderr bytes.Buffer
	exitCode, err := ExecuteWithIO(context.Background(), cmdChain, strings.NewReader(""), &stdout, &stderr)
	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	assert.Equal(t, "$(echo pwned)\n", stdout.String())
}

// Single quotes inside a substitution body suppress the nested
// substitution: the body's own lexing handles its quoting.
func TestExecutor_SingleQuotesInsideBodySuppressNested(t *testing.T) {
	e, _ := newTestExecutor()
	cmdChain, err := parser.ParseWithExecutor("echo $(echo '$(pwd)')", e)
	require.NoError(t, err)

	var stdout, stderr bytes.Buffer
	exitCode, err := ExecuteWithIO(context.Background(), cmdChain, strings.NewReader(""), &stdout, &stderr)
	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	assert.Equal(t, "$(pwd)\n", stdout.String())
}

// A failing substitution yields its captured stdout (possibly empty) and
// the line continues: `echo $(nosuchcmd)` prints an empty line, exit 0,
// with the failure reported on stderr.
func TestExecutor_LineContinuesAfterSubstitutionRuntimeFailure(t *testing.T) {
	var stderr bytes.Buffer
	e := NewExecutor(context.Background(), strings.NewReader(""), &stderr)

	cmdChain, err := parser.ParseWithExecutor("echo $(definitely-not-a-command-xyz)", e)
	require.NoError(t, err)

	var stdout bytes.Buffer
	exitCode, err := ExecuteWithIO(context.Background(), cmdChain, strings.NewReader(""), &stdout, &stderr)
	require.NoError(t, err)
	assert.Equal(t, 0, exitCode)
	assert.Equal(t, "\n", stdout.String())
	assert.Contains(t, stderr.String(), "definitely-not-a-command-xyz")
}

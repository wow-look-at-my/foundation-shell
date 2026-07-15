package shell_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// getFshExecPath locates the compiled fsh-exec binary (built by `just
// build`), mirroring getFshPath.
func getFshExecPath(t *testing.T) string {
	path := filepath.Join("..", "..", "..", "build", "fsh-exec")
	if _, err := os.Stat(path); err != nil {
		t.Skipf("fsh-exec executable not found at %s - run 'just build' first", path)
	}
	absPath, _ := filepath.Abs(path)
	return absPath
}

func runFshExec(t *testing.T, args ...string) (stdout, stderr string, exitCode int) {
	t.Helper()
	cmd := exec.Command(getFshExecPath(t), args...)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	cmd.Stdin = strings.NewReader("")

	err := cmd.Run()
	exitCode = 0
	if err != nil {
		exitErr, ok := err.(*exec.ExitError)
		require.True(t, ok, "failed to run fsh-exec: %v", err)
		exitCode = exitErr.ExitCode()
	}
	return outBuf.String(), errBuf.String(), exitCode
}

// The joined-argv form: all arguments become one command line.
func TestIntegration_FshExec_JoinedForm(t *testing.T) {
	stdout, _, code := runFshExec(t, "echo", "hello", "world")
	assert.Equal(t, 0, code)
	assert.Equal(t, "hello world\n", stdout)
}

// The -c form: the next argument IS the command line.
func TestIntegration_FshExec_DashC(t *testing.T) {
	stdout, _, code := runFshExec(t, "-c", "echo hi | tr a-z A-Z")
	assert.Equal(t, 0, code)
	assert.Equal(t, "HI\n", stdout)
}

// -c without an argument: usage on stderr, status 2.
func TestIntegration_FshExec_DashCMissingArg(t *testing.T) {
	_, stderr, code := runFshExec(t, "-c")
	assert.Equal(t, 2, code)
	assert.Contains(t, stderr, "Usage")
}

// exit propagates as the process exit code.
func TestIntegration_FshExec_ExitCode(t *testing.T) {
	_, _, code := runFshExec(t, "exit 7")
	assert.Equal(t, 7, code)

	_, _, code = runFshExec(t, "-c", "nosuchcmd-xyz")
	assert.Equal(t, 127, code)
}

// The flagship deadlock regression, end-to-end through the real binary:
// `yes | head -1` prints y and terminates promptly with status 0.
func TestIntegration_FshExec_PipelineEarlyExit(t *testing.T) {
	fshExec := getFshExecPath(t)

	cmd := exec.Command(fshExec, "yes | head -1")
	var outBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stdin = strings.NewReader("")
	require.NoError(t, cmd.Start())
	t.Cleanup(func() { _ = cmd.Process.Kill() })

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		assert.NoError(t, err, "expected status 0")
		assert.Equal(t, "y\n", outBuf.String())
	case <-time.After(15 * time.Second):
		t.Fatal("yes | head -1 hung (pipeline deadlock regression)")
	}
}

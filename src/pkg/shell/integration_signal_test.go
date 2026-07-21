package shell_test

import (
	"bytes"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// END-TO-END signal tests against the compiled fsh binary (execution.md
// §Signals and Cancellation). Timing is handled by POLLING for the child
// process via /proc, never by fixed sleeps, with generous CI-safe caps.
// =============================================================================

const signalPollCap = 15 * time.Second

// waitForChildComm polls until pid has a direct child whose comm matches,
// returning true when found. Children are listed per TASK (Go forks from
// arbitrary OS threads, so the child may hang off any tid, not just the
// main thread).
func waitForChildComm(t *testing.T, pid int, comm string) bool {
	t.Helper()
	pidStr := strconv.Itoa(pid)
	taskDir := "/proc/" + pidStr + "/task"
	deadline := time.Now().Add(signalPollCap)
	for time.Now().Before(deadline) {
		tasks, err := os.ReadDir(taskDir)
		if err == nil {
			for _, task := range tasks {
				data, err := os.ReadFile(taskDir + "/" + task.Name() + "/children")
				if err != nil {
					continue
				}
				for _, child := range strings.Fields(string(data)) {
					commData, err := os.ReadFile("/proc/" + child + "/comm")
					if err == nil && strings.TrimSpace(string(commData)) == comm {
						return true
					}
				}
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	return false
}

// The SECOND command of a session must be interruptible (the historical
// Ignore/Reset regression left SIGINT permanently ignored after the first
// command): SIGINT to the SHELL pid kills the CHILD, the shell survives and
// exits normally with 130.
func TestIntegration_SigintForwardedDuringSecondCommand(t *testing.T) {
	if _, err := os.Stat("/proc/self/comm"); err != nil {
		t.Skip("requires /proc (linux)")
	}
	fsh := getFshPath(t)

	cmd := exec.Command(fsh)
	cmd.Stdin = strings.NewReader("true\nsleep 30\n")
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	require.NoError(t, cmd.Start())
	t.Cleanup(func() { _ = cmd.Process.Kill() })

	// `true` runs first; wait for the SECOND command's child.
	require.True(t, waitForChildComm(t, cmd.Process.Pid, "sleep"),
		"sleep child never appeared under fsh")

	start := time.Now()
	require.NoError(t, cmd.Process.Signal(syscall.SIGINT))

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		elapsed := time.Since(start)
		assert.Less(t, elapsed, signalPollCap, "child must die promptly, not run 30s")
		// The shell SURVIVED the SIGINT (normal exit, not a signal death)
		// and reports the interrupted child's status 130.
		var exitErr *exec.ExitError
		require.ErrorAs(t, err, &exitErr)
		ws, ok := exitErr.Sys().(syscall.WaitStatus)
		require.True(t, ok)
		assert.False(t, ws.Signaled(), "the shell itself must not die of SIGINT")
		assert.Equal(t, 130, exitErr.ExitCode())
	case <-time.After(signalPollCap):
		t.Fatal("fsh did not exit after SIGINT -- signal ignored (regression)")
	}
}

// With NO child running, the shell keeps the default disposition: SIGINT
// kills it (a fresh non-interactive fsh blocked reading stdin dies).
func TestIntegration_IdleShellDiesOnSigint(t *testing.T) {
	fsh := getFshPath(t)

	// Hold the write end open so fsh stays blocked reading stdin with no
	// command ever started.
	r, w, err := os.Pipe()
	require.NoError(t, err)
	defer w.Close()
	defer r.Close()

	cmd := exec.Command(fsh)
	cmd.Stdin = r
	require.NoError(t, cmd.Start())
	t.Cleanup(func() { _ = cmd.Process.Kill() })

	// The default disposition applies from process start; no readiness
	// handshake is required for it to be lethal.
	time.Sleep(200 * time.Millisecond)
	require.NoError(t, cmd.Process.Signal(syscall.SIGINT))

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		var exitErr *exec.ExitError
		require.ErrorAs(t, err, &exitErr)
		ws, ok := exitErr.Sys().(syscall.WaitStatus)
		require.True(t, ok)
		assert.True(t, ws.Signaled(), "idle shell should die of the signal")
		assert.Equal(t, syscall.SIGINT, ws.Signal())
	case <-time.After(signalPollCap):
		t.Fatal("idle fsh survived SIGINT -- disposition wrongly ignored")
	}
}

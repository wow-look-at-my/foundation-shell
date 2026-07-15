package command

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"foundation-shell/pkg/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSigintForwardedToChild verifies the signal design (execution.md
// §Signals and Cancellation):
//
//	(a) SIGINT delivered to the SHELL process while a child runs is
//	    forwarded to the child; the shell survives and the child's death
//	    reports 130 (128+SIGINT).
//	(b) This still holds for the SECOND command. The old Ignore/Reset
//	    design left SIGINT permanently ignored after the first command
//	    (signal.Reset does not undo signal.Ignore), so later children
//	    inherited SIG_IGN — the regression this test pins down.
//
// The child signals readiness by writing to a pipe, so there are no fixed
// sleeps; timeouts are generous for CI.
func TestSigintForwardedToChild(t *testing.T) {
	// Safety net: keep the test process alive if a SIGINT arrives outside
	// Execute's forwarding window.
	safety := make(chan os.Signal, 1)
	signal.Notify(safety, syscall.SIGINT)
	defer signal.Stop(safety)

	runOnce := func(round string) {
		r, w, err := os.Pipe()
		require.NoError(t, err)
		defer r.Close()

		// After echo, sh execs into sleep (same pid); SIGINT default-kills
		// either way.
		spec := &parser.CommandSpec{Args: []string{"sh", "-c", "echo ready && exec sleep 30"}}

		type result struct {
			code int
			err  error
		}
		done := make(chan result, 1)
		go func() {
			var stderr bytes.Buffer
			code, execErr := Execute(context.Background(), spec, nil, w, &stderr)
			w.Close()
			done <- result{code, execErr}
		}()

		// Wait until the child is definitely running: it wrote "ready".
		// Execute registers its forwarder BEFORE starting the child, so
		// the forwarding window is certainly open now.
		buf := make([]byte, len("ready\n"))
		_, err = io.ReadFull(r, buf)
		require.NoError(t, err, "%s: child never became ready", round)
		require.Equal(t, "ready\n", string(buf))

		// Deliver SIGINT to the SHELL process (not the process group).
		require.NoError(t, syscall.Kill(os.Getpid(), syscall.SIGINT))

		select {
		case res := <-done:
			assert.Nil(t, res.err, round)
			assert.Equal(t, 130, res.code,
				"%s: interrupted child must report 130", round)
		case <-time.After(20 * time.Second):
			t.Fatalf("%s: child was not interrupted -- SIGINT lost or still ignored", round)
		}
	}

	runOnce("first command")
	// The regression: a second command must be exactly as interruptible.
	runOnce("second command")
}

package chain

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"foundation-shell/pkg/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runPipeline executes src under a guard context so a reintroduced
// deadlock fails the test with a clear message instead of hanging: the
// context kills the children after guardTimeout, and the elapsed assertion
// flags it.
const guardTimeout = 15 * time.Second

func runPipeline(t *testing.T, src string) (code int, stdout, stderr string) {
	t.Helper()
	cmdChain, err := parser.Parse(src)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), guardTimeout)
	defer cancel()

	var out, errBuf bytes.Buffer
	start := time.Now()
	code, execErr := ExecuteWithIO(ctx, cmdChain, strings.NewReader(""), &out, &errBuf)
	elapsed := time.Since(start)

	require.NoError(t, execErr)
	require.Less(t, elapsed, guardTimeout,
		"pipeline %q must terminate on its own (deadlock regression)", src)
	return code, out.String(), errBuf.String()
}

// The canonical early-exit case: the consumer stops reading, the producer's
// next write fails (in-process EPIPE), and the pipeline terminates with the
// rightmost status. Historically this deadlocked forever.
func TestPipelineEarlyExit_YesHead(t *testing.T) {
	code, stdout, stderr := runPipeline(t, "yes | head -1")
	assert.Equal(t, 0, code)
	assert.Equal(t, "y\n", stdout)
	// The producer's termination is silent: no error spam.
	assert.Empty(t, stderr)
}

// Even a FINITE producer larger than the pipe rendezvous used to hang.
func TestPipelineEarlyExit_FiniteProducer(t *testing.T) {
	code, stdout, _ := runPipeline(t, "seq 1 100000 | head -2")
	assert.Equal(t, 0, code)
	assert.Equal(t, "1\n2\n", stdout)
}

// A builtin consumer that never reads stdin must not wedge the producer.
func TestPipelineEarlyExit_BuiltinConsumer(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)

	code, stdout, _ := runPipeline(t, "echo x | pwd")
	assert.Equal(t, 0, code)
	assert.Equal(t, cwd+"\n", stdout)
}

// Spec §7.4: a pipeline command whose stdin is redirected from a file has
// its incoming pipe closed at wiring time, so the upstream producer
// terminates instead of blocking.
func TestPipelineInputRedirectDisconnectsPipe(t *testing.T) {
	inFile := filepath.Join(t.TempDir(), "in.txt")
	require.NoError(t, os.WriteFile(inFile, []byte("file contents\n"), 0o644))

	code, stdout, _ := runPipeline(t, "echo hi | cat < "+inFile)
	assert.Equal(t, 0, code)
	assert.Equal(t, "file contents\n", stdout)
}

// Spawn failures inside pipelines are reported (never swallowed) and do
// not wedge the pipeline. Rightmost-wins for the status.
func TestPipelineSpawnFailure_Producer(t *testing.T) {
	code, stdout, stderr := runPipeline(t, "definitely-not-a-cmd-xyz | cat")
	assert.Equal(t, 0, code) // cat succeeded; rightmost wins
	assert.Empty(t, stdout)
	assert.Equal(t, "definitely-not-a-cmd-xyz: command not found\n", stderr)
}

func TestPipelineSpawnFailure_Consumer(t *testing.T) {
	code, _, stderr := runPipeline(t, "echo hi | definitely-not-a-cmd-xyz")
	assert.Equal(t, 127, code)
	assert.Equal(t, "definitely-not-a-cmd-xyz: command not found\n", stderr)
}

// EVERY segment's spawn failure is reported, not just the last one's.
func TestPipelineSpawnFailure_MultipleReported(t *testing.T) {
	_, _, stderr := runPipeline(t, "not-a-cmd-one | not-a-cmd-two | cat")
	assert.Contains(t, stderr, "not-a-cmd-one: command not found")
	assert.Contains(t, stderr, "not-a-cmd-two: command not found")
}

// A long pipeline with an early-exiting middle stage still terminates and
// streams correctly.
func TestPipelineEarlyExit_MiddleStage(t *testing.T) {
	code, stdout, _ := runPipeline(t, "seq 1 100000 | head -3 | tail -1")
	assert.Equal(t, 0, code)
	assert.Equal(t, "3\n", stdout)
}

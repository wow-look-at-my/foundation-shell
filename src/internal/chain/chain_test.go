package chain

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"

	"foundation-shell/internal/token"
	"foundation-shell/pkg/parser"
	"github.com/stretchr/testify/assert"
)

// executeWithCapture runs Execute but captures stdout
func executeWithCapture(ctx context.Context, chain *parser.Chain) (exitCode int, stdout string, err error) {
	// Save original stdout
	oldStdout := os.Stdout

	// Create a pipe to capture stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Run the chain
	exitCode, err = Execute(ctx, chain)

	// Restore stdout and read captured output
	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	r.Close()

	return exitCode, buf.String(), err
}

func TestEmptyChain(t *testing.T) {
	ctx := context.Background()

	// nil chain
	exitCode, err := Execute(ctx, nil)
	assert.Nil(t, err)

	assert.Equal(t, 0, exitCode)

	// empty commands
	chain := &parser.Chain{Commands: []*parser.CommandSpec{}}
	exitCode, err = Execute(ctx, chain)
	assert.Nil(t, err)

	assert.Equal(t, 0, exitCode)

}

func TestSingleCommand(t *testing.T) {
	ctx := context.Background()
	chain := &parser.Chain{
		Commands: []*parser.CommandSpec{
			{Args: []string{"echo", "hello"}},
		},
	}

	exitCode, stdout, err := executeWithCapture(ctx, chain)
	assert.Nil(t, err)

	assert.Equal(t, 0, exitCode)

	assert.Equal(t, "hello\n", stdout)

}

func TestPipelineEchoCat(t *testing.T) {
	ctx := context.Background()
	chain := &parser.Chain{
		Commands: []*parser.CommandSpec{
			{Args: []string{"echo", "hello"}},
			{Args: []string{"cat"}},
		},
		Operators: []token.TokenType{token.Pipe},
	}

	exitCode, stdout, err := executeWithCapture(ctx, chain)
	assert.Nil(t, err)

	assert.Equal(t, 0, exitCode)

	assert.Equal(t, "hello\n", stdout)

}

func TestPipelineEchoGrep(t *testing.T) {
	ctx := context.Background()
	chain := &parser.Chain{
		Commands: []*parser.CommandSpec{
			{Args: []string{"echo", "-e", "a\nb\nc"}},
			{Args: []string{"grep", "b"}},
		},
		Operators: []token.TokenType{token.Pipe},
	}

	exitCode, stdout, err := executeWithCapture(ctx, chain)
	assert.Nil(t, err)

	assert.Equal(t, 0, exitCode)

	assert.Equal(t, "b\n", stdout)

}

func TestLongPipeline(t *testing.T) {
	ctx := context.Background()
	chain := &parser.Chain{
		Commands: []*parser.CommandSpec{
			{Args: []string{"echo", "test"}},
			{Args: []string{"cat"}},
			{Args: []string{"cat"}},
			{Args: []string{"cat"}},
		},
		Operators: []token.TokenType{token.Pipe, token.Pipe, token.Pipe},
	}

	exitCode, stdout, err := executeWithCapture(ctx, chain)
	assert.Nil(t, err)

	assert.Equal(t, 0, exitCode)

	assert.Equal(t, "test\n", stdout)

}

func TestAndOperatorSuccess(t *testing.T) {
	ctx := context.Background()
	chain := &parser.Chain{
		Commands: []*parser.CommandSpec{
			{Args: []string{"true"}},
			{Args: []string{"echo", "yes"}},
		},
		Operators: []token.TokenType{token.And},
	}

	exitCode, stdout, err := executeWithCapture(ctx, chain)
	assert.Nil(t, err)

	assert.Equal(t, 0, exitCode)

	assert.Equal(t, "yes\n", stdout)

}

func TestAndOperatorShortCircuit(t *testing.T) {
	ctx := context.Background()
	chain := &parser.Chain{
		Commands: []*parser.CommandSpec{
			{Args: []string{"false"}},
			{Args: []string{"echo", "yes"}},
		},
		Operators: []token.TokenType{token.And},
	}

	exitCode, stdout, err := executeWithCapture(ctx, chain)
	assert.Nil(t, err)

	assert.Equal(t, 1, exitCode)

	assert.Equal(t, "", stdout)

}

func TestOrOperatorSuccess(t *testing.T) {
	ctx := context.Background()
	chain := &parser.Chain{
		Commands: []*parser.CommandSpec{
			{Args: []string{"false"}},
			{Args: []string{"echo", "yes"}},
		},
		Operators: []token.TokenType{token.Or},
	}

	exitCode, stdout, err := executeWithCapture(ctx, chain)
	assert.Nil(t, err)

	assert.Equal(t, 0, exitCode)

	assert.Equal(t, "yes\n", stdout)

}

func TestOrOperatorShortCircuit(t *testing.T) {
	ctx := context.Background()
	chain := &parser.Chain{
		Commands: []*parser.CommandSpec{
			{Args: []string{"true"}},
			{Args: []string{"echo", "yes"}},
		},
		Operators: []token.TokenType{token.Or},
	}

	exitCode, stdout, err := executeWithCapture(ctx, chain)
	assert.Nil(t, err)

	assert.Equal(t, 0, exitCode)

	assert.Equal(t, "", stdout)

}

func TestMixedOperators(t *testing.T) {
	ctx := context.Background()
	// false || echo one && echo two
	chain := &parser.Chain{
		Commands: []*parser.CommandSpec{
			{Args: []string{"false"}},
			{Args: []string{"echo", "one"}},
			{Args: []string{"echo", "two"}},
		},
		Operators: []token.TokenType{token.Or, token.And},
	}

	exitCode, stdout, err := executeWithCapture(ctx, chain)
	assert.Nil(t, err)

	assert.Equal(t, 0, exitCode)

	assert.Equal(t, "one\ntwo\n", stdout)

}

func TestPipelineWithAnd(t *testing.T) {
	ctx := context.Background()
	// echo test | grep test && echo found
	chain := &parser.Chain{
		Commands: []*parser.CommandSpec{
			{Args: []string{"echo", "test"}},
			{Args: []string{"grep", "test"}},
			{Args: []string{"echo", "found"}},
		},
		Operators: []token.TokenType{token.Pipe, token.And},
	}

	exitCode, stdout, err := executeWithCapture(ctx, chain)
	assert.Nil(t, err)

	assert.Equal(t, 0, exitCode)

	// Output should be "test\n" from grep followed by "found\n" from echo
	assert.Equal(t, "test\nfound\n", stdout)

}

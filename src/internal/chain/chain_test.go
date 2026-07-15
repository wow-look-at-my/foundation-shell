package chain

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"

	"foundation-shell/internal/token"
	"foundation-shell/pkg/parser"
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
	if err != nil {
		t.Errorf("Execute(nil) error = %v, want nil", err)
	}
	if exitCode != 0 {
		t.Errorf("Execute(nil) exitCode = %d, want 0", exitCode)
	}

	// empty commands
	chain := &parser.Chain{Commands: []*parser.CommandSpec{}}
	exitCode, err = Execute(ctx, chain)
	if err != nil {
		t.Errorf("Execute(empty) error = %v, want nil", err)
	}
	if exitCode != 0 {
		t.Errorf("Execute(empty) exitCode = %d, want 0", exitCode)
	}
}

func TestSingleCommand(t *testing.T) {
	ctx := context.Background()
	chain := &parser.Chain{
		Commands: []*parser.CommandSpec{
			{Args: []string{"echo", "hello"}},
		},
	}

	exitCode, stdout, err := executeWithCapture(ctx, chain)
	if err != nil {
		t.Errorf("Execute error = %v, want nil", err)
	}
	if exitCode != 0 {
		t.Errorf("exitCode = %d, want 0", exitCode)
	}
	if stdout != "hello\n" {
		t.Errorf("stdout = %q, want %q", stdout, "hello\n")
	}
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
	if err != nil {
		t.Errorf("Execute error = %v, want nil", err)
	}
	if exitCode != 0 {
		t.Errorf("exitCode = %d, want 0", exitCode)
	}
	if stdout != "hello\n" {
		t.Errorf("stdout = %q, want %q", stdout, "hello\n")
	}
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
	if err != nil {
		t.Errorf("Execute error = %v, want nil", err)
	}
	if exitCode != 0 {
		t.Errorf("exitCode = %d, want 0", exitCode)
	}
	if stdout != "b\n" {
		t.Errorf("stdout = %q, want %q", stdout, "b\n")
	}
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
	if err != nil {
		t.Errorf("Execute error = %v, want nil", err)
	}
	if exitCode != 0 {
		t.Errorf("exitCode = %d, want 0", exitCode)
	}
	if stdout != "test\n" {
		t.Errorf("stdout = %q, want %q", stdout, "test\n")
	}
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
	if err != nil {
		t.Errorf("Execute error = %v, want nil", err)
	}
	if exitCode != 0 {
		t.Errorf("exitCode = %d, want 0", exitCode)
	}
	if stdout != "yes\n" {
		t.Errorf("stdout = %q, want %q", stdout, "yes\n")
	}
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
	if err != nil {
		t.Errorf("Execute error = %v, want nil", err)
	}
	if exitCode != 1 {
		t.Errorf("exitCode = %d, want 1", exitCode)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want empty", stdout)
	}
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
	if err != nil {
		t.Errorf("Execute error = %v, want nil", err)
	}
	if exitCode != 0 {
		t.Errorf("exitCode = %d, want 0", exitCode)
	}
	if stdout != "yes\n" {
		t.Errorf("stdout = %q, want %q", stdout, "yes\n")
	}
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
	if err != nil {
		t.Errorf("Execute error = %v, want nil", err)
	}
	if exitCode != 0 {
		t.Errorf("exitCode = %d, want 0", exitCode)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want empty", stdout)
	}
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
	if err != nil {
		t.Errorf("Execute error = %v, want nil", err)
	}
	if exitCode != 0 {
		t.Errorf("exitCode = %d, want 0", exitCode)
	}
	if stdout != "one\ntwo\n" {
		t.Errorf("stdout = %q, want %q", stdout, "one\ntwo\n")
	}
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
	if err != nil {
		t.Errorf("Execute error = %v, want nil", err)
	}
	if exitCode != 0 {
		t.Errorf("exitCode = %d, want 0", exitCode)
	}
	// Output should be "test\n" from grep followed by "found\n" from echo
	if stdout != "test\nfound\n" {
		t.Errorf("stdout = %q, want %q", stdout, "test\nfound\n")
	}
}

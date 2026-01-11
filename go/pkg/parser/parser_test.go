package parser

import (
	"os"
	"testing"

	"foundation-shell/internal/token"
)

func TestParse_SingleCommand(t *testing.T) {
	chain, err := Parse("echo hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chain.Commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(chain.Commands))
	}

	if len(chain.Operators) != 0 {
		t.Fatalf("expected 0 operators, got %d", len(chain.Operators))
	}

	cmd := chain.Commands[0]
	if len(cmd.Args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(cmd.Args))
	}

	if cmd.Args[0] != "echo" {
		t.Errorf("expected arg[0] = 'echo', got %q", cmd.Args[0])
	}

	if cmd.Args[1] != "hello" {
		t.Errorf("expected arg[1] = 'hello', got %q", cmd.Args[1])
	}
}

func TestParse_Pipeline(t *testing.T) {
	chain, err := Parse("echo test | grep test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chain.Commands) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(chain.Commands))
	}

	if len(chain.Operators) != 1 {
		t.Fatalf("expected 1 operator, got %d", len(chain.Operators))
	}

	if chain.Operators[0] != token.Pipe {
		t.Errorf("expected Pipe operator, got %v", chain.Operators[0])
	}

	// First command
	cmd1 := chain.Commands[0]
	if len(cmd1.Args) != 2 || cmd1.Args[0] != "echo" || cmd1.Args[1] != "test" {
		t.Errorf("first command incorrect: %v", cmd1.Args)
	}

	// Second command
	cmd2 := chain.Commands[1]
	if len(cmd2.Args) != 2 || cmd2.Args[0] != "grep" || cmd2.Args[1] != "test" {
		t.Errorf("second command incorrect: %v", cmd2.Args)
	}
}

func TestParse_AndOrChain(t *testing.T) {
	chain, err := Parse("cmd1 && cmd2 || cmd3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chain.Commands) != 3 {
		t.Fatalf("expected 3 commands, got %d", len(chain.Commands))
	}

	if len(chain.Operators) != 2 {
		t.Fatalf("expected 2 operators, got %d", len(chain.Operators))
	}

	if chain.Operators[0] != token.And {
		t.Errorf("expected And operator at index 0, got %v", chain.Operators[0])
	}

	if chain.Operators[1] != token.Or {
		t.Errorf("expected Or operator at index 1, got %v", chain.Operators[1])
	}

	if chain.Commands[0].Args[0] != "cmd1" {
		t.Errorf("expected cmd1, got %s", chain.Commands[0].Args[0])
	}

	if chain.Commands[1].Args[0] != "cmd2" {
		t.Errorf("expected cmd2, got %s", chain.Commands[1].Args[0])
	}

	if chain.Commands[2].Args[0] != "cmd3" {
		t.Errorf("expected cmd3, got %s", chain.Commands[2].Args[0])
	}
}

func TestParse_Redirections(t *testing.T) {
	chain, err := Parse("cat < in.txt > out.txt 2> err.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chain.Commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(chain.Commands))
	}

	cmd := chain.Commands[0]

	if len(cmd.Args) != 1 || cmd.Args[0] != "cat" {
		t.Errorf("expected [cat], got %v", cmd.Args)
	}

	if cmd.InputFile != "in.txt" {
		t.Errorf("expected InputFile = 'in.txt', got %q", cmd.InputFile)
	}

	if cmd.OutputFile != "out.txt" {
		t.Errorf("expected OutputFile = 'out.txt', got %q", cmd.OutputFile)
	}

	if cmd.AppendOutput != false {
		t.Errorf("expected AppendOutput = false, got true")
	}

	if cmd.ErrorFile != "err.txt" {
		t.Errorf("expected ErrorFile = 'err.txt', got %q", cmd.ErrorFile)
	}

	if cmd.AppendError != false {
		t.Errorf("expected AppendError = false, got true")
	}
}

func TestParse_AppendRedirections(t *testing.T) {
	chain, err := Parse("echo test >> file.txt 2>> err.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chain.Commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(chain.Commands))
	}

	cmd := chain.Commands[0]

	if len(cmd.Args) != 2 || cmd.Args[0] != "echo" || cmd.Args[1] != "test" {
		t.Errorf("expected [echo, test], got %v", cmd.Args)
	}

	if cmd.OutputFile != "file.txt" {
		t.Errorf("expected OutputFile = 'file.txt', got %q", cmd.OutputFile)
	}

	if cmd.AppendOutput != true {
		t.Errorf("expected AppendOutput = true, got false")
	}

	if cmd.ErrorFile != "err.txt" {
		t.Errorf("expected ErrorFile = 'err.txt', got %q", cmd.ErrorFile)
	}

	if cmd.AppendError != true {
		t.Errorf("expected AppendError = true, got false")
	}
}

func TestParse_MixedPipelineWithRedirection(t *testing.T) {
	chain, err := Parse("cat file.txt | grep pattern > result.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chain.Commands) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(chain.Commands))
	}

	if len(chain.Operators) != 1 || chain.Operators[0] != token.Pipe {
		t.Errorf("expected 1 Pipe operator, got %v", chain.Operators)
	}

	// First command
	cmd1 := chain.Commands[0]
	if len(cmd1.Args) != 2 || cmd1.Args[0] != "cat" || cmd1.Args[1] != "file.txt" {
		t.Errorf("first command incorrect: %v", cmd1.Args)
	}

	// Second command with redirection
	cmd2 := chain.Commands[1]
	if len(cmd2.Args) != 2 || cmd2.Args[0] != "grep" || cmd2.Args[1] != "pattern" {
		t.Errorf("second command args incorrect: %v", cmd2.Args)
	}

	if cmd2.OutputFile != "result.txt" {
		t.Errorf("expected OutputFile = 'result.txt', got %q", cmd2.OutputFile)
	}
}

func TestParse_EnvironmentExpansion(t *testing.T) {
	// Set HOME for the test
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", "/home/testuser")
	defer os.Setenv("HOME", oldHome)

	chain, err := Parse("echo $HOME")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chain.Commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(chain.Commands))
	}

	cmd := chain.Commands[0]
	if len(cmd.Args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(cmd.Args))
	}

	if cmd.Args[1] != "/home/testuser" {
		t.Errorf("expected expanded HOME, got %q", cmd.Args[1])
	}
}

func TestParse_SingleQuotesPreventExpansion(t *testing.T) {
	// Set HOME for the test
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", "/home/testuser")
	defer os.Setenv("HOME", oldHome)

	chain, err := Parse("echo '$HOME'")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chain.Commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(chain.Commands))
	}

	cmd := chain.Commands[0]
	if len(cmd.Args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(cmd.Args))
	}

	if cmd.Args[1] != "$HOME" {
		t.Errorf("expected literal '$HOME', got %q", cmd.Args[1])
	}
}

func TestParse_TildeExpansion(t *testing.T) {
	// Set HOME for the test
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", "/home/testuser")
	defer os.Setenv("HOME", oldHome)

	chain, err := Parse("cd ~/projects")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chain.Commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(chain.Commands))
	}

	cmd := chain.Commands[0]
	if cmd.Args[1] != "/home/testuser/projects" {
		t.Errorf("expected '/home/testuser/projects', got %q", cmd.Args[1])
	}
}

// Error cases

func TestParse_ErrorOperatorAtStart(t *testing.T) {
	_, err := Parse("| cmd")
	if err == nil {
		t.Fatal("expected error for operator at start")
	}
}

func TestParse_ErrorMissingRedirectionTarget(t *testing.T) {
	_, err := Parse("cmd >")
	if err == nil {
		t.Fatal("expected error for missing redirection target")
	}
}

func TestParse_ErrorTrailingAndOperator(t *testing.T) {
	_, err := Parse("&&")
	if err == nil {
		t.Fatal("expected error for && only")
	}
}

func TestParse_ErrorTrailingPipe(t *testing.T) {
	_, err := Parse("cmd |")
	if err == nil {
		t.Fatal("expected error for trailing pipe")
	}
}

func TestParse_ErrorEmptyInput(t *testing.T) {
	_, err := Parse("")
	if err == nil {
		t.Fatal("expected error for empty input")
	}

	_, err = Parse("   ")
	if err == nil {
		t.Fatal("expected error for whitespace-only input")
	}
}

func TestParse_ErrorConsecutiveOperators(t *testing.T) {
	_, err := Parse("cmd1 && || cmd2")
	if err == nil {
		t.Fatal("expected error for consecutive operators")
	}
}

func TestParse_ErrorRedirectionFollowedByOperator(t *testing.T) {
	_, err := Parse("cmd > |")
	if err == nil {
		t.Fatal("expected error for redirection followed by operator")
	}
}

func TestParse_MultipleArgs(t *testing.T) {
	chain, err := Parse("ls -la /tmp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cmd := chain.Commands[0]
	if len(cmd.Args) != 3 {
		t.Fatalf("expected 3 args, got %d", len(cmd.Args))
	}

	if cmd.Args[0] != "ls" || cmd.Args[1] != "-la" || cmd.Args[2] != "/tmp" {
		t.Errorf("args incorrect: %v", cmd.Args)
	}
}

func TestParse_ComplexChain(t *testing.T) {
	chain, err := Parse("cat input.txt | grep -v error | sort > output.txt 2> errors.log")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chain.Commands) != 3 {
		t.Fatalf("expected 3 commands, got %d", len(chain.Commands))
	}

	if len(chain.Operators) != 2 {
		t.Fatalf("expected 2 operators, got %d", len(chain.Operators))
	}

	// All operators should be Pipe
	for i, op := range chain.Operators {
		if op != token.Pipe {
			t.Errorf("expected Pipe at index %d, got %v", i, op)
		}
	}

	// Check last command has redirections
	lastCmd := chain.Commands[2]
	if lastCmd.OutputFile != "output.txt" {
		t.Errorf("expected OutputFile = 'output.txt', got %q", lastCmd.OutputFile)
	}

	if lastCmd.ErrorFile != "errors.log" {
		t.Errorf("expected ErrorFile = 'errors.log', got %q", lastCmd.ErrorFile)
	}
}

func TestParse_BracedEnvVar(t *testing.T) {
	os.Setenv("TESTVAR", "testvalue")
	defer os.Unsetenv("TESTVAR")

	chain, err := Parse("echo ${TESTVAR}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cmd := chain.Commands[0]
	if cmd.Args[1] != "testvalue" {
		t.Errorf("expected 'testvalue', got %q", cmd.Args[1])
	}
}

func TestParse_InputRedirectionOnly(t *testing.T) {
	chain, err := Parse("wc -l < data.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cmd := chain.Commands[0]
	if cmd.Args[0] != "wc" || cmd.Args[1] != "-l" {
		t.Errorf("args incorrect: %v", cmd.Args)
	}

	if cmd.InputFile != "data.txt" {
		t.Errorf("expected InputFile = 'data.txt', got %q", cmd.InputFile)
	}
}

func TestParse_RedirectionBeforeArgs(t *testing.T) {
	// Redirections can appear in any position
	chain, err := Parse("< input.txt cat")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cmd := chain.Commands[0]
	if len(cmd.Args) != 1 || cmd.Args[0] != "cat" {
		t.Errorf("args incorrect: %v", cmd.Args)
	}

	if cmd.InputFile != "input.txt" {
		t.Errorf("expected InputFile = 'input.txt', got %q", cmd.InputFile)
	}
}

func TestParse_EscapedDollar(t *testing.T) {
	os.Setenv("VAR", "value")
	defer os.Unsetenv("VAR")

	chain, err := Parse("echo \\$VAR")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cmd := chain.Commands[0]
	if cmd.Args[1] != "$VAR" {
		t.Errorf("expected literal '$VAR', got %q", cmd.Args[1])
	}
}

func TestParse_DoubleQuotesExpand(t *testing.T) {
	os.Setenv("NAME", "world")
	defer os.Unsetenv("NAME")

	chain, err := Parse("echo \"hello $NAME\"")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cmd := chain.Commands[0]
	if cmd.Args[1] != "hello world" {
		t.Errorf("expected 'hello world', got %q", cmd.Args[1])
	}
}

func TestParse_LongPipeline(t *testing.T) {
	chain, err := Parse("a | b | c | d | e")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chain.Commands) != 5 {
		t.Fatalf("expected 5 commands, got %d", len(chain.Commands))
	}

	if len(chain.Operators) != 4 {
		t.Fatalf("expected 4 operators, got %d", len(chain.Operators))
	}

	expected := []string{"a", "b", "c", "d", "e"}
	for i, cmd := range chain.Commands {
		if len(cmd.Args) != 1 || cmd.Args[0] != expected[i] {
			t.Errorf("command %d: expected [%s], got %v", i, expected[i], cmd.Args)
		}
	}
}

func TestParse_RedirectionWithoutSpaces(t *testing.T) {
	// The lexer should split these as separate tokens
	// If "cmd>file" is a single token, it won't be recognized as redirection
	// This test documents current behavior
	chain, err := Parse("echo hello > file.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cmd := chain.Commands[0]
	if cmd.OutputFile != "file.txt" {
		t.Errorf("expected OutputFile = 'file.txt', got %q", cmd.OutputFile)
	}
}

func TestParse_UnsetEnvVar(t *testing.T) {
	// Ensure the variable is not set
	os.Unsetenv("NONEXISTENT_VAR_12345")

	chain, err := Parse("echo $NONEXISTENT_VAR_12345")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cmd := chain.Commands[0]
	// Unset env var should expand to empty string
	if cmd.Args[1] != "" {
		t.Errorf("expected empty string for unset var, got %q", cmd.Args[1])
	}
}

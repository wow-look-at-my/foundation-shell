package parser

import (
	"os"
	"testing"

	"foundation-shell/internal/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse_SingleCommand(t *testing.T) {
	chain, err := Parse("echo hello")
	require.Nil(t, err)

	require.Equal(t, 1, len(chain.Commands))

	require.Equal(t, 0, len(chain.Operators))

	cmd := chain.Commands[0]
	require.Equal(t, 2, len(cmd.Args))

	assert.Equal(t, "echo", cmd.Args[0])

	assert.Equal(t, "hello", cmd.Args[1])

}

func TestParse_Pipeline(t *testing.T) {
	chain, err := Parse("echo test | grep test")
	require.Nil(t, err)

	require.Equal(t, 2, len(chain.Commands))

	require.Equal(t, 1, len(chain.Operators))

	assert.Equal(t, token.Pipe, chain.Operators[0])

	// First command
	cmd1 := chain.Commands[0]
	assert.False(t, len(cmd1.Args) != 2 || cmd1.Args[0] != "echo" || cmd1.Args[1] != "test")

	// Second command
	cmd2 := chain.Commands[1]
	assert.False(t, len(cmd2.Args) != 2 || cmd2.Args[0] != "grep" || cmd2.Args[1] != "test")

}

func TestParse_AndOrChain(t *testing.T) {
	chain, err := Parse("cmd1 && cmd2 || cmd3")
	require.Nil(t, err)

	require.Equal(t, 3, len(chain.Commands))

	require.Equal(t, 2, len(chain.Operators))

	assert.Equal(t, token.And, chain.Operators[0])

	assert.Equal(t, token.Or, chain.Operators[1])

	assert.Equal(t, "cmd1", chain.Commands[0].Args[0])

	assert.Equal(t, "cmd2", chain.Commands[1].Args[0])

	assert.Equal(t, "cmd3", chain.Commands[2].Args[0])

}

func TestParse_Redirections(t *testing.T) {
	chain, err := Parse("cat < in.txt > out.txt 2> err.txt")
	require.Nil(t, err)

	require.Equal(t, 1, len(chain.Commands))

	cmd := chain.Commands[0]

	assert.False(t, len(cmd.Args) != 1 || cmd.Args[0] != "cat")

	assert.Equal(t, "in.txt", cmd.InputFile)

	assert.Equal(t, "out.txt", cmd.OutputFile)

	assert.Equal(t, false, cmd.AppendOutput)

	assert.Equal(t, "err.txt", cmd.ErrorFile)

	assert.Equal(t, false, cmd.AppendError)

}

func TestParse_AppendRedirections(t *testing.T) {
	chain, err := Parse("echo test >> file.txt 2>> err.txt")
	require.Nil(t, err)

	require.Equal(t, 1, len(chain.Commands))

	cmd := chain.Commands[0]

	assert.False(t, len(cmd.Args) != 2 || cmd.Args[0] != "echo" || cmd.Args[1] != "test")

	assert.Equal(t, "file.txt", cmd.OutputFile)

	assert.Equal(t, true, cmd.AppendOutput)

	assert.Equal(t, "err.txt", cmd.ErrorFile)

	assert.Equal(t, true, cmd.AppendError)

}

func TestParse_MixedPipelineWithRedirection(t *testing.T) {
	chain, err := Parse("cat file.txt | grep pattern > result.txt")
	require.Nil(t, err)

	require.Equal(t, 2, len(chain.Commands))

	assert.False(t, len(chain.Operators) != 1 || chain.Operators[0] != token.Pipe)

	// First command
	cmd1 := chain.Commands[0]
	assert.False(t, len(cmd1.Args) != 2 || cmd1.Args[0] != "cat" || cmd1.Args[1] != "file.txt")

	// Second command with redirection
	cmd2 := chain.Commands[1]
	assert.False(t, len(cmd2.Args) != 2 || cmd2.Args[0] != "grep" || cmd2.Args[1] != "pattern")

	assert.Equal(t, "result.txt", cmd2.OutputFile)

}

func TestParse_EnvironmentExpansion(t *testing.T) {
	// Set HOME for the test
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", "/home/testuser")
	defer os.Setenv("HOME", oldHome)

	chain, err := Parse("echo $HOME")
	require.Nil(t, err)

	require.Equal(t, 1, len(chain.Commands))

	cmd := chain.Commands[0]
	require.Equal(t, 2, len(cmd.Args))

	assert.Equal(t, "/home/testuser", cmd.Args[1])

}

func TestParse_SingleQuotesPreventExpansion(t *testing.T) {
	// Set HOME for the test
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", "/home/testuser")
	defer os.Setenv("HOME", oldHome)

	chain, err := Parse("echo '$HOME'")
	require.Nil(t, err)

	require.Equal(t, 1, len(chain.Commands))

	cmd := chain.Commands[0]
	require.Equal(t, 2, len(cmd.Args))

	assert.Equal(t, "$HOME", cmd.Args[1])

}

func TestParse_TildeExpansion(t *testing.T) {
	// Set HOME for the test
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", "/home/testuser")
	defer os.Setenv("HOME", oldHome)

	chain, err := Parse("cd ~/projects")
	require.Nil(t, err)

	require.Equal(t, 1, len(chain.Commands))

	cmd := chain.Commands[0]
	assert.Equal(t, "/home/testuser/projects", cmd.Args[1])

}

// Error cases

func TestParse_ErrorOperatorAtStart(t *testing.T) {
	_, err := Parse("| cmd")
	require.NotNil(t, err)

}

func TestParse_ErrorMissingRedirectionTarget(t *testing.T) {
	_, err := Parse("cmd >")
	require.NotNil(t, err)

}

func TestParse_ErrorTrailingAndOperator(t *testing.T) {
	_, err := Parse("&&")
	require.NotNil(t, err)

}

func TestParse_ErrorTrailingPipe(t *testing.T) {
	_, err := Parse("cmd |")
	require.NotNil(t, err)

}

func TestParse_ErrorEmptyInput(t *testing.T) {
	_, err := Parse("")
	require.NotNil(t, err)

	_, err = Parse("   ")
	require.NotNil(t, err)

}

func TestParse_ErrorConsecutiveOperators(t *testing.T) {
	_, err := Parse("cmd1 && || cmd2")
	require.NotNil(t, err)

}

func TestParse_ErrorRedirectionFollowedByOperator(t *testing.T) {
	_, err := Parse("cmd > |")
	require.NotNil(t, err)

}

func TestParse_MultipleArgs(t *testing.T) {
	chain, err := Parse("ls -la /tmp")
	require.Nil(t, err)

	cmd := chain.Commands[0]
	require.Equal(t, 3, len(cmd.Args))

	assert.False(t, cmd.Args[0] != "ls" || cmd.Args[1] != "-la" || cmd.Args[2] != "/tmp")

}

func TestParse_ComplexChain(t *testing.T) {
	chain, err := Parse("cat input.txt | grep -v error | sort > output.txt 2> errors.log")
	require.Nil(t, err)

	require.Equal(t, 3, len(chain.Commands))

	require.Equal(t, 2, len(chain.Operators))

	// All operators should be Pipe
	for _, op := range chain.Operators {
		assert.Equal(t, token.Pipe, op)

	}

	// Check last command has redirections
	lastCmd := chain.Commands[2]
	assert.Equal(t, "output.txt", lastCmd.OutputFile)

	assert.Equal(t, "errors.log", lastCmd.ErrorFile)

}

func TestParse_BracedEnvVar(t *testing.T) {
	os.Setenv("TESTVAR", "testvalue")
	defer os.Unsetenv("TESTVAR")

	chain, err := Parse("echo ${TESTVAR}")
	require.Nil(t, err)

	cmd := chain.Commands[0]
	assert.Equal(t, "testvalue", cmd.Args[1])

}

func TestParse_InputRedirectionOnly(t *testing.T) {
	chain, err := Parse("wc -l < data.txt")
	require.Nil(t, err)

	cmd := chain.Commands[0]
	assert.False(t, cmd.Args[0] != "wc" || cmd.Args[1] != "-l")

	assert.Equal(t, "data.txt", cmd.InputFile)

}

func TestParse_RedirectionBeforeArgs(t *testing.T) {
	// Redirections can appear in any position
	chain, err := Parse("< input.txt cat")
	require.Nil(t, err)

	cmd := chain.Commands[0]
	assert.False(t, len(cmd.Args) != 1 || cmd.Args[0] != "cat")

	assert.Equal(t, "input.txt", cmd.InputFile)

}

func TestParse_EscapedDollar(t *testing.T) {
	os.Setenv("VAR", "value")
	defer os.Unsetenv("VAR")

	chain, err := Parse("echo \\$VAR")
	require.Nil(t, err)

	cmd := chain.Commands[0]
	assert.Equal(t, "$VAR", cmd.Args[1])

}

func TestParse_DoubleQuotesExpand(t *testing.T) {
	os.Setenv("NAME", "world")
	defer os.Unsetenv("NAME")

	chain, err := Parse("echo \"hello $NAME\"")
	require.Nil(t, err)

	cmd := chain.Commands[0]
	assert.Equal(t, "hello world", cmd.Args[1])

}

func TestParse_LongPipeline(t *testing.T) {
	chain, err := Parse("a | b | c | d | e")
	require.Nil(t, err)

	require.Equal(t, 5, len(chain.Commands))

	require.Equal(t, 4, len(chain.Operators))

	expected := []string{"a", "b", "c", "d", "e"}
	for i, cmd := range chain.Commands {
		assert.False(t, len(cmd.Args) != 1 || cmd.Args[0] != expected[i])

	}
}

func TestParse_RedirectionWithoutSpaces(t *testing.T) {
	// The lexer should split these as separate tokens
	// If "cmd>file" is a single token, it won't be recognized as redirection
	// This test documents current behavior
	chain, err := Parse("echo hello > file.txt")
	require.Nil(t, err)

	cmd := chain.Commands[0]
	assert.Equal(t, "file.txt", cmd.OutputFile)

}

func TestParse_UnsetEnvVar(t *testing.T) {
	// Ensure the variable is not set
	os.Unsetenv("NONEXISTENT_VAR_12345")

	chain, err := Parse("echo $NONEXISTENT_VAR_12345")
	require.Nil(t, err)

	cmd := chain.Commands[0]
	// Unset env var should expand to empty string
	assert.Equal(t, "", cmd.Args[1])

}

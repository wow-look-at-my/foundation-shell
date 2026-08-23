package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A lone unquoted & is the background operator in other shells. This shell
// has no background jobs, so the lexer gives back an ordinary word
// (lexer.md §3.3.2). An accepted & would be absorbed into argv along with
// every word after it: `server &` would run in the foreground and never
// return, and `cmd_a & cmd_b` would never run cmd_b.
func TestParse_LoneAmpersandRejected(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"trailing", "sleep 30 &"},
		{"between commands", "echo a & echo b"},
		{"leading", "& echo a"},
		{"after a pipeline", "echo a | grep a &"},
		{"after an operator", "true && sleep 1 &"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.input)
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrBackgroundUnsupported)
			assert.Equal(t, "background execution is not supported", err.Error())
		})
	}
}

// The guard reads the PRE-expansion word, so every other use of & stays a
// legal word: quoting sets wasQuoted, an escape keeps its marker, and an
// embedded & never forms a token of its own.
func TestParse_AmpersandInWordsStillLegal(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantArg string
	}{
		{"embedded in a word", "echo a&b", "a&b"},
		{"double quoted", `echo "a & b"`, "a & b"},
		{"single quoted", "echo 'a & b'", "a & b"},
		{"escaped alone", `echo \&`, "&"},
		{"escaped in a word", `echo x\&`, "x&"},
		{"url query string", `echo "h?a=1&b=2"`, "h?a=1&b=2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chain, err := Parse(tt.input)
			require.NoError(t, err)
			require.Len(t, chain.Commands, 1)
			assert.Equal(t, []string{"echo", tt.wantArg}, chain.Commands[0].Args)
		})
	}
}

// && is still the AND operator: the guard matches a word, never an operator.
func TestParse_AndOperatorUnaffected(t *testing.T) {
	chain, err := Parse("true && echo ok")
	require.NoError(t, err)
	assert.Len(t, chain.Commands, 2)
}

// `<<` and `<<<` lex as consecutive `<` operators. Reported as themselves,
// they no longer read as a missing filename.
func TestParse_HeredocRejected(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"here-document", "cat <<EOF"},
		{"here-document with a space", "cat << EOF"},
		{"here-string", "cat <<<word"},
		{"heredoc mid-chain", "cat <<EOF | grep x"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.input)
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrHeredocUnsupported)
			assert.Equal(t, "here-documents are not supported", err.Error())
		})
	}
}

// A single < is still an ordinary input redirection.
func TestParse_SingleInputRedirectionUnaffected(t *testing.T) {
	chain, err := Parse("cat < in.txt")
	require.NoError(t, err)
	require.Len(t, chain.Commands, 1)
	assert.Equal(t, "in.txt", chain.Commands[0].InputFile)
}

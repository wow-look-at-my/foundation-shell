package lexer

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func op(s string) TokenContext {
	return TokenContext{Content: s, IsOperator: true}
}

func word(s string) TokenContext {
	return TokenContext{Content: s}
}

// Operators do not require surrounding whitespace: unquoted, unescaped
// operator characters end the current word and become operator tokens by
// maximal munch.
func TestTokenize_OperatorsWithoutWhitespace(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []TokenContext
	}{
		{
			name:     "pipe without spaces",
			input:    "echo hello|grep world",
			expected: []TokenContext{word("echo"), word("hello"), op("|"), word("grep"), word("world")},
		},
		{
			name:     "semicolon without spaces",
			input:    "echo a;echo b",
			expected: []TokenContext{word("echo"), word("a"), op(";"), word("echo"), word("b")},
		},
		{
			name:     "and without spaces",
			input:    "a&&b",
			expected: []TokenContext{word("a"), op("&&"), word("b")},
		},
		{
			name:     "or without spaces",
			input:    "a||b",
			expected: []TokenContext{word("a"), op("||"), word("b")},
		},
		{
			name:     "single ampersand is a literal word character",
			input:    "a&b",
			expected: []TokenContext{word("a&b")},
		},
		{
			name:     "maximal munch takes && then literal ampersand",
			input:    "a&&&b",
			expected: []TokenContext{word("a"), op("&&"), word("&b")},
		},
		{
			name:     "output redirection without spaces",
			input:    "echo hi>out.txt",
			expected: []TokenContext{word("echo"), word("hi"), op(">"), word("out.txt")},
		},
		{
			name:     "append redirection without spaces",
			input:    "echo hi>>out.txt",
			expected: []TokenContext{word("echo"), word("hi"), op(">>"), word("out.txt")},
		},
		{
			name:     "input redirection without spaces",
			input:    "cat<in.txt",
			expected: []TokenContext{word("cat"), op("<"), word("in.txt")},
		},
		{
			name:     "operators with mixed spacing",
			input:    "echo a |grep b| cat",
			expected: []TokenContext{word("echo"), word("a"), op("|"), word("grep"), word("b"), op("|"), word("cat")},
		},
		{
			name:     "operator directly after command",
			input:    "echo>file",
			expected: []TokenContext{word("echo"), op(">"), word("file")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Tokenize(tt.input)
			require.Nil(t, err)

			assertTokensEqual(t, tt.expected, result)
		})
	}
}

// 2> and 2>> are recognized only when the pending word is exactly an
// unquoted, unescaped "2"; that 2 is consumed into the operator.
func TestTokenize_StderrRedirectionRecognition(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []TokenContext
	}{
		{
			name:     "2> merges the pending bare 2",
			input:    "cmd 2>err.log",
			expected: []TokenContext{word("cmd"), op("2>"), word("err.log")},
		},
		{
			name:     "2>> merges the pending bare 2",
			input:    "cmd 2>>err.log",
			expected: []TokenContext{word("cmd"), op("2>>"), word("err.log")},
		},
		{
			name:     "word ending in 2 keeps the 2",
			input:    "echo a2>f",
			expected: []TokenContext{word("echo"), word("a2"), op(">"), word("f")},
		},
		{
			name:     "multi-digit pending word is not a stderr redirect",
			input:    "echo 22>f",
			expected: []TokenContext{word("echo"), word("22"), op(">"), word("f")},
		},
		{
			name:  "quoted 2 is not a stderr redirect",
			input: `echo "2">f`,
			expected: []TokenContext{
				word("echo"),
				{Content: "2", WasQuoted: true},
				op(">"),
				word("f"),
			},
		},
		{
			name:     "escaped 2 is not a stderr redirect",
			input:    `echo \2>f`,
			expected: []TokenContext{word("echo"), word("2"), op(">"), word("f")},
		},
		{
			name:     "standalone 2 separated by space stays an argument",
			input:    "echo 2 >f",
			expected: []TokenContext{word("echo"), word("2"), op(">"), word("f")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Tokenize(tt.input)
			require.Nil(t, err)

			assertTokensEqual(t, tt.expected, result)
		})
	}
}

// Quoted or escaped operator characters are literal word content, never
// operators.
func TestTokenize_QuotedOperatorsAreLiteral(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []TokenContext
	}{
		{
			name:  "single-quoted pipe",
			input: "echo '|'",
			expected: []TokenContext{
				word("echo"),
				{Content: "|", WasSingleQuoted: true, WasQuoted: true},
			},
		},
		{
			name:  "double-quoted pipe",
			input: `echo "|"`,
			expected: []TokenContext{
				word("echo"),
				{Content: "|", WasQuoted: true},
			},
		},
		{
			name:     "escaped pipe",
			input:    `echo \|`,
			expected: []TokenContext{word("echo"), word("|")},
		},
		{
			name:  "single-quoted semicolon",
			input: "echo ';'",
			expected: []TokenContext{
				word("echo"),
				{Content: ";", WasSingleQuoted: true, WasQuoted: true},
			},
		},
		{
			name:  "quoted greater-than",
			input: `grep '>' data.txt`,
			expected: []TokenContext{
				word("grep"),
				{Content: ">", WasSingleQuoted: true, WasQuoted: true},
				word("data.txt"),
			},
		},
		{
			name:  "operator characters inside a quoted word",
			input: `echo "a|b;c>d"`,
			expected: []TokenContext{
				word("echo"),
				{Content: "a|b;c>d", WasQuoted: true},
			},
		},
		{
			name:     "operator characters inside a substitution body",
			input:    "echo $(cat f | grep x; true)",
			expected: []TokenContext{word("echo"), word("$(cat f | grep x; true)")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Tokenize(tt.input)
			require.Nil(t, err)

			assertTokensEqual(t, tt.expected, result)
		})
	}
}

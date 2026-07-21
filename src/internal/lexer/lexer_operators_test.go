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

// An unquoted # at word start begins a comment running to the next
// unquoted newline or end of input.
func TestTokenize_Comments(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []TokenContext
	}{
		{
			name:     "comment after arguments",
			input:    "echo a # rest",
			expected: []TokenContext{word("echo"), word("a")},
		},
		{
			name:     "comment without space after hash",
			input:    "echo a #rest of line",
			expected: []TokenContext{word("echo"), word("a")},
		},
		{
			name:     "whole-line comment",
			input:    "# just a comment",
			expected: nil,
		},
		{
			name:     "shebang line",
			input:    "#!/usr/bin/env fsh",
			expected: nil,
		},
		{
			name:     "hash inside a word is literal",
			input:    "foo#bar",
			expected: []TokenContext{word("foo#bar")},
		},
		{
			name:  "quoted hash is literal",
			input: "echo '#' x",
			expected: []TokenContext{
				word("echo"),
				{Content: "#", WasSingleQuoted: true, WasQuoted: true},
				word("x"),
			},
		},
		{
			name:  "hash inside double quotes is literal",
			input: `echo "a#b"`,
			expected: []TokenContext{
				word("echo"),
				{Content: "a#b", WasQuoted: true},
			},
		},
		{
			name:  "hash right after empty quotes is part of the word",
			input: "''#foo",
			expected: []TokenContext{
				{Content: "#foo", WasSingleQuoted: true, WasQuoted: true},
			},
		},
		{
			name:     "comment directly after an operator",
			input:    "echo|#comment",
			expected: []TokenContext{word("echo"), op("|")},
		},
		{
			name:     "comment swallows unclosed quote",
			input:    "echo ok # it's fine",
			expected: []TokenContext{word("echo"), word("ok")},
		},
		{
			name:     "hash inside substitution body is body text",
			input:    "echo $(a # b)",
			expected: []TokenContext{word("echo"), word("$(a # b)")},
		},
		{
			name:     "comment ends at newline",
			input:    "echo a # c\necho b",
			expected: []TokenContext{word("echo"), word("a"), op(";"), word("echo"), word("b")},
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

// An unquoted newline outside substitutions separates commands (implicit
// ;) unless the previous token is already an operator.
func TestTokenize_NewlineSeparator(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []TokenContext
	}{
		{
			name:     "newline separates commands",
			input:    "echo a\necho b",
			expected: []TokenContext{word("echo"), word("a"), op(";"), word("echo"), word("b")},
		},
		{
			name:     "blank lines emit no extra separators",
			input:    "echo a\n\n\necho b",
			expected: []TokenContext{word("echo"), word("a"), op(";"), word("echo"), word("b")},
		},
		{
			name:     "leading newlines are swallowed",
			input:    "\n\necho a",
			expected: []TokenContext{word("echo"), word("a")},
		},
		{
			// lexer.md §3.4: the separator is pending and materializes only
			// before a following token — a trailing newline produces nothing.
			name:     "trailing newline produces nothing",
			input:    "echo a\n",
			expected: []TokenContext{word("echo"), word("a")},
		},
		{
			name:     "trailing newline run produces nothing",
			input:    "echo a\n\n  \n",
			expected: []TokenContext{word("echo"), word("a")},
		},
		{
			name:     "newline after && is line continuation",
			input:    "echo a &&\necho b",
			expected: []TokenContext{word("echo"), word("a"), op("&&"), word("echo"), word("b")},
		},
		{
			name:     "newline after pipe is line continuation",
			input:    "echo a |\ncat",
			expected: []TokenContext{word("echo"), word("a"), op("|"), word("cat")},
		},
		{
			name:     "newline after semicolon is swallowed",
			input:    "echo a;\necho b",
			expected: []TokenContext{word("echo"), word("a"), op(";"), word("echo"), word("b")},
		},
		{
			// lexer.md §3.4: after a REDIRECTION operator the separator is
			// NOT suppressed — a redirection cannot be continued across a
			// newline. The parser rejects the resulting `> ;` sequence.
			name:     "newline after > is NOT line continuation",
			input:    "echo hi >\nout.txt",
			expected: []TokenContext{word("echo"), word("hi"), op(">"), op(";"), word("out.txt")},
		},
		{
			name:     "newline after 2>> is NOT line continuation",
			input:    "cmd 2>>\nerr.log",
			expected: []TokenContext{word("cmd"), op("2>>"), op(";"), word("err.log")},
		},
		{
			name:     "newline after < is NOT line continuation",
			input:    "cat <\nin.txt",
			expected: []TokenContext{word("cat"), op("<"), op(";"), word("in.txt")},
		},
		{
			// The separator materializes once even across blank lines, and a
			// dangling redirection at EOF (after a trailing newline) simply
			// ends the token stream at the operator.
			name:     "dangling redirection before trailing newline",
			input:    "echo hi >\n",
			expected: []TokenContext{word("echo"), word("hi"), op(">")},
		},
		{
			name:  "newline inside single quotes is literal",
			input: "echo 'a\nb'",
			expected: []TokenContext{
				word("echo"),
				{Content: "a\nb", WasSingleQuoted: true, WasQuoted: true},
			},
		},
		{
			name:  "newline inside double quotes is literal",
			input: "echo \"a\nb\"",
			expected: []TokenContext{
				word("echo"),
				{Content: "a\nb", WasQuoted: true},
			},
		},
		{
			name:     "newline inside substitution body is body text",
			input:    "echo $(a\nb)",
			expected: []TokenContext{word("echo"), word("$(a\nb)")},
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

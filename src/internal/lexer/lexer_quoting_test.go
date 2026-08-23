package lexer

import (
	"github.com/stretchr/testify/require"
	"testing"
)

// Quote characters inside a command-substitution body are preserved
// verbatim in the token content: the body is re-parsed when the
// substitution executes. They never set the token's quoting flags, and a
// quoted ) or ` does not close the substitution.
func TestTokenize_QuotesInsideSubstitution(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []TokenContext
	}{
		{
			name:  "double quotes inside dollar paren are preserved",
			input: `echo $(echo "a  b")`,
			expected: []TokenContext{
				{Content: "echo"},
				{Content: `$(echo "a  b")`},
			},
		},
		{
			name:  "single quotes inside dollar paren are preserved",
			input: `echo $(echo 'a  b')`,
			expected: []TokenContext{
				{Content: "echo"},
				{Content: `$(echo 'a  b')`},
			},
		},
		{
			name:  "double-quoted close paren stays in the body",
			input: `echo $(echo "x)y")`,
			expected: []TokenContext{
				{Content: "echo"},
				{Content: `$(echo "x)y")`},
			},
		},
		{
			name:  "single-quoted close paren stays in the body",
			input: `echo $(echo ')')`,
			expected: []TokenContext{
				{Content: "echo"},
				{Content: `$(echo ')')`},
			},
		},
		{
			name:  "double quotes inside backticks are preserved",
			input: "echo `echo \"a  b\"`",
			expected: []TokenContext{
				{Content: "echo"},
				{Content: "`echo \"a  b\"`"},
			},
		},
		{
			name:  "quoted backtick inside backticks stays in the body",
			input: "echo `echo \"a\\`b\"`",
			// The escaped backtick is consumed pairwise so it cannot close
			// the substitution; the body keeps it verbatim.
			expected: []TokenContext{
				{Content: "echo"},
				{Content: "`echo \"a\\`b\"`"},
			},
		},
		{
			name:  "substitution inside double quotes keeps inner quotes",
			input: `echo "$(echo "a  b")"`,
			// Outer quotes are stripped and flag the token; body quotes are
			// preserved verbatim.
			expected: []TokenContext{
				{Content: "echo"},
				{Content: `$(echo "a  b")`, WasQuoted: true},
			},
		},
		{
			name:  "nested substitution inside quoted body",
			input: `echo $(echo "$(date)")`,
			expected: []TokenContext{
				{Content: "echo"},
				{Content: `$(echo "$(date)")`},
			},
		},
		{
			name:  "escaped close paren inside body stays verbatim",
			input: `echo $(printf %s a\)b)`,
			expected: []TokenContext{
				{Content: "echo"},
				{Content: `$(printf %s a\)b)`},
			},
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

// Quoting flags reset at every unquoted-whitespace word boundary, even when
// the word was empty: `echo '' $HOME` must still expand $HOME downstream.
func TestTokenize_FlagResetAtWordBoundary(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []TokenContext
	}{
		{
			name:  "empty single quotes do not leak into next word",
			input: "echo '' $HOME",
			expected: []TokenContext{
				{Content: "echo"},
				{Content: "", WasSingleQuoted: true, WasQuoted: true},
				{Content: "$HOME"},
			},
		},
		{
			name:  "empty double quotes do not leak into next word",
			input: `echo "" $HOME`,
			expected: []TokenContext{
				{Content: "echo"},
				{Content: "", WasQuoted: true},
				{Content: "$HOME"},
			},
		},
		{
			name:  "non-empty single quotes reset as before",
			input: "echo 'x' $HOME",
			expected: []TokenContext{
				{Content: "echo"},
				{Content: "x", WasSingleQuoted: true, WasQuoted: true},
				{Content: "$HOME"},
			},
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

// Escaped backticks are literal characters: the lexer marks them with the
// escape marker so the expander does not execute them.
func TestTokenize_EscapedBacktick(t *testing.T) {
	m := string(EscapeMarker)

	tests := []struct {
		name     string
		input    string
		expected []TokenContext
	}{
		{
			name:  "escaped backticks around a word",
			input: `echo \` + "`" + `whoami\` + "`",
			expected: []TokenContext{
				{Content: "echo"},
				{Content: m + "`whoami" + m + "`", WasEscaped: true},
			},
		},
		{
			name:  "escaped backtick inside double quotes",
			input: `echo "foo \` + "`" + ` bar"`,
			expected: []TokenContext{
				{Content: "echo"},
				{Content: "foo " + m + "` bar", WasQuoted: true, WasEscaped: true},
			},
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

func TestStripEscapeMarkers_Backtick(t *testing.T) {
	m := string(EscapeMarker)

	require.Equal(t, "`whoami`", StripEscapeMarkers(m+"`whoami"+m+"`"))
}

package lexer

import (
	"github.com/stretchr/testify/require"
	"testing"
)

// Depth-tracked quote NESTING is Foundation Shell's flagship non-POSIX
// feature. Inside an open quote region, a same-type quote character nests
// one level deeper iff the previous character is whitespace and the next
// character exists, is not whitespace, and is not a quote character;
// otherwise it closes one level. Nested quote characters stay literal in
// the content; only the outermost pair is stripped.
func TestTokenize_NestedQuotes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []TokenContext
	}{
		{
			name:  "single-quote nesting one level",
			input: `echo 'outer 'inner' end'`,
			// Depth trace: 1 -> 2 -> 1 -> 0
			expected: []TokenContext{
				{Content: "echo"},
				{Content: `outer 'inner' end`, WasSingleQuoted: true, WasQuoted: true},
			},
		},
		{
			name:  "the flagship nesting example",
			input: `echo 'a 'b' c' 'd'`,
			// POSIX would concatenate to one token `a b c`; nesting keeps
			// the inner quotes literal instead.
			expected: []TokenContext{
				{Content: "echo"},
				{Content: `a 'b' c`, WasSingleQuoted: true, WasQuoted: true},
				{Content: "d", WasSingleQuoted: true, WasQuoted: true},
			},
		},
		{
			name:  "multi-level nesting",
			input: `echo 'l1 'l2 'l3' l2' l1'`,
			// Depth trace: 1 -> 2 -> 3 -> 2 -> 1 -> 0
			expected: []TokenContext{
				{Content: "echo"},
				{Content: `l1 'l2 'l3' l2' l1`, WasSingleQuoted: true, WasQuoted: true},
			},
		},
		{
			name:  "double-quote nesting",
			input: `echo "outer "inner" end"`,
			expected: []TokenContext{
				{Content: "echo"},
				{Content: `outer "inner" end`, WasQuoted: true},
			},
		},
		{
			name: "quote-char lookahead keeps interleaved regions closing",
			// Every interior closer is followed by a quote character, so
			// each one CLOSES: the classic quote-the-quotes idiom is
			// unchanged by nesting.
			input: `echo "Hello, "'"'"$USER"'"'"!"`,
			expected: []TokenContext{
				{Content: "echo"},
				{Content: `Hello, "$USER"!`, WasSingleQuoted: true, WasQuoted: true},
			},
		},
		{
			name:  "backtick nesting is one substitution token",
			input: "echo `outer `inner` end`",
			// The body (nested backticks preserved literally) re-parses
			// when the substitution executes.
			expected: []TokenContext{
				{Content: "echo"},
				{Content: "`outer `inner` end`"},
			},
		},
		{
			name:  "nesting applies inside substitution bodies",
			input: `echo $(echo 'a 'b' c')`,
			expected: []TokenContext{
				{Content: "echo"},
				{Content: `$(echo 'a 'b' c')`},
			},
		},
		{
			name:  "backtick substitution inside double quotes",
			input: "echo \"`date`\"",
			expected: []TokenContext{
				{Content: "echo"},
				{Content: "`date`", WasQuoted: true},
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

// POSIX-identical quoting must be unaffected by the nesting rule: an
// adjacent or word-final quote character never satisfies the nesting test.
func TestTokenize_NestingPOSIXCompatible(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []TokenContext
	}{
		{
			name:  "separate quoted words",
			input: `echo 'a' 'b'`,
			expected: []TokenContext{
				{Content: "echo"},
				{Content: "a", WasSingleQuoted: true, WasQuoted: true},
				{Content: "b", WasSingleQuoted: true, WasQuoted: true},
			},
		},
		{
			name:  "quoted then unquoted concatenation",
			input: `echo 'a'b`,
			expected: []TokenContext{
				{Content: "echo"},
				{Content: "ab", WasSingleQuoted: true, WasQuoted: true},
			},
		},
		{
			name:  "the escaped-apostrophe idiom",
			input: `echo 'don'\''t'`,
			expected: []TokenContext{
				{Content: "echo"},
				{Content: "don't", WasSingleQuoted: true, WasQuoted: true},
			},
		},
		{
			name:  "empty quotes",
			input: `echo ''`,
			expected: []TokenContext{
				{Content: "echo"},
				{Content: "", WasSingleQuoted: true, WasQuoted: true},
			},
		},
		{
			name:  "adjacent quote pairs concatenate",
			input: `echo 'a''b'`,
			expected: []TokenContext{
				{Content: "echo"},
				{Content: "ab", WasSingleQuoted: true, WasQuoted: true},
			},
		},
		{
			name:  "single space in quotes",
			input: `echo ' '`,
			expected: []TokenContext{
				{Content: "echo"},
				{Content: " ", WasSingleQuoted: true, WasQuoted: true},
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

// The cost of nesting: a quote character that nests (whitespace before,
// word character after) leaves the region OPEN, so inputs that POSIX would
// accept fail loudly. An even quote count can now be unclosed.
func TestTokenize_NestingUnclosedErrors(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectedErr string
	}{
		{
			name:        "nested opener never closed",
			input:       `echo 'hello 'world`,
			expectedErr: "unclosed single quote",
		},
		{
			name:        "double-quote closer reads as nested opener",
			input:       `echo "Total: "$N`,
			expectedErr: "unclosed double quote",
		},
		{
			name:        "space then word after quote nests",
			input:       `echo ' 'x`,
			expectedErr: "unclosed single quote",
		},
		{
			name:        "nested opener before a substitution",
			input:       `echo "count: "$(date)`,
			expectedErr: "unclosed double quote",
		},
		{
			name:        "backtick nests and stays open",
			input:       "echo `a `b",
			expectedErr: "unclosed backtick",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Tokenize(tt.input)
			require.NotNil(t, err)

			require.Equal(t, tt.expectedErr, err.Error())
		})
	}
}

// Unclosed constructs report the INNERMOST unclosed construct.
func TestTokenize_UnclosedInnermostConstruct(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectedErr string
	}{
		{
			name:        "substitution open inside double quotes",
			input:       `echo "$(a`,
			expectedErr: "unclosed command substitution $(...)",
		},
		{
			name:        "single quote open inside substitution body",
			input:       `echo $(echo 'a`,
			expectedErr: "unclosed single quote",
		},
		{
			name:        "double quote open inside backtick body",
			input:       "echo `echo \"a",
			expectedErr: "unclosed double quote",
		},
		{
			name:        "backtick open inside substitution",
			input:       "echo $(`a",
			expectedErr: "unclosed backtick",
		},
		{
			name:        "substitution open inside backticks",
			input:       "echo `a$(b",
			expectedErr: "unclosed command substitution $(...)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Tokenize(tt.input)
			require.NotNil(t, err)

			require.Equal(t, tt.expectedErr, err.Error())
		})
	}
}

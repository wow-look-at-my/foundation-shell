package syntax

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

// requireSingleWordType asserts the input is valid and that a token with
// the given value and type exists.
func requireWordWithType(t *testing.T, input, value string, semType SemanticType) {
	t.Helper()

	result := Analyze(input)

	require.True(t, result.Valid, "Analyze(%q) errors: %#v", input, result.Errors)

	for _, tok := range result.Tokens {
		if tok.Value == value {
			assert.Equal(t, semType, tok.Type, "type of %q in %q", value, input)
			return
		}
	}
	t.Fatalf("token %q not found in %#v", value, result.Tokens)
}

// requireOnlyError asserts the input is invalid with exactly one error
// carrying the given message.
func requireOnlyError(t *testing.T, input, message string) {
	t.Helper()

	result := Analyze(input)

	require.False(t, result.Valid, "Analyze(%q) unexpectedly valid", input)

	require.Equal(t, 1, len(result.Errors), "Analyze(%q) errors: %#v", input, result.Errors)

	assert.Equal(t, message, result.Errors[0].Message, "Analyze(%q)", input)
}

// Depth-tracked quote nesting: same-type quote characters nest when
// preceded by whitespace and followed by a non-whitespace, non-quote
// character; otherwise they close. Nested regions stay one word token.
func TestAnalyze_NestedQuoteRegions(t *testing.T) {
	t.Run("nested single quotes are one string token", func(t *testing.T) {
		requireWordWithType(t, "echo 'outer 'inner' end'", "'outer 'inner' end'", TypeSingleQuotedString)
	})

	t.Run("flagship nested word plus separate word", func(t *testing.T) {
		requireWordWithType(t, "echo 'a 'b' c' 'd'", "'a 'b' c'", TypeSingleQuotedString)
		requireWordWithType(t, "echo 'a 'b' c' 'd'", "'d'", TypeSingleQuotedString)
	})

	t.Run("multi-level nesting", func(t *testing.T) {
		requireWordWithType(t, "echo 'l1 'l2 'l3' l2' l1'", "'l1 'l2 'l3' l2' l1'", TypeSingleQuotedString)
	})

	t.Run("nested double quotes", func(t *testing.T) {
		requireWordWithType(t, `echo "outer "inner" end"`, `"outer "inner" end"`, TypeDoubleQuotedString)
	})

	t.Run("nested backticks are one substitution token", func(t *testing.T) {
		requireWordWithType(t, "echo `outer `inner` end`", "`outer `inner` end`", TypeBacktick)
	})

	t.Run("interleaved quote-the-quotes idiom stays one word", func(t *testing.T) {
		requireWordWithType(t, `echo "Hello, "'"'"$USER"'"'"!"`, `"Hello, "'"'"$USER"'"'"!"`, TypeDoubleQuotedString)
	})

	t.Run("backtick substitution inside double quotes", func(t *testing.T) {
		requireWordWithType(t, "echo \"`date`\"", "\"`date`\"", TypeDoubleQuotedString)
	})

	t.Run("nesting applies inside substitution bodies", func(t *testing.T) {
		requireWordWithType(t, "echo $(echo 'a 'b' c')", "$(echo 'a 'b' c')", TypeCommandSubst)
	})
}

// The cost of nesting: a quote that nests leaves the region open, so
// POSIX-valid closers followed by word characters fail loudly with the
// same message the lexer produces.
func TestAnalyze_NestingUnclosedErrors(t *testing.T) {
	tests := []struct {
		input  string
		errMsg string
	}{
		{"echo 'hello 'world", "unclosed single quote"},
		{`echo "Total: "$N`, "unclosed double quote"},
		{"echo ' 'x", "unclosed single quote"},
		{`echo "count: "$(date)`, "unclosed double quote"},
		{"echo `a `b", "unclosed backtick"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			requireOnlyError(t, tt.input, tt.errMsg)
		})
	}
}

// Unclosed constructs report the INNERMOST unclosed construct, matching
// the lexer's error for the same input.
func TestAnalyze_UnclosedInnermostConstruct(t *testing.T) {
	tests := []struct {
		input  string
		errMsg string
	}{
		{`echo "$(a`, "unclosed command substitution $(...)"},
		{"echo $(echo 'a", "unclosed single quote"},
		{"echo `echo \"a", "unclosed double quote"},
		{"echo $(`a", "unclosed backtick"},
		{"echo `a$(b", "unclosed command substitution $(...)"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			requireOnlyError(t, tt.input, tt.errMsg)
		})
	}
}

// Cross-type adjacency: a closed region of one type followed by an
// unclosed region of another type reports the open one.
func TestAnalyze_CrossTypeAdjacentUnclosed(t *testing.T) {
	requireOnlyError(t, `echo "a"'b`, "unclosed single quote")
}

// The spec's escaped-apostrophe workaround is valid, and with a trailing
// pipe the ONLY error is the trailing operator: the escape and the closed
// quote regions must not confuse the quote tracking.
func TestAnalyze_EscapedApostropheWorkaround(t *testing.T) {
	result := Analyze(`echo 'it'\''s working'`)

	assert.True(t, result.Valid, "errors: %#v", result.Errors)

	requireOnlyError(t, `echo 'it'\''s working' |`, "unexpected operator at end")
}

// Consecutive chain operators are invalid, with the parser's exact
// message. A newline is an implicit ;, so an operator starting a
// continuation line after a WORD is consecutive with it; after an
// operator the newline is a line continuation instead.
func TestAnalyze_ConsecutiveOperators(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		errMsg string
	}{
		{"pipe after pipe", "echo a | | b", "consecutive operators: | followed by |"},
		{"pipe after semicolon", "echo a ; | b", "consecutive operators: ; followed by |"},
		{"and after or", "echo a || && b", "consecutive operators: || followed by &&"},
		{"semicolon after semicolon", "echo a ; ; b", "consecutive operators: ; followed by ;"},
		{"operator on next line", "echo a\n| foo", "consecutive operators: ; followed by |"},
		{"operator after continuation line operator", "echo a &&\n|| b", "consecutive operators: && followed by ||"},
		{"redirection followed by operator", "echo > | b", "missing redirection target: > followed by operator |"},
		{"redirection followed by redirection", "echo > > f", "missing redirection target: > followed by operator >"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Analyze(tt.input)

			require.False(t, result.Valid, "Analyze(%q) unexpectedly valid", tt.input)

			found := false
			for _, err := range result.Errors {
				if err.Message == tt.errMsg {
					found = true
					break
				}
			}
			assert.True(t, found, "expected %q, got %#v", tt.errMsg, result.Errors)
		})
	}
}

// Line continuations and plain newline separators stay valid: the
// consecutive-operator check must not fire on them.
func TestAnalyze_NewlineContinuation_Valid(t *testing.T) {
	tests := []string{
		"echo a &&\necho b",
		"echo a |\ntr a b",
		"echo a\necho b",
		"echo a ;\necho b",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			result := Analyze(input)

			assert.True(t, result.Valid, "Analyze(%q) errors: %#v", input, result.Errors)
		})
	}
}

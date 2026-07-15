package syntax

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

// Substitution tracking must survive closed quote constructs earlier in
// the word: the guards use quote parity, not "never seen a quote".
func TestAnalyze_SubstitutionAfterClosedQuotes(t *testing.T) {
	t.Run("quoted prefix then command substitution is one word", func(t *testing.T) {
		result := Analyze("echo 'a'$(date)")

		require.True(t, result.Valid)

		found := false
		for _, tok := range result.Tokens {
			if tok.Value == "'a'$(date)" {
				found = true
				break
			}
		}
		assert.True(t, found, "expected 'a'$(date) to stay one token, got %#v", result.Tokens)
	})

	t.Run("quoted prefix then backticks is one word", func(t *testing.T) {
		result := Analyze("echo 'a'`date`")

		require.True(t, result.Valid)

		found := false
		for _, tok := range result.Tokens {
			if tok.Value == "'a'`date`" {
				found = true
				break
			}
		}
		assert.True(t, found, "expected 'a'`date` to stay one token, got %#v", result.Tokens)
	})
}

// A ) that is quoted inside a substitution body is body text; it must not
// close the substitution.
func TestAnalyze_QuotedParenInsideSubstitution(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"single-quoted close paren", "echo $(echo ')')"},
		{"double-quoted close paren", `echo $(echo "x)y")`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Analyze(tt.input)

			assert.True(t, result.Valid, "expected valid, got errors: %#v", result.Errors)
		})
	}
}

// An unclosed substitution containing quotes reports the substitution
// error, not a bogus quote error.
func TestAnalyze_UnclosedSubstitutionWithQuotedParen(t *testing.T) {
	result := Analyze("echo $(echo ')'")

	require.False(t, result.Valid)

	found := false
	for _, err := range result.Errors {
		if err.Message == "unclosed command substitution $(...)" {
			found = true
			break
		}
	}
	assert.True(t, found, "expected unclosed command substitution error, got %#v", result.Errors)
}

// Comments mirror the lexer: # at a word-start position runs to the next
// newline or end of input; # inside a word or substitution body stays
// literal.
func TestAnalyze_Comments(t *testing.T) {
	t.Run("comment token to end of input", func(t *testing.T) {
		result := Analyze("echo a # rest of line")

		require.True(t, result.Valid)

		last := result.Tokens[len(result.Tokens)-1]
		assert.Equal(t, TypeComment, last.Type)

		assert.Equal(t, "# rest of line", last.Value)
	})

	t.Run("whole-line comment", func(t *testing.T) {
		result := Analyze("# just a comment")

		require.True(t, result.Valid)

		require.Equal(t, 1, len(result.Tokens))

		assert.Equal(t, TypeComment, result.Tokens[0].Type)
	})

	t.Run("comment hides operators and quotes from analysis", func(t *testing.T) {
		result := Analyze("echo ok # trailing | and 'unclosed")

		assert.True(t, result.Valid, "content after # must not produce errors: %#v", result.Errors)
	})

	t.Run("comment ends at newline", func(t *testing.T) {
		result := Analyze("# comment\necho b")

		require.True(t, result.Valid)

		assert.Equal(t, TypeComment, result.Tokens[0].Type)

		assert.Equal(t, "# comment", result.Tokens[0].Value)

		// "echo" after the newline is a Command
		foundCmd := false
		for _, tok := range result.Tokens {
			if tok.Value == "echo" && tok.Type == TypeCommand {
				foundCmd = true
				break
			}
		}
		assert.True(t, foundCmd)
	})

	t.Run("hash inside a word is literal", func(t *testing.T) {
		result := Analyze("echo foo#bar")

		require.True(t, result.Valid)

		found := false
		for _, tok := range result.Tokens {
			if tok.Value == "foo#bar" && tok.Type == TypeArgument {
				found = true
				break
			}
		}
		assert.True(t, found, "foo#bar should stay one argument, got %#v", result.Tokens)
	})

	t.Run("hash inside substitution body is body text", func(t *testing.T) {
		result := Analyze("echo $(a # b)")

		require.True(t, result.Valid)

		found := false
		for _, tok := range result.Tokens {
			if tok.Value == "$(a # b)" {
				found = true
				break
			}
		}
		assert.True(t, found, "expected $(a # b) as one token, got %#v", result.Tokens)
	})
}

// A newline separates commands: the first word after it is a Command
// again, matching the lexer's implicit ;.
func TestAnalyze_NewlineStartsNewCommand(t *testing.T) {
	result := Analyze("echo a\ngrep b")

	require.True(t, result.Valid)

	grepIsCommand := false
	for _, tok := range result.Tokens {
		if tok.Value == "grep" {
			grepIsCommand = tok.Type == TypeCommand
			break
		}
	}
	assert.True(t, grepIsCommand, "grep after a newline must be a Command, got %#v", result.Tokens)
}

// 2>/2>> only match at word-start positions; mid-word the 2 belongs to
// the pending word, matching the lexer (echo a2>f = word a2, operator >).
func TestAnalyze_StderrRedirectContext(t *testing.T) {
	t.Run("word-start 2> is a redirection", func(t *testing.T) {
		result := Analyze("cmd 2>err.log")

		require.True(t, result.Valid)

		found := false
		for _, tok := range result.Tokens {
			if tok.Value == "2>" && tok.Type == TypeRedirection {
				found = true
				break
			}
		}
		assert.True(t, found)
	})

	t.Run("mid-word 2 stays in the word", func(t *testing.T) {
		result := Analyze("echo a2>f")

		require.True(t, result.Valid)

		var values []string
		var types []SemanticType
		for _, tok := range result.Tokens {
			if tok.Type == TypeWhitespace {
				continue
			}
			values = append(values, tok.Value)
			types = append(types, tok.Type)
		}
		assert.Equal(t, []string{"echo", "a2", ">", "f"}, values)

		assert.Equal(t, []SemanticType{TypeCommand, TypeArgument, TypeRedirection, TypeRedirectionTarget}, types)
	})
}

// Trailing-operator and missing-target detection skip trailing whitespace
// and comments (they are not significant tokens).
func TestAnalyze_TrailingOperatorWithTrailingWhitespace(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		errMsg string
	}{
		{"trailing pipe with space", "echo hello | ", "unexpected operator at end"},
		{"trailing and with tabs", "echo hello &&\t\t", "unexpected operator at end"},
		{"trailing redirect with space", "echo > ", "missing redirection target"},
		{"trailing redirect with newline", "echo >\n", "missing redirection target"},
		{"trailing pipe before comment", "echo | # to be continued", "unexpected operator at end"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Analyze(tt.input)

			require.False(t, result.Valid, "expected invalid for %q", tt.input)

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

func TestAnalyze_TrailingSemicolonWithWhitespace_Valid(t *testing.T) {
	result := Analyze("echo hello ; ")

	assert.True(t, result.Valid)
}

// Input starting with a chain operator is an error, matching the parser.
func TestAnalyze_OperatorAtStart(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		errMsg string
	}{
		{"pipe at start", "| foo", "unexpected operator at start: |"},
		{"and at start with leading spaces", "  && foo", "unexpected operator at start: &&"},
		{"or at start", "|| foo", "unexpected operator at start: ||"},
		{"semicolon at start", "; foo", "unexpected operator at start: ;"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Analyze(tt.input)

			require.False(t, result.Valid)

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

// A lone operator is reported once, as an operator-at-start error (the
// parser reports the same input the same way).
func TestAnalyze_LoneOperator_SingleError(t *testing.T) {
	result := Analyze("|")

	require.False(t, result.Valid)

	require.Equal(t, 1, len(result.Errors))

	assert.Equal(t, "unexpected operator at start: |", result.Errors[0].Message)
}

// Redirections may start a command: < in.txt cat is valid.
func TestAnalyze_RedirectionAtStart_Valid(t *testing.T) {
	result := Analyze("< in.txt cat")

	assert.True(t, result.Valid, "got errors: %#v", result.Errors)
}

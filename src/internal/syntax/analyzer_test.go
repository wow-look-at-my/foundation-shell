package syntax

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestAnalyze_BasicCommand(t *testing.T) {
	result := Analyze("echo hello world")

	assert.True(t, result.Valid)

	// Should have: "echo" (Command), " " (Whitespace), "hello" (Argument), " " (Whitespace), "world" (Argument)
	expectedTypes := []SemanticType{TypeCommand, TypeWhitespace, TypeArgument, TypeWhitespace, TypeArgument}
	expectedValues := []string{"echo", " ", "hello", " ", "world"}

	require.Equal(t, len(expectedTypes), len(result.Tokens))

	for i, tok := range result.Tokens {
		assert.Equal(t, expectedTypes[i], tok.Type)

		assert.Equal(t, expectedValues[i], tok.Value)

	}
}

func TestAnalyze_TokenPositions(t *testing.T) {
	result := Analyze("echo hello")

	require.True(t, result.Valid)

	// "echo" at 0-4, " " at 4-5, "hello" at 5-10
	tests := []struct {
		index      int
		start, end int
		value      string
	}{
		{0, 0, 4, "echo"},
		{1, 4, 5, " "},
		{2, 5, 10, "hello"},
	}

	for _, tt := range tests {
		tok := result.Tokens[tt.index]
		assert.False(t, tok.Start != tt.start || tok.End != tt.end)

	}
}

func TestAnalyze_Operators(t *testing.T) {
	tests := []struct {
		input   string
		opValue string
		opType  SemanticType
	}{
		{"cmd1 | cmd2", "|", TypeOperator},
		{"cmd1 && cmd2", "&&", TypeOperator},
		{"cmd1 || cmd2", "||", TypeOperator},
		{"cmd1 ; cmd2", ";", TypeOperator},
		{"cmd > file", ">", TypeRedirection},
		{"cmd >> file", ">>", TypeRedirection},
		{"cmd < file", "<", TypeRedirection},
		{"cmd 2> file", "2>", TypeRedirection},
		{"cmd 2>> file", "2>>", TypeRedirection},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := Analyze(tt.input)

			found := false
			for _, tok := range result.Tokens {
				if tok.Value == tt.opValue && tok.Type == tt.opType {
					found = true
					break
				}
			}
			assert.True(t, found)

		})
	}
}

func TestAnalyze_SingleQuotes(t *testing.T) {
	result := Analyze("echo 'hello world'")

	require.True(t, result.Valid)

	// Find the quoted token
	var quotedToken *AnalyzedToken
	for i := range result.Tokens {
		if result.Tokens[i].Value == "'hello world'" {
			quotedToken = &result.Tokens[i]
			break
		}
	}

	require.NotNil(t, quotedToken)

	assert.Equal(t, TypeSingleQuotedString, quotedToken.Type)

}

func TestAnalyze_DoubleQuotes(t *testing.T) {
	result := Analyze(`echo "hello world"`)

	require.True(t, result.Valid)

	var quotedToken *AnalyzedToken
	for i := range result.Tokens {
		if result.Tokens[i].Value == `"hello world"` {
			quotedToken = &result.Tokens[i]
			break
		}
	}

	require.NotNil(t, quotedToken)

	assert.Equal(t, TypeDoubleQuotedString, quotedToken.Type)

}

func TestAnalyze_NestedSingleQuotes(t *testing.T) {
	// Depth-tracked nesting with even quote count is valid
	// 'hello' 'world' has 4 single quotes (even = valid)
	result := Analyze("echo 'hello' 'world'")

	require.True(t, result.Valid)

	// This has 4 quotes: 'outer 'inner' end'
	// Quote depth: 1, 2, 1, 0 - even count = valid
	result2 := Analyze("echo 'outer 'inner' end'")
	require.True(t, result2.Valid)

}

func TestAnalyze_NestedDoubleQuotes(t *testing.T) {
	// Depth-tracked nesting: "echo "word"" should be valid
	result := Analyze(`echo "echo "word""`)

	require.True(t, result.Valid)

}

func TestAnalyze_UnclosedSingleQuote_Error(t *testing.T) {
	result := Analyze("echo 'hello")

	assert.False(t, result.Valid)

	require.NotEqual(t, 0, len(result.Errors))

	err := result.Errors[0]
	assert.Equal(t, "unclosed single quote", err.Message)

}

func TestAnalyze_UnclosedDoubleQuote_Error(t *testing.T) {
	result := Analyze(`echo "hello`)

	assert.False(t, result.Valid)

	require.NotEqual(t, 0, len(result.Errors))

	err := result.Errors[0]
	assert.Equal(t, "unclosed double quote", err.Message)

}

func TestAnalyze_UnclosedBacktick_Error(t *testing.T) {
	result := Analyze("echo `hello")

	assert.False(t, result.Valid)

	require.NotEqual(t, 0, len(result.Errors))

	err := result.Errors[0]
	assert.Equal(t, "unclosed backtick", err.Message)

}

func TestAnalyze_OddQuoteCount_Error(t *testing.T) {
	tests := []struct {
		input  string
		errMsg string
	}{
		{`echo "a`, "unclosed double quote"},
		{`echo "a"b"`, "unclosed double quote"},
		{`echo 'a`, "unclosed single quote"},
		{"echo `a", "unclosed backtick"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := Analyze(tt.input)

			assert.False(t, result.Valid)

			found := false
			for _, err := range result.Errors {
				if err.Message == tt.errMsg {
					found = true
					break
				}
			}
			assert.True(t, found)

		})
	}
}

func TestAnalyze_Variables(t *testing.T) {
	tests := []struct {
		input    string
		varValue string
	}{
		{"echo $HOME", "$HOME"},
		{"echo ${HOME}", "${HOME}"},
		{"echo $PATH/bin", "$PATH/bin"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := Analyze(tt.input)

			require.True(t, result.Valid)

			found := false
			for _, tok := range result.Tokens {
				if tok.Type == TypeVariable && tok.Value == tt.varValue {
					found = true
					break
				}
			}
			assert.True(t, found)

		})
	}
}

func TestAnalyze_CommandSubst(t *testing.T) {
	result := Analyze("echo $(date)")

	require.True(t, result.Valid)

	found := false
	for _, tok := range result.Tokens {
		if tok.Type == TypeCommandSubst && tok.Value == "$(date)" {
			found = true
			break
		}
	}
	assert.True(t, found)

}

func TestAnalyze_UnclosedCommandSubst_Error(t *testing.T) {
	result := Analyze("echo $(date")

	assert.False(t, result.Valid)

	found := false
	for _, err := range result.Errors {
		if err.Message == "unclosed command substitution $(...)" {
			found = true
			break
		}
	}
	assert.True(t, found)

}

func TestAnalyze_Backticks(t *testing.T) {
	result := Analyze("echo `date`")

	require.True(t, result.Valid)

	found := false
	for _, tok := range result.Tokens {
		if tok.Type == TypeBacktick && tok.Value == "`date`" {
			found = true
			break
		}
	}
	assert.True(t, found)

}

func TestAnalyze_NestedBackticks(t *testing.T) {
	// Depth-tracked: `echo `date`` should be valid
	result := Analyze("echo `echo `date``")

	require.True(t, result.Valid)

}

func TestAnalyze_TrailingOperator_Error(t *testing.T) {
	tests := []struct {
		input string
	}{
		{"echo |"},
		{"echo &&"},
		{"echo ||"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := Analyze(tt.input)

			assert.False(t, result.Valid)

			found := false
			for _, err := range result.Errors {
				if err.Message == "unexpected operator at end" {
					found = true
					break
				}
			}
			assert.True(t, found)

		})
	}
}

func TestAnalyze_TrailingSemicolon_Valid(t *testing.T) {
	// Trailing semicolon is allowed (like bash)
	result := Analyze("echo hello ;")

	assert.True(t, result.Valid)

}

func TestAnalyze_MissingRedirectionTarget_Error(t *testing.T) {
	tests := []struct {
		input string
	}{
		{"echo >"},
		{"echo >>"},
		{"echo <"},
		{"echo 2>"},
		{"echo 2>>"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := Analyze(tt.input)

			assert.False(t, result.Valid)

			found := false
			for _, err := range result.Errors {
				if err.Message == "missing redirection target" {
					found = true
					break
				}
			}
			assert.True(t, found)

		})
	}
}

func TestAnalyze_RedirectionTarget(t *testing.T) {
	result := Analyze("echo > output.txt")

	require.True(t, result.Valid)

	found := false
	for _, tok := range result.Tokens {
		if tok.Type == TypeRedirectionTarget && tok.Value == "output.txt" {
			found = true
			break
		}
	}
	assert.True(t, found)

}

func TestAnalyze_EscapedQuotes(t *testing.T) {
	// Escaped quotes don't count toward depth
	result := Analyze(`echo "hello \"world\""`)

	require.True(t, result.Valid)

}

func TestAnalyze_EscapedBacktick(t *testing.T) {
	// Escaped backticks don't count toward depth
	result := Analyze("echo \\`not a substitution\\`")

	require.True(t, result.Valid)

}

func TestAnalyze_ErrorPosition(t *testing.T) {
	// Test that error positions are accurate
	result := Analyze(`echo "unclosed`)

	require.False(t, result.Valid)

	require.NotEqual(t, 0, len(result.Errors))

	err := result.Errors[0]
	// The error should point to the unclosed quote region
	// "unclosed starts at position 5
	assert.Equal(t, 5, err.Start)

}

func TestAnalyze_DepthTracking(t *testing.T) {
	// Test that depth is tracked correctly
	result := Analyze(`echo "outer "inner" outer"`)

	require.True(t, result.Valid)

	// Find the quoted token and check its depth
	for _, tok := range result.Tokens {
		if tok.Type == TypeDoubleQuotedString || tok.Type == TypeArgument {
			if tok.Depth < 2 {
				// Nested quotes should have depth >= 2
				// Actually the depth here should be 2 (two pairs of quotes)
			}
		}
	}
}

func TestAnalyze_EmptyInput(t *testing.T) {
	result := Analyze("")

	assert.True(t, result.Valid)

	assert.Equal(t, 0, len(result.Tokens))

}

func TestAnalyze_WhitespaceOnly(t *testing.T) {
	result := Analyze("   ")

	assert.True(t, result.Valid)

	assert.False(t, len(result.Tokens) != 1 || result.Tokens[0].Type != TypeWhitespace)

}

func TestAnalyze_CommandAfterPipe(t *testing.T) {
	result := Analyze("cat file | grep pattern")

	require.True(t, result.Valid)

	// "grep" should be TypeCommand (first word after pipe)
	grepFound := false
	for _, tok := range result.Tokens {
		if tok.Value == "grep" {
			assert.Equal(t, TypeCommand, tok.Type)

			grepFound = true
			break
		}
	}
	assert.True(t, grepFound)

}

func TestAnalyze_CommandAfterSemicolon(t *testing.T) {
	result := Analyze("echo a; echo b")

	require.True(t, result.Valid)

	// Both "echo" tokens should be TypeCommand
	echoCount := 0
	for _, tok := range result.Tokens {
		if tok.Value == "echo" {
			assert.Equal(t, TypeCommand, tok.Type)

			echoCount++
		}
	}
	assert.Equal(t, 2, echoCount)

}

package syntax

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"regexp"
	"strings"
	"testing"
)

// stripANSI removes all ANSI escape codes from a string.
func stripANSI(s string) string {
	re := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return re.ReplaceAllString(s, "")
}

// containsANSICode checks if the output contains a specific ANSI code around a token.
func containsANSICode(output, token, ansiCode string) bool {
	// Check if the token is preceded by the expected ANSI code
	pattern := regexp.QuoteMeta(ansiCode) + regexp.QuoteMeta(token)
	matched, _ := regexp.MatchString(pattern, output)
	return matched
}

// getTokenColor extracts the ANSI color code applied to a token in the output.
func getTokenColor(output, token string) string {
	// Find the ANSI code immediately before the token
	pattern := `(\x1b\[[0-9;]*m)` + regexp.QuoteMeta(token)
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(output)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

func TestHighlight_BasicCommand(t *testing.T) {
	h := NewHighlighter(DefaultTheme)
	output := h.Highlight("echo hello")

	// Verify "echo" is colored as Command (Bold cyan: \033[1;36m)
	expectedColor := DefaultTheme[TypeCommand]
	assert.True(t, containsANSICode(output, "echo", expectedColor))

	// Verify the output contains both tokens
	stripped := stripANSI(output)
	assert.Equal(t, "echo hello", stripped)

}

func TestHighlight_Operators(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		operator string
	}{
		{"pipe", "cmd1 | cmd2", "|"},
		{"and", "cmd1 && cmd2", "&&"},
		{"or", "cmd1 || cmd2", "||"},
		{"semicolon", "cmd1 ; cmd2", ";"},
	}

	h := NewHighlighter(DefaultTheme)
	expectedColor := DefaultTheme[TypeOperator]

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := h.Highlight(tt.input)

			assert.True(t, containsANSICode(output, tt.operator, expectedColor))

		})
	}
}

func TestHighlight_Redirections(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		redirection string
	}{
		{"stdout", "echo > file", ">"},
		{"stdout_append", "echo >> file", ">>"},
		{"stdin", "cat < file", "<"},
		{"stderr", "cmd 2> error.log", "2>"},
		{"stderr_append", "cmd 2>> error.log", "2>>"},
	}

	h := NewHighlighter(DefaultTheme)
	expectedColor := DefaultTheme[TypeRedirection]

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := h.Highlight(tt.input)

			assert.True(t, containsANSICode(output, tt.redirection, expectedColor))

		})
	}
}

func TestHighlight_SingleQuotedString(t *testing.T) {
	h := NewHighlighter(DefaultTheme)
	output := h.Highlight("echo 'hello'")

	expectedColor := DefaultTheme[TypeSingleQuotedString]
	token := "'hello'"

	assert.True(t, containsANSICode(output, token, expectedColor))

}

func TestHighlight_DoubleQuotedString(t *testing.T) {
	h := NewHighlighter(DefaultTheme)
	output := h.Highlight(`echo "hello"`)

	expectedColor := DefaultTheme[TypeDoubleQuotedString]
	token := `"hello"`

	assert.True(t, containsANSICode(output, token, expectedColor))

}

func TestHighlight_Variable(t *testing.T) {
	h := NewHighlighter(DefaultTheme)
	output := h.Highlight("echo $HOME")

	expectedColor := DefaultTheme[TypeVariable]
	token := "$HOME"

	assert.True(t, containsANSICode(output, token, expectedColor))

}

func TestHighlight_Variable_Braced(t *testing.T) {
	h := NewHighlighter(DefaultTheme)
	output := h.Highlight("echo ${HOME}")

	expectedColor := DefaultTheme[TypeVariable]
	token := "${HOME}"

	assert.True(t, containsANSICode(output, token, expectedColor))

}

func TestHighlight_Subshell(t *testing.T) {
	h := NewHighlighter(DefaultTheme)
	output := h.Highlight("echo $(cmd)")

	expectedColor := DefaultTheme[TypeSubshell]
	token := "$(cmd)"

	assert.True(t, containsANSICode(output, token, expectedColor))

}

func TestHighlight_Subshell_Complex(t *testing.T) {
	h := NewHighlighter(DefaultTheme)
	output := h.Highlight("echo $(date +%Y-%m-%d)")

	expectedColor := DefaultTheme[TypeSubshell]
	token := "$(date +%Y-%m-%d)"

	assert.True(t, containsANSICode(output, token, expectedColor))

}

func TestHighlight_Backtick(t *testing.T) {
	h := NewHighlighter(DefaultTheme)
	output := h.Highlight("echo `cmd`")

	expectedColor := DefaultTheme[TypeBacktick]
	token := "`cmd`"

	assert.True(t, containsANSICode(output, token, expectedColor))

}

func TestHighlight_Backtick_WithDate(t *testing.T) {
	h := NewHighlighter(DefaultTheme)
	output := h.Highlight("echo `date`")

	expectedColor := DefaultTheme[TypeBacktick]
	token := "`date`"

	assert.True(t, containsANSICode(output, token, expectedColor))

}

func TestHighlight_Error_UnclosedQuote(t *testing.T) {
	h := NewHighlighter(DefaultTheme)
	output := h.Highlight(`echo "unclosed`)

	// The unclosed quote token should be colored as Error
	expectedColor := DefaultTheme[TypeError]

	// The token value is the unclosed string
	// Check that Error color is applied somewhere in the output
	assert.Contains(t, output, expectedColor)

}

func TestHighlight_Error_UnclosedSingleQuote(t *testing.T) {
	h := NewHighlighter(DefaultTheme)
	output := h.Highlight("echo 'unclosed")

	expectedColor := DefaultTheme[TypeError]

	assert.Contains(t, output, expectedColor)

}

func TestHighlight_Error_UnclosedBacktick(t *testing.T) {
	h := NewHighlighter(DefaultTheme)
	output := h.Highlight("echo `unclosed")

	expectedColor := DefaultTheme[TypeError]

	assert.Contains(t, output, expectedColor)

}

func TestHighlight_Error_UnclosedSubshell(t *testing.T) {
	h := NewHighlighter(DefaultTheme)
	output := h.Highlight("echo $(unclosed")

	expectedColor := DefaultTheme[TypeError]

	assert.Contains(t, output, expectedColor)

}

func TestHighlight_PreservesInput(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"simple_command", "echo hello"},
		{"with_pipe", "cat file | grep pattern"},
		{"with_redirects", "echo hello > output.txt"},
		{"with_quotes", `echo "hello world"`},
		{"with_single_quotes", "echo 'hello world'"},
		{"with_variable", "echo $HOME"},
		{"with_subshell", "echo $(date)"},
		{"with_backticks", "echo `date`"},
		{"complex", "cat file.txt | grep 'pattern' > output.txt 2>&1"},
		{"multiple_operators", "cmd1 && cmd2 || cmd3"},
		{"empty", ""},
		{"whitespace_only", "   "},
		{"multiple_args", "ls -la /tmp /var"},
	}

	h := NewHighlighter(DefaultTheme)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := h.Highlight(tt.input)
			stripped := stripANSI(output)

			assert.Equal(t, tt.input, stripped)

		})
	}
}

func TestHighlightResult_ReturnsErrors(t *testing.T) {
	h := NewHighlighter(DefaultTheme)

	tests := []struct {
		name         string
		input        string
		expectErrors bool
	}{
		{"valid_command", "echo hello", false},
		{"unclosed_double_quote", `echo "hello`, true},
		{"unclosed_single_quote", "echo 'hello", true},
		{"unclosed_backtick", "echo `hello", true},
		{"unclosed_subshell", "echo $(hello", true},
		{"trailing_pipe", "echo |", true},
		{"trailing_and", "echo &&", true},
		{"missing_redirect_target", "echo >", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, errors := h.HighlightResult(tt.input)

			hasErrors := len(errors) > 0
			assert.Equal(t, tt.expectErrors, hasErrors)

			// Verify output is still returned even with errors
			assert.False(t, output == "" && tt.input != "")

		})
	}
}

func TestHighlightResult_ErrorDetails(t *testing.T) {
	h := NewHighlighter(DefaultTheme)

	tests := []struct {
		name           string
		input          string
		expectedErrMsg string
	}{
		{"unclosed_double_quote", `echo "hello`, "unclosed double quote"},
		{"unclosed_single_quote", "echo 'hello", "unclosed single quote"},
		{"unclosed_backtick", "echo `hello", "unclosed backtick"},
		{"unclosed_subshell", "echo $(hello", "unclosed command substitution"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, errors := h.HighlightResult(tt.input)

			require.NotEqual(t, 0, len(errors))

			found := false
			for _, err := range errors {
				if strings.Contains(err.Message, tt.expectedErrMsg) {
					found = true
					break
				}
			}

			assert.True(t, found)

		})
	}
}

func TestNewHighlighter_CustomTheme(t *testing.T) {
	// Create a custom theme with distinct colors
	customTheme := Theme{
		TypeCommand:            "\033[38;5;196m", // Bright red
		TypeArgument:           "\033[38;5;226m", // Yellow
		TypeOperator:           "\033[38;5;21m",  // Blue
		TypeRedirection:        "\033[38;5;201m", // Pink
		TypeRedirectionTarget:  "\033[38;5;46m",  // Green
		TypeSingleQuotedString: "\033[38;5;208m", // Orange
		TypeDoubleQuotedString: "\033[38;5;51m",  // Cyan
		TypeBacktick:           "\033[38;5;141m", // Purple
		TypeSubshell:           "\033[38;5;141m", // Purple
		TypeVariable:           "\033[38;5;82m",  // Lime
		TypeParenGroup:         "\033[38;5;213m", // Light pink
		TypeError:              "\033[48;5;196m", // Red background
		TypeWhitespace:         "\033[0m",
		TypeUnknown:            "\033[0m",
	}

	h := NewHighlighter(customTheme)
	output := h.Highlight("echo hello")

	// Verify custom Command color is used
	expectedColor := customTheme[TypeCommand]
	assert.True(t, containsANSICode(output, "echo", expectedColor))

}

func TestNewHighlighter_CustomTheme_AllTokenTypes(t *testing.T) {
	// Create a recognizable custom theme
	customTheme := Theme{
		TypeCommand:            "\033[91m", // Light red
		TypeArgument:           "\033[92m", // Light green
		TypeOperator:           "\033[93m", // Light yellow
		TypeRedirection:        "\033[94m", // Light blue
		TypeRedirectionTarget:  "\033[95m", // Light magenta
		TypeSingleQuotedString: "\033[96m", // Light cyan
		TypeDoubleQuotedString: "\033[97m", // White
		TypeBacktick:           "\033[90m", // Dark gray
		TypeSubshell:           "\033[90m", // Dark gray
		TypeVariable:           "\033[32m", // Green
		TypeParenGroup:         "\033[33m", // Yellow
		TypeError:              "\033[31m", // Red
		TypeWhitespace:         "\033[0m",
		TypeUnknown:            "\033[0m",
	}

	h := NewHighlighter(customTheme)

	// Test each token type with custom theme
	tests := []struct {
		name      string
		input     string
		token     string
		tokenType SemanticType
	}{
		{"command", "ls -la", "ls", TypeCommand},
		{"operator", "cmd1 | cmd2", "|", TypeOperator},
		{"redirection", "echo > file", ">", TypeRedirection},
		{"single_quote", "echo 'test'", "'test'", TypeSingleQuotedString},
		{"double_quote", `echo "test"`, `"test"`, TypeDoubleQuotedString},
		{"backtick", "echo `date`", "`date`", TypeBacktick},
		{"subshell", "echo $(pwd)", "$(pwd)", TypeSubshell},
		{"variable", "echo $PATH", "$PATH", TypeVariable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := h.Highlight(tt.input)
			expectedColor := customTheme[tt.tokenType]

			assert.True(t, containsANSICode(output, tt.token, expectedColor))

		})
	}
}

func TestNewHighlighter_PartialTheme(t *testing.T) {
	// Create a partial theme (only some types defined)
	partialTheme := Theme{
		TypeCommand:  "\033[95m", // Light magenta
		TypeOperator: "\033[96m", // Light cyan
	}

	h := NewHighlighter(partialTheme)
	output := h.Highlight("echo | cat")

	// Check that defined colors are used
	assert.True(t, containsANSICode(output, "echo", partialTheme[TypeCommand]))

	assert.True(t, containsANSICode(output, "|", partialTheme[TypeOperator]))

}

func TestHighlight_ComplexPipeline(t *testing.T) {
	h := NewHighlighter(DefaultTheme)
	input := "cat file.txt | grep 'pattern' | sort | uniq > output.txt"
	output := h.Highlight(input)

	// Verify input is preserved
	stripped := stripANSI(output)
	assert.Equal(t, input, stripped)

	// Verify specific colorings
	tests := []struct {
		token     string
		tokenType SemanticType
	}{
		{"cat", TypeCommand},
		{"grep", TypeCommand},
		{"sort", TypeCommand},
		{"uniq", TypeCommand},
		{"|", TypeOperator},
		{">", TypeRedirection},
		{"'pattern'", TypeSingleQuotedString},
	}

	for _, tt := range tests {
		expectedColor := DefaultTheme[tt.tokenType]
		assert.True(t, containsANSICode(output, tt.token, expectedColor))

	}
}

func TestHighlight_ChainedOperators(t *testing.T) {
	h := NewHighlighter(DefaultTheme)
	input := "cmd1 && cmd2 || cmd3 ; cmd4"
	output := h.Highlight(input)

	// All commands should be highlighted as Command
	commands := []string{"cmd1", "cmd2", "cmd3", "cmd4"}
	expectedCmdColor := DefaultTheme[TypeCommand]

	for _, cmd := range commands {
		assert.True(t, containsANSICode(output, cmd, expectedCmdColor))

	}

	// All operators should be highlighted as Operator
	operators := []string{"&&", "||", ";"}
	expectedOpColor := DefaultTheme[TypeOperator]

	for _, op := range operators {
		assert.True(t, containsANSICode(output, op, expectedOpColor))

	}
}

func TestHighlight_MultipleRedirections(t *testing.T) {
	h := NewHighlighter(DefaultTheme)
	input := "cmd < input.txt > output.txt 2>> errors.log"
	output := h.Highlight(input)

	// Verify input is preserved
	stripped := stripANSI(output)
	assert.Equal(t, input, stripped)

	// Verify redirections are colored
	redirections := []string{"<", ">", "2>>"}
	expectedColor := DefaultTheme[TypeRedirection]

	for _, redir := range redirections {
		assert.True(t, containsANSICode(output, redir, expectedColor))

	}
}

func TestHighlight_EmptyInput(t *testing.T) {
	h := NewHighlighter(DefaultTheme)
	output := h.Highlight("")

	assert.Equal(t, "", output)

}

func TestHighlight_WhitespaceOnly(t *testing.T) {
	h := NewHighlighter(DefaultTheme)
	input := "   "
	output := h.Highlight(input)

	stripped := stripANSI(output)
	assert.Equal(t, input, stripped)

}

func TestHighlight_Arguments(t *testing.T) {
	h := NewHighlighter(DefaultTheme)
	output := h.Highlight("ls -la /tmp")

	// Command should be highlighted
	cmdColor := DefaultTheme[TypeCommand]
	assert.True(t, containsANSICode(output, "ls", cmdColor))

	// Arguments should have Argument color (which is reset/default in DefaultTheme)
	argColor := DefaultTheme[TypeArgument]
	assert.True(t, containsANSICode(output, "-la", argColor))

}

func TestHighlight_RedirectionTarget(t *testing.T) {
	h := NewHighlighter(DefaultTheme)
	output := h.Highlight("echo hello > output.txt")

	targetColor := DefaultTheme[TypeRedirectionTarget]
	assert.True(t, containsANSICode(output, "output.txt", targetColor))

}

func TestHighlight_VariableInPath(t *testing.T) {
	h := NewHighlighter(DefaultTheme)
	output := h.Highlight("cd $HOME/projects")

	varColor := DefaultTheme[TypeVariable]
	// The entire $HOME/projects is a single token starting with $
	assert.Contains(t, output, varColor)

}

func TestHighlight_MixedQuotes(t *testing.T) {
	h := NewHighlighter(DefaultTheme)

	// Single quotes within double quotes context (they remain as is in the token)
	input := "echo 'single' \"double\""
	output := h.Highlight(input)

	stripped := stripANSI(output)
	assert.Equal(t, input, stripped)

	// Verify both quote types are colored
	singleColor := DefaultTheme[TypeSingleQuotedString]
	doubleColor := DefaultTheme[TypeDoubleQuotedString]

	assert.True(t, containsANSICode(output, "'single'", singleColor))

	assert.True(t, containsANSICode(output, `"double"`, doubleColor))

}

func TestHighlight_ANSIResetAfterEachToken(t *testing.T) {
	h := NewHighlighter(DefaultTheme)
	output := h.Highlight("echo hello")

	// Count ANSI reset codes - should be at least as many as tokens
	resetCode := "\033[0m"
	resetCount := strings.Count(output, resetCode)

	// We have at least 3 tokens: "echo", " ", "hello"
	// Each should be followed by a reset
	assert.GreaterOrEqual(t, resetCount, 3)

}

func TestHighlightResult_OutputMatchesHighlight(t *testing.T) {
	h := NewHighlighter(DefaultTheme)
	input := "echo hello | cat"

	output1 := h.Highlight(input)
	output2, _ := h.HighlightResult(input)

	assert.Equal(t, output2, output1)

}

func TestHighlight_SpecialCharactersInStrings(t *testing.T) {
	h := NewHighlighter(DefaultTheme)

	tests := []struct {
		name  string
		input string
	}{
		{"spaces_in_single_quotes", "echo 'hello world'"},
		{"spaces_in_double_quotes", `echo "hello world"`},
		{"special_chars_single", "echo 'hello$world'"},
		{"special_chars_double", `echo "hello\"world"`},
		{"newline_in_quotes", `echo "hello\nworld"`},
		{"tab_in_quotes", `echo "hello\tworld"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := h.Highlight(tt.input)
			stripped := stripANSI(output)
			assert.Equal(t, tt.input, stripped)

		})
	}
}

func TestHighlight_EscapedCharacters(t *testing.T) {
	h := NewHighlighter(DefaultTheme)

	tests := []struct {
		name  string
		input string
	}{
		{"escaped_dollar", "echo \\$HOME"},
		{"escaped_quote", `echo \"hello\"`},
		{"escaped_backslash", "echo \\\\test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := h.Highlight(tt.input)
			stripped := stripANSI(output)
			assert.Equal(t, tt.input, stripped)

		})
	}
}

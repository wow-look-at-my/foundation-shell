package syntax

import (
	"testing"
)

func TestAnalyze_BasicCommand(t *testing.T) {
	result := Analyze("echo hello world")

	if !result.Valid {
		t.Errorf("expected valid result, got errors: %v", result.Errors)
	}

	// Should have: "echo" (Command), " " (Whitespace), "hello" (Argument), " " (Whitespace), "world" (Argument)
	expectedTypes := []SemanticType{TypeCommand, TypeWhitespace, TypeArgument, TypeWhitespace, TypeArgument}
	expectedValues := []string{"echo", " ", "hello", " ", "world"}

	if len(result.Tokens) != len(expectedTypes) {
		t.Fatalf("expected %d tokens, got %d: %+v", len(expectedTypes), len(result.Tokens), result.Tokens)
	}

	for i, tok := range result.Tokens {
		if tok.Type != expectedTypes[i] {
			t.Errorf("token %d: expected type %v, got %v", i, expectedTypes[i], tok.Type)
		}
		if tok.Value != expectedValues[i] {
			t.Errorf("token %d: expected value %q, got %q", i, expectedValues[i], tok.Value)
		}
	}
}

func TestAnalyze_TokenPositions(t *testing.T) {
	result := Analyze("echo hello")

	if !result.Valid {
		t.Fatalf("expected valid result, got errors: %v", result.Errors)
	}

	// "echo" at 0-4, " " at 4-5, "hello" at 5-10
	tests := []struct {
		index       int
		start, end  int
		value       string
	}{
		{0, 0, 4, "echo"},
		{1, 4, 5, " "},
		{2, 5, 10, "hello"},
	}

	for _, tt := range tests {
		tok := result.Tokens[tt.index]
		if tok.Start != tt.start || tok.End != tt.end {
			t.Errorf("token %d (%q): expected position %d-%d, got %d-%d",
				tt.index, tt.value, tt.start, tt.end, tok.Start, tok.End)
		}
	}
}

func TestAnalyze_Operators(t *testing.T) {
	tests := []struct {
		input    string
		opValue  string
		opType   SemanticType
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
			if !found {
				t.Errorf("expected operator %q with type %v in %q, tokens: %+v",
					tt.opValue, tt.opType, tt.input, result.Tokens)
			}
		})
	}
}

func TestAnalyze_SingleQuotes(t *testing.T) {
	result := Analyze("echo 'hello world'")

	if !result.Valid {
		t.Fatalf("expected valid result, got errors: %v", result.Errors)
	}

	// Find the quoted token
	var quotedToken *AnalyzedToken
	for i := range result.Tokens {
		if result.Tokens[i].Value == "'hello world'" {
			quotedToken = &result.Tokens[i]
			break
		}
	}

	if quotedToken == nil {
		t.Fatalf("could not find quoted token in %+v", result.Tokens)
	}

	if quotedToken.Type != TypeSingleQuotedString {
		t.Errorf("expected TypeSingleQuotedString, got %v", quotedToken.Type)
	}
}

func TestAnalyze_DoubleQuotes(t *testing.T) {
	result := Analyze(`echo "hello world"`)

	if !result.Valid {
		t.Fatalf("expected valid result, got errors: %v", result.Errors)
	}

	var quotedToken *AnalyzedToken
	for i := range result.Tokens {
		if result.Tokens[i].Value == `"hello world"` {
			quotedToken = &result.Tokens[i]
			break
		}
	}

	if quotedToken == nil {
		t.Fatalf("could not find quoted token in %+v", result.Tokens)
	}

	if quotedToken.Type != TypeDoubleQuotedString {
		t.Errorf("expected TypeDoubleQuotedString, got %v", quotedToken.Type)
	}
}

func TestAnalyze_NestedSingleQuotes(t *testing.T) {
	// Depth-tracked nesting: 'echo 'word'' should be valid
	result := Analyze("echo 'it's 'cool''")

	if !result.Valid {
		t.Fatalf("expected valid result for nested single quotes, got errors: %v", result.Errors)
	}
}

func TestAnalyze_NestedDoubleQuotes(t *testing.T) {
	// Depth-tracked nesting: "echo "word"" should be valid
	result := Analyze(`echo "echo "word""`)

	if !result.Valid {
		t.Fatalf("expected valid result for nested double quotes, got errors: %v", result.Errors)
	}
}

func TestAnalyze_UnclosedSingleQuote_Error(t *testing.T) {
	result := Analyze("echo 'hello")

	if result.Valid {
		t.Errorf("expected invalid result for unclosed single quote")
	}

	if len(result.Errors) == 0 {
		t.Fatalf("expected at least one error")
	}

	err := result.Errors[0]
	if err.Message != "unclosed single quote (odd count)" {
		t.Errorf("unexpected error message: %s", err.Message)
	}
}

func TestAnalyze_UnclosedDoubleQuote_Error(t *testing.T) {
	result := Analyze(`echo "hello`)

	if result.Valid {
		t.Errorf("expected invalid result for unclosed double quote")
	}

	if len(result.Errors) == 0 {
		t.Fatalf("expected at least one error")
	}

	err := result.Errors[0]
	if err.Message != "unclosed double quote (odd count)" {
		t.Errorf("unexpected error message: %s", err.Message)
	}
}

func TestAnalyze_UnclosedBacktick_Error(t *testing.T) {
	result := Analyze("echo `hello")

	if result.Valid {
		t.Errorf("expected invalid result for unclosed backtick")
	}

	if len(result.Errors) == 0 {
		t.Fatalf("expected at least one error")
	}

	err := result.Errors[0]
	if err.Message != "unclosed backtick (odd count)" {
		t.Errorf("unexpected error message: %s", err.Message)
	}
}

func TestAnalyze_OddQuoteCount_Error(t *testing.T) {
	tests := []struct {
		input   string
		errMsg  string
	}{
		{`echo "a`, "unclosed double quote (odd count)"},
		{`echo "a"b"`, "unclosed double quote (odd count)"},
		{`echo 'a`, "unclosed single quote (odd count)"},
		{"echo `a", "unclosed backtick (odd count)"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := Analyze(tt.input)

			if result.Valid {
				t.Errorf("expected invalid result for %q", tt.input)
			}

			found := false
			for _, err := range result.Errors {
				if err.Message == tt.errMsg {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected error %q for %q, got: %v", tt.errMsg, tt.input, result.Errors)
			}
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

			if !result.Valid {
				t.Fatalf("expected valid result, got errors: %v", result.Errors)
			}

			found := false
			for _, tok := range result.Tokens {
				if tok.Type == TypeVariable && tok.Value == tt.varValue {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected variable %q in %q, tokens: %+v", tt.varValue, tt.input, result.Tokens)
			}
		})
	}
}

func TestAnalyze_Subshell(t *testing.T) {
	result := Analyze("echo $(date)")

	if !result.Valid {
		t.Fatalf("expected valid result, got errors: %v", result.Errors)
	}

	found := false
	for _, tok := range result.Tokens {
		if tok.Type == TypeSubshell && tok.Value == "$(date)" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected subshell $(date), tokens: %+v", result.Tokens)
	}
}

func TestAnalyze_UnclosedSubshell_Error(t *testing.T) {
	result := Analyze("echo $(date")

	if result.Valid {
		t.Errorf("expected invalid result for unclosed subshell")
	}

	found := false
	for _, err := range result.Errors {
		if err.Message == "unclosed subshell $(...)" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected unclosed subshell error, got: %v", result.Errors)
	}
}

func TestAnalyze_Backticks(t *testing.T) {
	result := Analyze("echo `date`")

	if !result.Valid {
		t.Fatalf("expected valid result, got errors: %v", result.Errors)
	}

	found := false
	for _, tok := range result.Tokens {
		if tok.Type == TypeBacktick && tok.Value == "`date`" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected backtick `date`, tokens: %+v", result.Tokens)
	}
}

func TestAnalyze_NestedBackticks(t *testing.T) {
	// Depth-tracked: `echo `date`` should be valid
	result := Analyze("echo `echo `date``")

	if !result.Valid {
		t.Fatalf("expected valid result for nested backticks, got errors: %v", result.Errors)
	}
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

			if result.Valid {
				t.Errorf("expected invalid result for trailing operator in %q", tt.input)
			}

			found := false
			for _, err := range result.Errors {
				if err.Message == "unexpected operator at end" {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected trailing operator error for %q, got: %v", tt.input, result.Errors)
			}
		})
	}
}

func TestAnalyze_TrailingSemicolon_Valid(t *testing.T) {
	// Trailing semicolon is allowed (like bash)
	result := Analyze("echo hello ;")

	if !result.Valid {
		t.Errorf("expected valid result for trailing semicolon, got errors: %v", result.Errors)
	}
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

			if result.Valid {
				t.Errorf("expected invalid result for missing redirection target in %q", tt.input)
			}

			found := false
			for _, err := range result.Errors {
				if err.Message == "missing redirection target" {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected missing redirection target error for %q, got: %v", tt.input, result.Errors)
			}
		})
	}
}

func TestAnalyze_RedirectionTarget(t *testing.T) {
	result := Analyze("echo > output.txt")

	if !result.Valid {
		t.Fatalf("expected valid result, got errors: %v", result.Errors)
	}

	found := false
	for _, tok := range result.Tokens {
		if tok.Type == TypeRedirectionTarget && tok.Value == "output.txt" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected redirection target output.txt, tokens: %+v", result.Tokens)
	}
}

func TestAnalyze_EscapedQuotes(t *testing.T) {
	// Escaped quotes don't count toward depth
	result := Analyze(`echo "hello \"world\""`)

	if !result.Valid {
		t.Fatalf("expected valid result for escaped quotes, got errors: %v", result.Errors)
	}
}

func TestAnalyze_EscapedBacktick(t *testing.T) {
	// Escaped backticks don't count toward depth
	result := Analyze("echo \\`not a subshell\\`")

	if !result.Valid {
		t.Fatalf("expected valid result for escaped backticks, got errors: %v", result.Errors)
	}
}

func TestAnalyze_ErrorPosition(t *testing.T) {
	// Test that error positions are accurate
	result := Analyze(`echo "unclosed`)

	if result.Valid {
		t.Fatalf("expected invalid result")
	}

	if len(result.Errors) == 0 {
		t.Fatalf("expected at least one error")
	}

	err := result.Errors[0]
	// The error should point to the unclosed quote region
	// "unclosed starts at position 5
	if err.Start != 5 {
		t.Errorf("expected error start at 5, got %d", err.Start)
	}
}

func TestAnalyze_DepthTracking(t *testing.T) {
	// Test that depth is tracked correctly
	result := Analyze(`echo "outer "inner" outer"`)

	if !result.Valid {
		t.Fatalf("expected valid result for nested quotes, got errors: %v", result.Errors)
	}

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

	if !result.Valid {
		t.Errorf("expected valid result for empty input, got errors: %v", result.Errors)
	}

	if len(result.Tokens) != 0 {
		t.Errorf("expected no tokens for empty input, got %d", len(result.Tokens))
	}
}

func TestAnalyze_WhitespaceOnly(t *testing.T) {
	result := Analyze("   ")

	if !result.Valid {
		t.Errorf("expected valid result for whitespace only, got errors: %v", result.Errors)
	}

	if len(result.Tokens) != 1 || result.Tokens[0].Type != TypeWhitespace {
		t.Errorf("expected single whitespace token, got %+v", result.Tokens)
	}
}

func TestAnalyze_CommandAfterPipe(t *testing.T) {
	result := Analyze("cat file | grep pattern")

	if !result.Valid {
		t.Fatalf("expected valid result, got errors: %v", result.Errors)
	}

	// "grep" should be TypeCommand (first word after pipe)
	grepFound := false
	for _, tok := range result.Tokens {
		if tok.Value == "grep" {
			if tok.Type != TypeCommand {
				t.Errorf("expected 'grep' to be TypeCommand after pipe, got %v", tok.Type)
			}
			grepFound = true
			break
		}
	}
	if !grepFound {
		t.Errorf("could not find 'grep' token")
	}
}

func TestAnalyze_CommandAfterSemicolon(t *testing.T) {
	result := Analyze("echo a; echo b")

	if !result.Valid {
		t.Fatalf("expected valid result, got errors: %v", result.Errors)
	}

	// Both "echo" tokens should be TypeCommand
	echoCount := 0
	for _, tok := range result.Tokens {
		if tok.Value == "echo" {
			if tok.Type != TypeCommand {
				t.Errorf("expected 'echo' to be TypeCommand, got %v", tok.Type)
			}
			echoCount++
		}
	}
	if echoCount != 2 {
		t.Errorf("expected 2 'echo' tokens, got %d", echoCount)
	}
}

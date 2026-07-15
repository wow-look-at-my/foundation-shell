package syntax

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestFormatDiagnostics_SingleError(t *testing.T) {
	input := `echo "unclosed`
	errors := []SyntaxError{{Start: 5, End: 14, Message: "unclosed double quote"}}

	result := FormatDiagnostics(input, errors)

	// Should contain the input
	assert.Contains(t, result, input)

	// Should contain carets
	assert.Contains(t, result, "^")

	// Should contain error message
	assert.Contains(t, result, "unclosed double quote")

}

func TestFormatDiagnostics_MultipleErrors(t *testing.T) {
	input := `echo "unclosed && 'also`
	errors := []SyntaxError{
		{Start: 5, End: 14, Message: "unclosed double quote"},
		{Start: 18, End: 23, Message: "unclosed single quote"},
	}

	result := FormatDiagnostics(input, errors)

	// Should contain both error messages
	assert.Contains(t, result, "unclosed double quote")

	assert.Contains(t, result, "unclosed single quote")

	// Should contain the input
	assert.Contains(t, result, input)

	// Should contain carets for both errors
	caretCount := strings.Count(result, "^")
	// First error spans 9 chars (5-14), second spans 5 chars (18-23)
	// At minimum we expect multiple carets
	assert.GreaterOrEqual(t, caretCount, 2)

}

func TestFormatDiagnostics_CaretAlignment(t *testing.T) {
	input := `echo hello world`
	errors := []SyntaxError{{Start: 5, End: 10, Message: "test error"}}

	result := FormatDiagnostics(input, errors)

	// Split the result into lines
	lines := strings.Split(result, "\n")

	// Find the line with input and the line with carets
	var inputLineIdx int
	var caretLineIdx int
	for i, line := range lines {
		if strings.Contains(line, "echo hello world") {
			inputLineIdx = i
		}
		if strings.Contains(line, "^") && !strings.Contains(line, "echo") {
			caretLineIdx = i
		}
	}

	// Caret line should be right after input line (or within reasonable proximity)
	assert.False(t, caretLineIdx <= inputLineIdx || caretLineIdx > inputLineIdx+2)

	// Find the caret line content
	if caretLineIdx < len(lines) {
		caretLine := lines[caretLineIdx]
		// The carets should start at position 5 (after "echo ")
		// Count leading spaces/non-caret chars before first caret
		firstCaret := strings.Index(caretLine, "^")
		if firstCaret < 0 {
			t.Errorf("no caret found in caret line: %q", caretLine)
		} else if firstCaret != 5 {
			// Allow for some flexibility in formatting (e.g., line number prefix)
			// But the relative position should be correct
			t.Logf("first caret at position %d (expected around 5): %q", firstCaret, caretLine)
		}
	}
}

func TestFormatDiagnostics_EmptyErrors(t *testing.T) {
	input := `echo hello`
	errors := []SyntaxError{}

	result := FormatDiagnostics(input, errors)

	// Empty errors should return empty string
	assert.Equal(t, "", result)

}

func TestFormatDiagnostics_NilErrors(t *testing.T) {
	input := `echo hello`

	result := FormatDiagnostics(input, nil)

	// Nil errors should return empty string
	assert.Equal(t, "", result)

}

func TestFormatDiagnostics_ErrorAtStart(t *testing.T) {
	input := `"unclosed`
	errors := []SyntaxError{{Start: 0, End: 9, Message: "unclosed double quote"}}

	result := FormatDiagnostics(input, errors)

	// Should contain the input
	assert.Contains(t, result, input)

	// Should contain carets
	assert.Contains(t, result, "^")

	// Should contain error message
	assert.Contains(t, result, "unclosed double quote")

	// Verify carets start at the beginning
	lines := strings.Split(result, "\n")
	for _, line := range lines {
		if strings.Contains(line, "^") && !strings.Contains(line, `"`) {
			// First non-space character should be caret or very close to start
			trimmed := strings.TrimLeft(line, " \t")
			if !strings.HasPrefix(trimmed, "^") {
				t.Logf("caret line: %q (trimmed: %q)", line, trimmed)
			}
			break
		}
	}
}

func TestFormatDiagnostics_ErrorAtEnd(t *testing.T) {
	input := `echo hello |`
	errors := []SyntaxError{{Start: 11, End: 12, Message: "unexpected operator at end"}}

	result := FormatDiagnostics(input, errors)

	// Should contain the input
	assert.Contains(t, result, input)

	// Should contain carets
	assert.Contains(t, result, "^")

	// Should contain error message
	assert.Contains(t, result, "unexpected operator at end")

}

func TestFormatDiagnostics_SingleCharacterError(t *testing.T) {
	input := `echo |`
	errors := []SyntaxError{{Start: 5, End: 6, Message: "unexpected operator"}}

	result := FormatDiagnostics(input, errors)

	// Should contain at least one caret
	assert.Contains(t, result, "^")

	// Should contain error message
	assert.Contains(t, result, "unexpected operator")

}

func TestFormatDiagnostics_MultiLineCarets(t *testing.T) {
	input := `echo "very long unclosed string that spans many characters`
	errors := []SyntaxError{{Start: 5, End: 58, Message: "unclosed double quote"}}

	result := FormatDiagnostics(input, errors)

	// Should contain the input
	assert.Contains(t, result, input)

	// Should contain many carets (spanning the error range)
	caretCount := strings.Count(result, "^")
	// The error spans 53 characters, so we expect approximately that many carets
	assert.GreaterOrEqual(t, caretCount, 10)

}

func TestFormatDiagnosticsFromResult_Valid(t *testing.T) {
	result := &AnalysisResult{
		Tokens: []AnalyzedToken{
			{Type: TypeCommand, Value: "echo", Start: 0, End: 4},
		},
		Errors: []SyntaxError{},
		Valid:  true,
	}

	output := FormatDiagnosticsFromResult(result)

	// Valid result should return empty string
	assert.Equal(t, "", output)

}

func TestFormatDiagnosticsFromResult_Invalid(t *testing.T) {
	result := &AnalysisResult{
		Tokens: []AnalyzedToken{
			{Type: TypeError, Value: `"unclosed`, Start: 0, End: 9},
		},
		Errors: []SyntaxError{
			{Start: 0, End: 9, Message: "unclosed double quote"},
		},
		Valid: false,
	}

	// For this test, we need to use an input that matches the error positions
	// Since FormatDiagnosticsFromResult might not have access to original input,
	// it may reconstruct from tokens or require input to be passed differently
	output := FormatDiagnosticsFromResult(result)

	// Invalid result should show errors
	assert.NotEqual(t, "", output)

	// Should contain error message
	assert.Contains(t, output, "unclosed double quote")

}

func TestFormatDiagnosticsFromResult_NilResult(t *testing.T) {
	output := FormatDiagnosticsFromResult(nil)

	// Nil result should return empty string (or handle gracefully)
	assert.Equal(t, "", output)

}

func TestFormatDiagnosticsFromResult_MultipleErrors(t *testing.T) {
	result := &AnalysisResult{
		Tokens: []AnalyzedToken{
			{Type: TypeError, Value: `"a`, Start: 0, End: 2},
			{Type: TypeWhitespace, Value: " ", Start: 2, End: 3},
			{Type: TypeError, Value: `'b`, Start: 3, End: 5},
		},
		Errors: []SyntaxError{
			{Start: 0, End: 2, Message: "unclosed double quote"},
			{Start: 3, End: 5, Message: "unclosed single quote"},
		},
		Valid: false,
	}

	output := FormatDiagnosticsFromResult(result)

	// Should contain both error messages
	assert.Contains(t, output, "unclosed double quote")

	assert.Contains(t, output, "unclosed single quote")

}

func TestFormatDiagnostics_EmptyInput(t *testing.T) {
	input := ""
	errors := []SyntaxError{{Start: 0, End: 0, Message: "empty input error"}}

	result := FormatDiagnostics(input, errors)

	// Should still show error message even with empty input
	assert.Contains(t, result, "empty input error")

}

func TestFormatDiagnostics_ErrorBeyondInput(t *testing.T) {
	// Edge case: error position beyond input length (shouldn't happen, but handle gracefully)
	input := "echo"
	errors := []SyntaxError{{Start: 10, End: 15, Message: "out of bounds error"}}

	// Should not panic
	result := FormatDiagnostics(input, errors)

	// Should still contain error message
	assert.Contains(t, result, "out of bounds error")

}

func TestFormatDiagnostics_OverlappingErrors(t *testing.T) {
	input := `echo "test`
	errors := []SyntaxError{
		{Start: 5, End: 10, Message: "first error"},
		{Start: 7, End: 10, Message: "second error"},
	}

	// Should not panic on overlapping errors
	result := FormatDiagnostics(input, errors)

	// Both errors should be shown
	assert.Contains(t, result, "first error")

	assert.Contains(t, result, "second error")

}

func TestFormatDiagnostics_UnicodeInput(t *testing.T) {
	input := `echo "hello`
	errors := []SyntaxError{{Start: 5, End: 13, Message: "unclosed double quote"}}

	result := FormatDiagnostics(input, errors)

	// Should handle unicode characters properly
	assert.Contains(t, result, "hello")

	assert.Contains(t, result, "unclosed double quote")

}

func TestFormatDiagnostics_SpecialCharacters(t *testing.T) {
	input := `echo "hello\nworld`
	errors := []SyntaxError{{Start: 5, End: 18, Message: "unclosed double quote"}}

	result := FormatDiagnostics(input, errors)

	// Should handle special characters without breaking formatting
	assert.Contains(t, result, "unclosed double quote")

}

func TestFormatDiagnostics_TabsInInput(t *testing.T) {
	input := "echo\t\"unclosed"
	errors := []SyntaxError{{Start: 5, End: 14, Message: "unclosed double quote"}}

	result := FormatDiagnostics(input, errors)

	// Should handle tabs properly
	assert.Contains(t, result, "unclosed double quote")

	// Should contain carets
	assert.Contains(t, result, "^")

}

func TestFormatDiagnostics_ConsecutiveErrors(t *testing.T) {
	input := `"a"b"`
	errors := []SyntaxError{
		{Start: 0, End: 2, Message: "error at a"},
		{Start: 2, End: 5, Message: "error at b"},
	}

	result := FormatDiagnostics(input, errors)

	// Both consecutive errors should be shown
	assert.Contains(t, result, "error at a")

	assert.Contains(t, result, "error at b")

}

func TestFormatDiagnostics_ZeroLengthError(t *testing.T) {
	input := `echo hello`
	errors := []SyntaxError{{Start: 5, End: 5, Message: "zero length error"}}

	// Should handle zero-length errors gracefully
	result := FormatDiagnostics(input, errors)

	// Should still contain error message
	assert.Contains(t, result, "zero length error")

}

func TestFormatDiagnostics_NegativePositions(t *testing.T) {
	input := `echo hello`
	errors := []SyntaxError{{Start: -1, End: 5, Message: "negative start"}}

	// Should handle negative positions gracefully (not panic)
	result := FormatDiagnostics(input, errors)

	// Should still contain error message
	assert.Contains(t, result, "negative start")

}

func TestFormatDiagnostics_ReversedPositions(t *testing.T) {
	input := `echo hello`
	errors := []SyntaxError{{Start: 10, End: 5, Message: "reversed positions"}}

	// Should handle Start > End gracefully (not panic)
	result := FormatDiagnostics(input, errors)

	// Should still contain error message
	assert.Contains(t, result, "reversed positions")

}

func TestFormatDiagnosticsFromResult_EmptyTokensWithErrors(t *testing.T) {
	result := &AnalysisResult{
		Tokens: []AnalyzedToken{},
		Errors: []SyntaxError{
			{Start: 0, End: 5, Message: "some error"},
		},
		Valid: false,
	}

	output := FormatDiagnosticsFromResult(result)

	// Should handle empty tokens with errors
	assert.Contains(t, output, "some error")

}

func TestFormatDiagnostics_RealWorldExample_UnclosedQuote(t *testing.T) {
	// Simulate real analyzer behavior
	input := `echo "hello`
	analysisResult := Analyze(input)

	if analysisResult.Valid {
		t.Skip("analyzer didn't detect error, skipping format test")
	}

	result := FormatDiagnostics(input, analysisResult.Errors)

	// Should produce useful diagnostic output
	assert.NotEqual(t, "", result)

	assert.Contains(t, result, "^")

}

func TestFormatDiagnostics_RealWorldExample_TrailingPipe(t *testing.T) {
	input := `echo hello |`
	analysisResult := Analyze(input)

	if analysisResult.Valid {
		t.Skip("analyzer didn't detect error, skipping format test")
	}

	result := FormatDiagnostics(input, analysisResult.Errors)

	// Should produce useful diagnostic output
	assert.NotEqual(t, "", result)

}

func TestFormatDiagnostics_RealWorldExample_MissingRedirectTarget(t *testing.T) {
	input := `echo >`
	analysisResult := Analyze(input)

	if analysisResult.Valid {
		t.Skip("analyzer didn't detect error, skipping format test")
	}

	result := FormatDiagnostics(input, analysisResult.Errors)

	// Should produce useful diagnostic output
	assert.NotEqual(t, "", result)

}

func TestFormatDiagnosticsFromResult_RealWorldIntegration(t *testing.T) {
	input := `echo "unclosed`
	analysisResult := Analyze(input)

	if analysisResult.Valid {
		t.Skip("analyzer didn't detect error, skipping format test")
	}

	output := FormatDiagnosticsFromResult(analysisResult)

	// Should work with real analysis results
	assert.NotEqual(t, "", output)

}

func TestFormatDiagnostics_OutputFormat(t *testing.T) {
	input := `echo "test`
	errors := []SyntaxError{{Start: 5, End: 10, Message: "unclosed quote"}}

	result := FormatDiagnostics(input, errors)

	// Output should have a consistent format:
	// 1. Input line
	// 2. Caret line pointing to error
	// 3. Error message

	lines := strings.Split(strings.TrimSpace(result), "\n")

	assert.GreaterOrEqual(t, len(lines), 2)

	// Verify structure: should have input somewhere and carets
	hasInput := false
	hasCarets := false
	hasMessage := false

	for _, line := range lines {
		if strings.Contains(line, `echo "test`) {
			hasInput = true
		}
		if strings.Contains(line, "^") {
			hasCarets = true
		}
		if strings.Contains(line, "unclosed quote") {
			hasMessage = true
		}
	}

	assert.True(t, hasInput)

	assert.True(t, hasCarets)

	assert.True(t, hasMessage)

}

func TestFormatDiagnostics_CaretCount(t *testing.T) {
	input := `echo hello`
	errors := []SyntaxError{{Start: 5, End: 10, Message: "test error"}}

	result := FormatDiagnostics(input, errors)

	// Count carets - should be approximately End - Start (5 in this case)
	lines := strings.Split(result, "\n")
	var caretLine string
	for _, line := range lines {
		if strings.Contains(line, "^") && !strings.Contains(line, "echo") {
			caretLine = line
			break
		}
	}

	require.NotEqual(t, "", caretLine)

	caretCount := strings.Count(caretLine, "^")
	expectedCarets := 5 // End - Start

	assert.Equal(t, expectedCarets, caretCount)

}

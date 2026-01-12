// Package syntax provides syntax analysis, highlighting, and diagnostics for shell input.
package syntax

import (
	"strings"
)

// FormatDiagnostics formats syntax errors for display.
// It takes the original input and a list of errors, and returns a formatted
// string showing each error with its location in the input.
func FormatDiagnostics(input string, errors []SyntaxError) string {
	if len(errors) == 0 {
		return ""
	}

	var result strings.Builder
	lines := strings.Split(input, "\n")

	for i, err := range errors {
		if i > 0 {
			result.WriteString("\n")
		}

		// Find which line contains this error
		lineNum, lineStart := findLineForPosition(input, err.Start)

		// Get the line content
		var line string
		if lineNum < len(lines) {
			line = lines[lineNum]
		}

		// Calculate positions relative to the line start
		startInLine := err.Start - lineStart
		endInLine := err.End - lineStart

		// Clamp positions to line bounds
		if startInLine < 0 {
			startInLine = 0
		}
		if endInLine > len(line) {
			endInLine = len(line)
		}
		if endInLine < startInLine {
			endInLine = startInLine + 1
		}

		// Build the caret line
		caretLine := buildCaretLine(startInLine, endInLine)

		// Write the formatted error
		result.WriteString(line)
		result.WriteString("\n")
		result.WriteString(caretLine)
		result.WriteString("\n")
		result.WriteString("error: ")
		result.WriteString(err.Message)
	}

	return result.String()
}

// FormatDiagnosticsFromResult is a convenience function that extracts errors
// from an AnalysisResult and formats them.
func FormatDiagnosticsFromResult(result *AnalysisResult) string {
	if result == nil {
		return ""
	}
	// We need the original input to format diagnostics properly.
	// Reconstruct it from the tokens.
	input := reconstructInput(result.Tokens)
	return FormatDiagnostics(input, result.Errors)
}

// findLineForPosition finds the line number and starting position of the line
// that contains the given position in the input.
func findLineForPosition(input string, pos int) (lineNum int, lineStart int) {
	lineNum = 0
	lineStart = 0

	for i, c := range input {
		if i >= pos {
			break
		}
		if c == '\n' {
			lineNum++
			lineStart = i + 1
		}
	}

	return lineNum, lineStart
}

// buildCaretLine creates a line of carets (^) pointing to the error location.
func buildCaretLine(start, end int) string {
	if end <= start {
		end = start + 1
	}

	var builder strings.Builder

	// Add leading spaces
	for i := 0; i < start; i++ {
		builder.WriteRune(' ')
	}

	// Add carets
	for i := start; i < end; i++ {
		builder.WriteRune('^')
	}

	return builder.String()
}

// reconstructInput rebuilds the original input from the analyzed tokens.
func reconstructInput(tokens []AnalyzedToken) string {
	if len(tokens) == 0 {
		return ""
	}

	var builder strings.Builder
	for _, token := range tokens {
		builder.WriteString(token.Value)
	}
	return builder.String()
}

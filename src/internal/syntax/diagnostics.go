// Package syntax provides syntax analysis, highlighting, and diagnostics for shell input.
package syntax

import (
	"strings"
)

// FormatDiagnostics formats syntax errors for display.
//
// Positions in SyntaxError are RUNE indices (the analyzer scans []rune),
// so all offset math here is rune-based end-to-end: multi-byte characters
// before the error must not shift the reported line or caret column.
//
// Output format: one block per error, consisting of the offending line, a
// caret line, and "error: <message>". Blocks are separated by exactly one
// blank line, and the entire output ends with exactly one trailing
// newline.
func FormatDiagnostics(input string, errors []SyntaxError) string {
	if len(errors) == 0 {
		return ""
	}

	var result strings.Builder
	runes := []rune(input)
	lines := strings.Split(input, "\n")

	for i, err := range errors {
		if i > 0 {
			// Each block already ends with a newline; one more produces
			// the single blank line between blocks.
			result.WriteString("\n")
		}

		// Find which line contains this error (rune-based)
		lineNum, lineStart := findLineForPosition(runes, err.Start)

		// Get the line content
		var line string
		if lineNum < len(lines) {
			line = lines[lineNum]
		}
		lineLen := len([]rune(line))

		// Calculate positions relative to the line start (in runes)
		startInLine := err.Start - lineStart
		endInLine := err.End - lineStart

		// Clamp positions to line bounds
		if startInLine < 0 {
			startInLine = 0
		}
		if endInLine > lineLen {
			endInLine = lineLen
		}
		if endInLine < startInLine {
			endInLine = startInLine + 1
		}

		// Build the caret line
		caretLine := buildCaretLine(startInLine, endInLine)

		// Write the formatted error block
		result.WriteString(line)
		result.WriteString("\n")
		result.WriteString(caretLine)
		result.WriteString("\n")
		result.WriteString("error: ")
		result.WriteString(err.Message)
		result.WriteString("\n")
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

// findLineForPosition finds the line number and starting rune index of the
// line that contains the given rune position.
func findLineForPosition(input []rune, pos int) (lineNum int, lineStart int) {
	for i := 0; i < len(input) && i < pos; i++ {
		if input[i] == '\n' {
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

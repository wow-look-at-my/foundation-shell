// Package syntax provides syntax analysis, highlighting, and diagnostics for shell input.
package syntax

import (
	"strings"
)

// Theme maps SemanticType values to ANSI escape codes for syntax highlighting.
type Theme map[SemanticType]string

// ANSI escape code for resetting text formatting.
const ansiReset = "\033[0m"

// DefaultTheme provides a default color scheme for syntax highlighting.
var DefaultTheme = Theme{
	TypeCommand:            "\033[1;36m", // Bold cyan
	TypeArgument:           "\033[0m",    // Default
	TypeOperator:           "\033[1;33m", // Bold yellow
	TypeRedirection:        "\033[35m",   // Magenta
	TypeRedirectionTarget:  "\033[0m",    // Default
	TypeSingleQuotedString: "\033[32m",   // Green
	TypeDoubleQuotedString: "\033[33m",   // Yellow
	TypeBacktick:           "\033[36m",   // Cyan
	TypeSubshell:           "\033[36m",   // Cyan
	TypeVariable:           "\033[34m",   // Blue
	TypeParenGroup:         "\033[35m",   // Magenta
	TypeError:              "\033[4;31m", // Underline red
	TypeWhitespace:         "\033[0m",    // Reset
	TypeUnknown:            "\033[0m",    // Reset
}

// Highlighter applies syntax highlighting to shell input using a theme.
type Highlighter struct {
	theme Theme
}

// NewHighlighter creates a new Highlighter with the specified theme.
func NewHighlighter(theme Theme) *Highlighter {
	return &Highlighter{
		theme: theme,
	}
}

// Highlight applies syntax highlighting to the input string and returns
// the ANSI-colored result.
func (h *Highlighter) Highlight(input string) string {
	result := Analyze(input)
	return h.applyTheme(result.Tokens)
}

// HighlightResult applies syntax highlighting and returns both the colored
// string and any syntax errors found.
func (h *Highlighter) HighlightResult(input string) (string, []SyntaxError) {
	result := Analyze(input)
	colored := h.applyTheme(result.Tokens)
	return colored, result.Errors
}

// applyTheme builds the highlighted string by applying theme colors to each token.
func (h *Highlighter) applyTheme(tokens []AnalyzedToken) string {
	var builder strings.Builder

	for _, token := range tokens {
		// Get the color for this token type
		color, ok := h.theme[token.Type]
		if !ok {
			color = ansiReset
		}

		// Apply color, write token value, then reset
		builder.WriteString(color)
		builder.WriteString(token.Value)
		builder.WriteString(ansiReset)
	}

	return builder.String()
}

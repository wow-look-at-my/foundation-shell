// Package expander provides shell-style expansion for tilde and environment variables.
package expander

import (
	"os"
	"strings"
	"unicode"
)

// EscapeMarker is the marker used to indicate an escaped dollar sign.
// When a token contains \x01$, it should be converted to a literal $ without expansion.
const EscapeMarker = "\x01"

// ExpandTilde expands tilde at the beginning of a token.
// - "~" becomes $HOME
// - "~/path" becomes $HOME/path
// - "~user" is left unchanged (user expansion not supported)
func ExpandTilde(token string) string {
	if len(token) == 0 || token[0] != '~' {
		return token
	}

	home := os.Getenv("HOME")
	if home == "" {
		return token
	}

	// Just "~"
	if len(token) == 1 {
		return home
	}

	// "~/..." - expand to home + rest
	if token[1] == '/' {
		return home + token[1:]
	}

	// "~user" or "~something" - leave unchanged (no user expansion support)
	return token
}

// ExpandEnvironment expands environment variables in a token.
// Supports both $VAR and ${VAR} syntax.
// Escaped dollars (marked with \x01$) are converted to literal $ without expansion.
// Non-existent variables expand to empty string.
func ExpandEnvironment(token string) string {
	if !strings.Contains(token, "$") {
		return token
	}

	var result strings.Builder
	result.Grow(len(token))

	i := 0
	for i < len(token) {
		// Check for escape marker before $
		if i < len(token)-1 && token[i] == '\x01' && token[i+1] == '$' {
			// Escaped dollar - write literal $ and skip the marker
			result.WriteByte('$')
			i += 2
			continue
		}

		if token[i] != '$' {
			result.WriteByte(token[i])
			i++
			continue
		}

		// We have a $ - try to expand it
		if i+1 >= len(token) {
			// $ at end of string - keep it literal
			result.WriteByte('$')
			i++
			continue
		}

		next := token[i+1]

		// Handle ${VAR} syntax
		if next == '{' {
			closeIdx := strings.Index(token[i+2:], "}")
			if closeIdx == -1 {
				// No closing brace - treat as literal
				result.WriteByte('$')
				i++
				continue
			}

			varName := token[i+2 : i+2+closeIdx]
			if varName == "" {
				// ${} - empty var name, keep literal
				result.WriteString("${}")
				i += 3
				continue
			}

			varValue := os.Getenv(varName)
			result.WriteString(varValue)
			i += 3 + closeIdx // skip ${, var name, and }
			continue
		}

		// Handle $VAR syntax - variable name is alphanumeric + underscore
		if !isVarStartChar(next) {
			// $ followed by non-variable char (like $$, $!, etc.)
			result.WriteByte('$')
			i++
			continue
		}

		// Find end of variable name
		varStart := i + 1
		varEnd := varStart
		for varEnd < len(token) && isVarChar(token[varEnd]) {
			varEnd++
		}

		varName := token[varStart:varEnd]
		varValue := os.Getenv(varName)
		result.WriteString(varValue)
		i = varEnd
	}

	return result.String()
}

// isVarStartChar returns true if c can start a variable name (letter or underscore).
func isVarStartChar(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// isVarChar returns true if c can be part of a variable name (alphanumeric or underscore).
func isVarChar(c byte) bool {
	r := rune(c)
	return c == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

// Expand performs full expansion on a token.
// If wasSingleQuoted is true, no expansion is performed (single quotes preserve literal content).
// Otherwise, tilde expansion is applied first, then environment variable expansion.
func Expand(token string, wasSingleQuoted bool) string {
	if wasSingleQuoted {
		return token
	}

	// Apply tilde expansion first (only at start of token)
	result := ExpandTilde(token)

	// Then apply environment variable expansion
	result = ExpandEnvironment(result)

	return result
}

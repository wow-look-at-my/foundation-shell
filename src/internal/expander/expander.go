// Package expander provides shell-style expansion for tilde and environment variables.
package expander

import (
	"errors"
	"os"
	"strings"
	"unicode"
)

// SubshellExecutor is an interface for executing subshell commands.
type SubshellExecutor interface {
	Execute(command string) (output string, exitCode int, err error)
}

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

// ExpandCommandSubstitution expands $(...) and `...` command substitutions in a token.
// It processes innermost substitutions first to handle nesting.
// Trailing newlines are trimmed from command output (standard shell behavior).
func ExpandCommandSubstitution(token string, executor SubshellExecutor) (string, error) {
	if executor == nil {
		return "", errors.New("executor cannot be nil")
	}

	result := token

	// Keep expanding until no more substitutions are found.
	// This naturally handles nesting by processing innermost first.
	for {
		// Try to find an innermost $(...) or `...` substitution
		dollarStart, dollarEnd, dollarCmd := findInnermostDollarParen(result)
		backtickStart, backtickEnd, backtickCmd := findInnermostBacktick(result)

		// If no substitutions found, we're done
		if dollarStart == -1 && backtickStart == -1 {
			break
		}

		// Determine which substitution to process (prefer the one that appears first,
		// or if they're at the same position, prefer $(...) syntax)
		var start, end int
		var cmd string

		if dollarStart == -1 {
			start, end, cmd = backtickStart, backtickEnd, backtickCmd
		} else if backtickStart == -1 {
			start, end, cmd = dollarStart, dollarEnd, dollarCmd
		} else if dollarStart <= backtickStart {
			start, end, cmd = dollarStart, dollarEnd, dollarCmd
		} else {
			start, end, cmd = backtickStart, backtickEnd, backtickCmd
		}

		// Execute the command
		output, _, err := executor.Execute(cmd)
		if err != nil {
			return "", err
		}

		// Trim trailing newlines (standard shell behavior)
		output = strings.TrimRight(output, "\n")

		// Replace the substitution with the output
		result = result[:start] + output + result[end:]
	}

	return result, nil
}

// findInnermostDollarParen finds the innermost $(...) substitution.
// Returns the start index (at $), end index (after closing paren), and the command inside.
// Returns -1, -1, "" if no substitution is found.
func findInnermostDollarParen(s string) (start, end int, cmd string) {
	// Find all $( positions and track parenthesis depth to find innermost
	bestStart := -1
	bestEnd := -1
	bestCmd := ""

	i := 0
	for i < len(s)-1 {
		if s[i] == '$' && s[i+1] == '(' {
			// Found a $( - now find its matching )
			parenStart := i + 2
			matchEnd := findMatchingParen(s, parenStart)
			if matchEnd != -1 {
				innerCmd := s[parenStart:matchEnd]
				// Check if this command contains no further $( - making it innermost
				if !strings.Contains(innerCmd, "$(") {
					// This is an innermost substitution
					bestStart = i
					bestEnd = matchEnd + 1
					bestCmd = innerCmd
					break
				}
			}
			i++
		} else {
			i++
		}
	}

	return bestStart, bestEnd, bestCmd
}

// findMatchingParen finds the closing parenthesis matching an opening paren.
// startIdx should be the index right after the opening paren.
// Returns the index of the closing paren, or -1 if not found.
func findMatchingParen(s string, startIdx int) int {
	depth := 1
	for i := startIdx; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

// findInnermostBacktick finds the innermost `...` substitution.
// Returns the start index (at first backtick), end index (after closing backtick), and the command inside.
// Returns -1, -1, "" if no substitution is found.
func findInnermostBacktick(s string) (start, end int, cmd string) {
	// For backticks, we need to find pairs. Nested backticks are tricky in real shells,
	// but we'll implement a simple version: find the first backtick, then find its pair.
	// For nested backticks like `echo `date``, we process innermost first.

	backticks := []int{}
	for i := 0; i < len(s); i++ {
		// Skip escaped backticks: the lexer marks \` with EscapeMarker so
		// it is a literal character, not a substitution delimiter.
		if s[i] == '`' && (i == 0 || s[i-1] != EscapeMarker[0]) {
			backticks = append(backticks, i)
		}
	}

	// Need at least 2 backticks for a substitution
	if len(backticks) < 2 {
		return -1, -1, ""
	}

	// For innermost-first processing with backticks:
	// If we have `echo `date``, we want to find the innermost pair first.
	// We'll use a simple heuristic: find the shortest span between consecutive backticks
	// that doesn't contain other backticks.

	for i := 0; i < len(backticks)-1; i++ {
		startPos := backticks[i]
		endPos := backticks[i+1]
		innerCmd := s[startPos+1 : endPos]

		// Check if this span contains no backticks - making it innermost
		if !strings.Contains(innerCmd, "`") {
			return startPos, endPos + 1, innerCmd
		}
	}

	// Fallback: use first and second backtick
	return backticks[0], backticks[1] + 1, s[backticks[0]+1 : backticks[1]]
}

// Package expander provides shell-style expansion for tilde, environment
// variables, and command substitution.
package expander

import (
	"os"
	"strconv"
	"strings"

	"foundation-shell/internal/lexer"
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
// Supports both $VAR and ${VAR} syntax; $? expands to 0.
// Escaped dollars (marked with \x01$) are converted to literal $ without expansion.
// Non-existent variables expand to empty string.
func ExpandEnvironment(token string) string {
	return expandVariables(token, nil)
}

// expandVariables is ExpandEnvironment with an injectable exit-status
// source for $?: nil means 0.
func expandVariables(token string, lastStatus func() int) string {
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

		// $? expands to the exit status of the most recent command. Other
		// special parameters ($$, $!, $1, ...) stay literal.
		if next == '?' {
			status := 0
			if lastStatus != nil {
				status = lastStatus()
			}
			result.WriteString(strconv.Itoa(status))
			i += 2
			continue
		}

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

// isVarChar returns true if c can be part of a variable name: ASCII
// [A-Za-z0-9_] bytes only. Treating raw bytes with unicode.IsLetter would
// pull UTF-8 lead/continuation bytes (>= 0x80) into the name byte by byte,
// so a multibyte sequence like é must instead end the name.
func isVarChar(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
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

// Options configures ExpandToken.
type Options struct {
	// Executor runs command-substitution bodies. When nil, substitution
	// spans are left verbatim (no execution and no expansion of body text),
	// matching Parse without an executor.
	Executor SubshellExecutor
	// LastStatus reports the exit status of the most recent command for $?
	// expansion. When nil, $? expands to 0.
	LastStatus func() int
}

// ExpandToken expands a value token's content in a single left-to-right
// pass. The literal text between substitution spans is variable-expanded;
// each TOP-LEVEL command-substitution span ($(...) or `...`, located with
// the lexer's own quote-aware scan) is executed via opts.Executor and its
// output -- with trailing newlines trimmed -- is spliced in verbatim.
//
// Spliced output and substitution bodies are never re-scanned: a
// substitution appearing in a command's OUTPUT stays literal text, and
// nesting inside a body is handled recursively when the executor re-enters
// the parser with the body. Escape-marked $ and ` characters (from \$ and
// \`) never start a span and are left for StripEscapeMarkers.
func ExpandToken(content string, opts Options) (string, error) {
	spans, err := lexer.FindSubstitutionSpans(content)
	if err != nil {
		return "", err
	}
	if len(spans) == 0 {
		return expandVariables(content, opts.LastStatus), nil
	}

	runes := []rune(content)
	var result strings.Builder
	prev := 0
	for _, span := range spans {
		result.WriteString(expandVariables(string(runes[prev:span.Start]), opts.LastStatus))
		if opts.Executor == nil {
			// No executor: the span stays verbatim, body text untouched.
			result.WriteString(string(runes[span.Start:span.End]))
		} else {
			output, _, err := opts.Executor.Execute(span.Body)
			if err != nil {
				return "", err
			}
			// Trim trailing newlines (standard shell behavior). The output
			// is spliced verbatim and scanning continues AFTER it.
			result.WriteString(strings.TrimRight(output, "\n"))
		}
		prev = span.End
	}
	result.WriteString(expandVariables(string(runes[prev:]), opts.LastStatus))
	return result.String(), nil
}

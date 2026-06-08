// Package lexer provides tokenization for shell input.
package lexer

import (
	"errors"
	"strings"
	"unicode"
)

// EscapeMarker is a special character used to mark escaped $ signs
// that should not be expanded later.
const EscapeMarker = '\x01'

// TokenContext represents a token with metadata about how it was quoted.
type TokenContext struct {
	Content         string
	WasSingleQuoted bool
}

// Tokenize splits the input string into tokens, respecting quotes and escapes.
//
// Quoting rules:
//   - Single quotes: Everything inside is literal, no escapes processed
//   - Double quotes: Content stays together, escapes are processed
//   - Command substitution: $(...) and `...` keep content together
//   - Backslash escapes (outside single quotes):
//   - \\ becomes \
//   - \  (backslash space) becomes literal space (doesn't split)
//   - \$ becomes marker + $ (prevents expansion later)
//   - \" becomes " (inside double quotes)
//   - Other \X becomes X
func Tokenize(input string) ([]TokenContext, error) {
	var tokens []TokenContext
	var current strings.Builder
	wasSingleQuoted := false
	inSingleQuotes := false
	inDoubleQuotes := false
	dollarParenDepth := 0 // Track $(...) nesting
	backtickDepth := 0    // Track `...` nesting (0 = outside, 1 = inside)

	runes := []rune(input)
	for i := 0; i < len(runes); i++ {
		c := runes[i]

		// Handle backslash escapes (not inside single quotes)
		if c == '\\' && !inSingleQuotes && i+1 < len(runes) {
			next := runes[i+1]
			switch next {
			case '\\':
				// \\ becomes \
				current.WriteRune('\\')
			case '$':
				// \$ becomes marker + $ to prevent expansion
				current.WriteRune(EscapeMarker)
				current.WriteRune('$')
			case ' ':
				// Escaped space becomes literal space
				current.WriteRune(' ')
			case '"':
				// \" becomes literal "
				current.WriteRune('"')
			case '\'':
				// \' becomes literal '
				current.WriteRune('\'')
			case '`':
				// \` becomes literal `
				current.WriteRune('`')
			case 'n':
				// \n becomes newline
				current.WriteRune('\n')
			case 't':
				// \t becomes tab
				current.WriteRune('\t')
			default:
				// Other escapes: just add the escaped character
				current.WriteRune(next)
			}
			i++ // Skip the next character
			continue
		}

		// Handle $( for command substitution (not inside single quotes)
		if c == '$' && !inSingleQuotes && i+1 < len(runes) && runes[i+1] == '(' {
			dollarParenDepth++
			current.WriteRune(c)
			current.WriteRune('(')
			i++ // Skip the (
			continue
		}

		// Handle ) to close $(...) (not inside single quotes)
		if c == ')' && !inSingleQuotes && dollarParenDepth > 0 {
			dollarParenDepth--
			current.WriteRune(c)
			continue
		}

		// Handle backticks for command substitution (not inside single quotes)
		if c == '`' && !inSingleQuotes {
			if backtickDepth == 0 {
				backtickDepth = 1
			} else {
				backtickDepth = 0
			}
			current.WriteRune(c)
			continue
		}

		// Handle single quote toggle (not inside double quotes or command substitution)
		if c == '\'' && !inDoubleQuotes && dollarParenDepth == 0 && backtickDepth == 0 {
			if !inSingleQuotes {
				// Entering single quotes
				wasSingleQuoted = true
			}
			inSingleQuotes = !inSingleQuotes
			continue
		}

		// Handle double quote toggle (not inside single quotes)
		if c == '"' && !inSingleQuotes {
			inDoubleQuotes = !inDoubleQuotes
			continue
		}

		// Handle whitespace as token separator (not inside any quotes or command substitution)
		if unicode.IsSpace(c) && !inSingleQuotes && !inDoubleQuotes && dollarParenDepth == 0 && backtickDepth == 0 {
			if current.Len() > 0 {
				tokens = append(tokens, TokenContext{
					Content:         current.String(),
					WasSingleQuoted: wasSingleQuoted,
				})
				current.Reset()
				wasSingleQuoted = false
			}
			continue
		}

		// Regular character - add to current token
		current.WriteRune(c)
	}

	// Check for unclosed quotes
	if inSingleQuotes {
		return nil, errors.New("unclosed single quote")
	}
	if inDoubleQuotes {
		return nil, errors.New("unclosed double quote")
	}
	if dollarParenDepth > 0 {
		return nil, errors.New("unclosed $(")
	}
	if backtickDepth > 0 {
		return nil, errors.New("unclosed backtick")
	}

	// Add the last token if any
	if current.Len() > 0 {
		tokens = append(tokens, TokenContext{
			Content:         current.String(),
			WasSingleQuoted: wasSingleQuoted,
		})
	}

	return tokens, nil
}

// StripEscapeMarkers removes escape markers from a string, used after
// expansion has been performed.
func StripEscapeMarkers(s string) string {
	return strings.ReplaceAll(s, string(EscapeMarker), "")
}

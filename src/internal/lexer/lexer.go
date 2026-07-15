// Package lexer provides tokenization for shell input.
package lexer

import (
	"errors"
	"strings"
	"unicode"
)

// EscapeMarker is a special character used to mark escaped $ and `
// characters that must not be treated as expansion/substitution syntax by
// later stages. It is removed by StripEscapeMarkers after expansion.
const EscapeMarker = '\x01'

// TokenContext represents a token with metadata about how it was produced.
type TokenContext struct {
	Content string
	// WasSingleQuoted is true when part of the word was single-quoted.
	// It suppresses all expansion for the whole token.
	WasSingleQuoted bool
	// WasQuoted is true when any part of the word was inside any quotes
	// (single or double). Quote characters inside command-substitution
	// bodies do NOT set it: they belong to the body, which is re-parsed
	// when the substitution executes.
	WasQuoted bool
	// IsOperator is true only for operator tokens produced by the lexer's
	// operator recognition (|, ||, &&, ;, <, >, >>, 2>, 2>>).
	IsOperator bool
}

// subContext tracks one open command substitution ($(...) or `...`)
// together with the quote state inside its body.
type subContext struct {
	kind     rune // '(' for $(...), '`' for backticks
	inSingle bool
	inDouble bool
}

// Tokenize splits the input string into tokens, respecting quotes and escapes.
//
// Quoting rules:
//   - Single quotes: Everything inside is literal, no escapes processed
//   - Double quotes: Content stays together, escapes are processed
//   - Command substitution: $(...) and `...` keep content together. The
//     body is preserved VERBATIM (including quotes and backslashes): it is
//     re-parsed when the substitution executes. Quote state inside the body
//     is still tracked so a quoted ) or ` does not close the substitution,
//     and unquoted whitespace inside an open substitution does not split
//     tokens.
//   - Backslash escapes (outside single quotes):
//   - \\ becomes \
//   - \  (backslash space) becomes literal space (doesn't split)
//   - \$ becomes marker + $ (prevents expansion later)
//   - \` becomes marker + ` (prevents command substitution later)
//   - \" becomes " (inside double quotes)
//   - Other \X becomes X
//
// A word consisting only of quotes ('' or "") produces an empty token with
// the corresponding quoting flags set.
func Tokenize(input string) ([]TokenContext, error) {
	var tokens []TokenContext
	var current strings.Builder
	wasSingleQuoted := false
	wasQuoted := false
	inSingleQuotes := false
	inDoubleQuotes := false
	var subStack []subContext

	// flushWord ends the current word: it emits a token when the word has
	// content or consisted only of quotes (empty argument), and ALWAYS
	// resets the per-word flags. Resetting unconditionally matters: an
	// empty quoted word ('' / "") must not leak its flags into the next
	// word (e.g. `echo '' $HOME` must still expand $HOME).
	flushWord := func() {
		if current.Len() > 0 || wasQuoted {
			tokens = append(tokens, TokenContext{
				Content:         current.String(),
				WasSingleQuoted: wasSingleQuoted,
				WasQuoted:       wasQuoted,
			})
			current.Reset()
		}
		wasSingleQuoted = false
		wasQuoted = false
	}

	runes := []rune(input)
	for i := 0; i < len(runes); i++ {
		c := runes[i]

		inSubstitution := len(subStack) > 0
		var top *subContext
		if inSubstitution {
			top = &subStack[len(subStack)-1]
		}
		// Effective single-quote state: a substitution body has its own
		// quote context; outside substitutions the outer flag applies.
		effSingle := inSingleQuotes
		if inSubstitution {
			effSingle = top.inSingle
		}

		// Handle backslash escapes (not inside single quotes)
		if c == '\\' && !effSingle && i+1 < len(runes) {
			next := runes[i+1]
			if inSubstitution {
				// Substitution bodies are preserved verbatim: the inner
				// lexer processes the escape when the body is re-parsed.
				// Consuming both characters here still prevents an escaped
				// ) ` " ' from affecting this scan's depth/quote state.
				current.WriteRune('\\')
				current.WriteRune(next)
			} else {
				switch next {
				case '\\':
					// \\ becomes \
					current.WriteRune('\\')
				case '$':
					// \$ becomes marker + $ to prevent expansion
					current.WriteRune(EscapeMarker)
					current.WriteRune('$')
				case '`':
					// \` becomes marker + ` so the expander treats it as a
					// literal backtick, not a substitution delimiter
					current.WriteRune(EscapeMarker)
					current.WriteRune('`')
				case ' ':
					// Escaped space becomes literal space
					current.WriteRune(' ')
				case '"':
					// \" becomes literal "
					current.WriteRune('"')
				case '\'':
					// \' becomes literal '
					current.WriteRune('\'')
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
			}
			i++ // Skip the escaped character
			continue
		}

		// Handle $( — opens a command substitution. Active inside double
		// quotes; suppressed in any single-quote context.
		if c == '$' && !effSingle && i+1 < len(runes) && runes[i+1] == '(' {
			subStack = append(subStack, subContext{kind: '('})
			current.WriteRune('$')
			current.WriteRune('(')
			i++ // Skip the (
			continue
		}

		// Handle ) — closes the innermost $(...) unless it is quoted
		// inside the substitution body.
		if c == ')' && inSubstitution && top.kind == '(' && !top.inSingle && !top.inDouble {
			subStack = subStack[:len(subStack)-1]
			current.WriteRune(c)
			continue
		}

		// Handle backticks. Suppressed in single quotes and when
		// double-quoted inside a substitution body; active inside outer
		// double quotes.
		if c == '`' && !effSingle && !(inSubstitution && top.inDouble) {
			if inSubstitution && top.kind == '`' {
				// Closing backtick of the innermost substitution
				subStack = subStack[:len(subStack)-1]
			} else {
				// Opening backtick (possibly nested inside $(...))
				subStack = append(subStack, subContext{kind: '`'})
			}
			current.WriteRune(c)
			continue
		}

		// Handle single quotes
		if c == '\'' {
			if inSubstitution {
				// Body text, preserved verbatim; tracked so a quoted )
				// or ` does not close the substitution. Does not set the
				// token's quoting flags.
				if !top.inDouble {
					top.inSingle = !top.inSingle
				}
				current.WriteRune(c)
				continue
			}
			if !inDoubleQuotes {
				if !inSingleQuotes {
					// Entering single quotes
					wasSingleQuoted = true
				}
				wasQuoted = true
				inSingleQuotes = !inSingleQuotes
				continue
			}
			// Inside double quotes: literal, falls through
		}

		// Handle double quotes
		if c == '"' {
			if inSubstitution {
				// Body text, preserved verbatim (see single quotes above)
				if !top.inSingle {
					top.inDouble = !top.inDouble
				}
				current.WriteRune(c)
				continue
			}
			if !inSingleQuotes {
				wasQuoted = true
				inDoubleQuotes = !inDoubleQuotes
				continue
			}
			// Inside single quotes: literal, falls through
		}

		// Handle whitespace as token separator (not inside any quotes or
		// command substitution)
		if unicode.IsSpace(c) && !inSingleQuotes && !inDoubleQuotes && !inSubstitution {
			flushWord()
			continue
		}

		// Regular character - add to current token
		current.WriteRune(c)
	}

	// Check for unclosed quotes and substitutions
	if inSingleQuotes {
		return nil, errors.New("unclosed single quote")
	}
	if inDoubleQuotes {
		return nil, errors.New("unclosed double quote")
	}
	for _, sc := range subStack {
		if sc.kind == '(' {
			return nil, errors.New("unclosed command substitution $(...)")
		}
	}
	if len(subStack) > 0 {
		return nil, errors.New("unclosed backtick")
	}

	// Add the last token if any
	flushWord()

	return tokens, nil
}

// StripEscapeMarkers removes escape markers from a string, used after
// expansion has been performed. It applies to every marked character
// (\$ and \` both use the marker).
func StripEscapeMarkers(s string) string {
	return strings.ReplaceAll(s, string(EscapeMarker), "")
}

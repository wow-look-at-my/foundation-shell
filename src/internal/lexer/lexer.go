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
	kind rune // '(' for $(...), '`' for backticks
	// depth is the same-type nesting level of a backtick region (see
	// QuoteNestsDeeper); it is always 1 for $(...) regions, which nest by
	// pushing a new context instead (their delimiters are asymmetric).
	depth int
	// single/double are the nesting depths of quote regions open inside
	// the substitution body (used only to guard the closing delimiter;
	// body text is preserved verbatim).
	single int
	double int
}

// isChainOperatorToken reports whether a token is one of the CHAIN
// operators (|, &&, ||, ;). Only chain operators suppress the newline
// separator (line continuation, lexer.md §3.4); redirection operators do
// not — a dangling redirection must not silently take the next line's
// first word as its target.
func isChainOperatorToken(t TokenContext) bool {
	if !t.IsOperator {
		return false
	}
	switch t.Content {
	case "|", "&&", "||", ";":
		return true
	}
	return false
}

// QuoteNestsDeeper implements the depth-tracked quote nesting rule shared
// by the lexer and the syntax analyzer. It reports whether an unescaped
// quote character at runes[i], read while a region of the SAME type is
// already open, opens a nested level instead of closing one. It nests one
// level deeper iff the previous input character is whitespace AND the next
// input character exists, is not whitespace, and is not a quote character
// (', ", or `). In every other case the character closes one level.
func QuoteNestsDeeper(runes []rune, i int) bool {
	if i == 0 || !unicode.IsSpace(runes[i-1]) {
		return false
	}
	if i+1 >= len(runes) {
		return false
	}
	next := runes[i+1]
	return !unicode.IsSpace(next) && next != '\'' && next != '"' && next != '`'
}

// Tokenize splits the input string into tokens, respecting quotes and escapes.
//
// Quoting rules:
//   - Single quotes: Everything inside is literal, no escapes processed
//   - Double quotes: Content stays together, escapes are processed
//   - Depth-tracked nesting: at depth 0 an unescaped quote character opens
//     its region. Inside an open region an unescaped SAME-type quote
//     character nests one level deeper iff the previous character is
//     whitespace and the next character exists, is not whitespace, and is
//     not a quote character (QuoteNestsDeeper); otherwise it closes one
//     level, and the region ends at depth 0. Nested quote characters stay
//     LITERAL in the token content: only the outermost pair is stripped.
//     So 'a 'b' c' is ONE token with content `a 'b' c`, while POSIX-shaped
//     input ('a' 'b', 'a'b, 'don'\''t', '', 'a''b') behaves exactly as in
//     POSIX because an adjacent or word-final quote never satisfies the
//     nesting test. Different-type quote characters inside a region are
//     literal content, as before.
//   - Command substitution: $(...) and `...` keep content together. The
//     body is preserved VERBATIM (including quotes and backslashes): it is
//     re-parsed when the substitution executes. Quote state inside the body
//     is still tracked (with the same nesting rule) so a quoted ) or ` does
//     not close the substitution, and unquoted whitespace inside an open
//     substitution does not split tokens. Backtick regions nest the same
//     way quotes do: `outer `inner` end` is one substitution whose body
//     keeps the inner backticks literally (the body re-parses at execution
//     time).
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
//
// Operators do not need surrounding whitespace: an unquoted, unescaped
// operator character outside any substitution ends the current word and
// produces an operator token (IsOperator=true) by maximal munch over
// {||, &&, 2>>, 2>, >>, |, ;, <, >}. 2>/2>> only apply when the pending
// word is exactly an unquoted "2" (which is consumed into the operator);
// a single & is a literal word character.
//
// An unquoted # at word start begins a comment running to the next
// unquoted newline or end of input. An unquoted newline outside
// substitutions is a SOFT command separator (lexer.md §3.4): it sets a
// pending flag, and a ; operator token materializes just before the next
// emitted token — unless no token has been emitted yet or the most recently
// emitted token is a CHAIN operator (|, &&, ||, ;), which makes the newline
// a line continuation. Redirection operators (<, >, >>, 2>, 2>>) do NOT
// suppress the separator: a redirection cannot be continued across a
// newline, so `cmd ><newline>file` lexes as cmd > ; file and the parser
// rejects it.
// Leading blank lines and a trailing newline produce nothing.
//
// An input left unclosed at end of input reports the INNERMOST unclosed
// construct: "unclosed single quote", "unclosed double quote", "unclosed
// backtick", or "unclosed command substitution $(...)". Note that with
// nesting an EVEN number of quote characters can be unclosed ('a 'b).
func Tokenize(input string) ([]TokenContext, error) {
	var tokens []TokenContext
	var current strings.Builder
	wasSingleQuoted := false
	wasQuoted := false
	wordHadEscape := false
	// Nesting depths of the outer quote regions. At most one is nonzero:
	// inside one type, the other type's characters are literal content.
	singleDepth := 0
	doubleDepth := 0
	var subStack []subContext
	// pendingNewline records a depth-0 newline; the separator it stands for
	// materializes just before the NEXT emitted token (lexer.md §3.4). A
	// flag still set at end of input is discarded (trailing newline).
	pendingNewline := false

	// emit appends a token, first materializing a pending newline separator
	// unless nothing has been emitted yet or the previous token is a chain
	// operator (line continuation). Redirection operators do NOT suppress
	// the separator.
	emit := func(tok TokenContext) {
		if pendingNewline {
			pendingNewline = false
			if len(tokens) > 0 && !isChainOperatorToken(tokens[len(tokens)-1]) {
				tokens = append(tokens, TokenContext{Content: ";", IsOperator: true})
			}
		}
		tokens = append(tokens, tok)
	}

	// flushWord ends the current word: it emits a token when the word has
	// content or consisted only of quotes (empty argument), and ALWAYS
	// resets the per-word flags. Resetting unconditionally matters: an
	// empty quoted word ('' / "") must not leak its flags into the next
	// word (e.g. `echo '' $HOME` must still expand $HOME).
	flushWord := func() {
		if current.Len() > 0 || wasQuoted {
			emit(TokenContext{
				Content:         current.String(),
				WasSingleQuoted: wasSingleQuoted,
				WasQuoted:       wasQuoted,
			})
			current.Reset()
		}
		wasSingleQuoted = false
		wasQuoted = false
		wordHadEscape = false
	}

	emitOperator := func(op string) {
		emit(TokenContext{Content: op, IsOperator: true})
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
		// quote context; outside substitutions the outer depth applies.
		effSingle := singleDepth > 0
		if inSubstitution {
			effSingle = top.single > 0
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
				wordHadEscape = true
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
		// quotes; suppressed in any single-quote context. $(...) regions
		// nest by pushing a new context: their delimiters are asymmetric,
		// so no same-type nesting rule is needed.
		if c == '$' && !effSingle && i+1 < len(runes) && runes[i+1] == '(' {
			subStack = append(subStack, subContext{kind: '(', depth: 1})
			current.WriteRune('$')
			current.WriteRune('(')
			i++ // Skip the (
			continue
		}

		// Handle ) — closes the innermost $(...) unless it is quoted
		// inside the substitution body.
		if c == ')' && inSubstitution && top.kind == '(' && top.single == 0 && top.double == 0 {
			subStack = subStack[:len(subStack)-1]
			current.WriteRune(c)
			continue
		}

		// Handle backticks. Suppressed in single quotes and when
		// double-quoted inside a substitution body; active inside outer
		// double quotes. Inside a backtick region a backtick either NESTS
		// (QuoteNestsDeeper) or closes one level; nested backticks stay in
		// the body verbatim, so `outer `inner` end` is one substitution.
		if c == '`' && !effSingle && !(inSubstitution && top.double > 0) {
			switch {
			case inSubstitution && top.kind == '`' && QuoteNestsDeeper(runes, i):
				top.depth++
			case inSubstitution && top.kind == '`':
				top.depth--
				if top.depth == 0 {
					subStack = subStack[:len(subStack)-1]
				}
			default:
				// Opening backtick (possibly nested inside $(...))
				subStack = append(subStack, subContext{kind: '`', depth: 1})
			}
			current.WriteRune(c)
			continue
		}

		// Handle single quotes
		if c == '\'' {
			if inSubstitution {
				// Body text, preserved verbatim; depth-tracked so a
				// quoted ) or ` does not close the substitution. Does not
				// set the token's quoting flags.
				if top.double == 0 {
					switch {
					case top.single == 0:
						top.single = 1
					case QuoteNestsDeeper(runes, i):
						top.single++
					default:
						top.single--
					}
				}
				current.WriteRune(c)
				continue
			}
			if doubleDepth == 0 {
				switch {
				case singleDepth == 0:
					// Opening quote of a region: stripped from content
					singleDepth = 1
					wasSingleQuoted = true
					wasQuoted = true
				case QuoteNestsDeeper(runes, i):
					// Nested opener: stays literal in the content
					singleDepth++
					current.WriteRune(c)
				default:
					singleDepth--
					if singleDepth > 0 {
						// Nested closer: stays literal in the content
						current.WriteRune(c)
					}
					// Outermost closer (depth 0): stripped
				}
				continue
			}
			// Inside double quotes: literal, falls through
		}

		// Handle double quotes
		if c == '"' {
			if inSubstitution {
				// Body text, preserved verbatim (see single quotes above)
				if top.single == 0 {
					switch {
					case top.double == 0:
						top.double = 1
					case QuoteNestsDeeper(runes, i):
						top.double++
					default:
						top.double--
					}
				}
				current.WriteRune(c)
				continue
			}
			if singleDepth == 0 {
				switch {
				case doubleDepth == 0:
					// Opening quote of a region: stripped from content
					doubleDepth = 1
					wasQuoted = true
				case QuoteNestsDeeper(runes, i):
					// Nested opener: stays literal in the content
					doubleDepth++
					current.WriteRune(c)
				default:
					doubleDepth--
					if doubleDepth > 0 {
						// Nested closer: stays literal in the content
						current.WriteRune(c)
					}
					// Outermost closer (depth 0): stripped
				}
				continue
			}
			// Inside single quotes: literal, falls through
		}

		// Handle comments: an unquoted # at word start (start of input or
		// after unquoted whitespace/operator), outside any substitution,
		// starts a comment running to the next unquoted newline or end of
		// input. foo#bar stays one word. A # inside a substitution body is
		// passed through as body text (known limitation: comment rules
		// apply only when the body is re-parsed at execution time).
		if c == '#' && singleDepth == 0 && doubleDepth == 0 && !inSubstitution &&
			current.Len() == 0 && !wasQuoted {
			for i+1 < len(runes) && runes[i+1] != '\n' {
				i++
			}
			// The terminating newline (if any) is handled on the next
			// iteration by the newline rule below.
			continue
		}

		// Handle newline as a SOFT command separator: an unquoted newline
		// outside substitutions flushes the word and sets the pending flag;
		// emit materializes the ; before the next token unless the previous
		// token is a chain operator (continuation). A run of newlines —
		// with or without comments between — collapses into one separator,
		// and a trailing newline produces nothing. Inside quotes a newline
		// is a literal word character; inside substitution bodies it is
		// body text.
		if c == '\n' && singleDepth == 0 && doubleDepth == 0 && !inSubstitution {
			flushWord()
			pendingNewline = true
			continue
		}

		// Handle operators without surrounding whitespace. An unquoted,
		// unescaped operator character outside any substitution ends the
		// current word and lexes an operator token by maximal munch over
		// {||, &&, 2>>, 2>, >>, |, ;, <, >}. Quoted operator characters
		// never get here (the quote handlers above consume them into word
		// content); escaped ones are consumed by the escape handler.
		if singleDepth == 0 && doubleDepth == 0 && !inSubstitution {
			switch c {
			case '&':
				// Single & is NOT an operator: a&b stays one word. Only
				// && is recognized.
				if i+1 < len(runes) && runes[i+1] == '&' {
					flushWord()
					emitOperator("&&")
					i++
					continue
				}
			case '|':
				flushWord()
				if i+1 < len(runes) && runes[i+1] == '|' {
					emitOperator("||")
					i++
				} else {
					emitOperator("|")
				}
				continue
			case ';':
				flushWord()
				emitOperator(";")
				continue
			case '<':
				flushWord()
				emitOperator("<")
				continue
			case '>':
				op := ">"
				if i+1 < len(runes) && runes[i+1] == '>' {
					op = ">>"
					i++
				}
				// 2> / 2>> apply only when the pending word is exactly an
				// unquoted, unescaped "2": that 2 is consumed into the
				// operator (echo a2>f keeps word a2 with operator >).
				if current.Len() == 1 && current.String() == "2" && !wasQuoted && !wordHadEscape {
					current.Reset()
					op = "2" + op
				} else {
					flushWord()
				}
				emitOperator(op)
				continue
			}
		}

		// Handle whitespace as token separator (not inside any quotes or
		// command substitution)
		if unicode.IsSpace(c) && singleDepth == 0 && doubleDepth == 0 && !inSubstitution {
			flushWord()
			continue
		}

		// Regular character - add to current token
		current.WriteRune(c)
	}

	// Check for unclosed quotes and substitutions, reporting the INNERMOST
	// unclosed construct: an open substitution (or a quote open inside its
	// body) is always inner to any outer quote region, because quotes
	// inside a body only touch the body's own context.
	if len(subStack) > 0 {
		top := subStack[len(subStack)-1]
		switch {
		case top.single > 0:
			return nil, errors.New("unclosed single quote")
		case top.double > 0:
			return nil, errors.New("unclosed double quote")
		case top.kind == '(':
			return nil, errors.New("unclosed command substitution $(...)")
		default:
			return nil, errors.New("unclosed backtick")
		}
	}
	if singleDepth > 0 {
		return nil, errors.New("unclosed single quote")
	}
	if doubleDepth > 0 {
		return nil, errors.New("unclosed double quote")
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

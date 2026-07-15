// Package syntax provides syntax analysis, highlighting, and diagnostics for shell input.
package syntax

import (
	"strings"
	"unicode"
)

// SemanticType represents the semantic meaning of a token for highlighting purposes.
type SemanticType int

const (
	// TypeUnknown is the default/unclassified type.
	TypeUnknown SemanticType = iota
	// TypeCommand is the first word in a command.
	TypeCommand
	// TypeArgument is a command argument.
	TypeArgument
	// TypeOperator is a control operator (|, &&, ||, ;).
	TypeOperator
	// TypeRedirection is a redirection operator (>, <, >>, 2>, 2>>).
	TypeRedirection
	// TypeRedirectionTarget is the target file for a redirection.
	TypeRedirectionTarget
	// TypeSingleQuotedString is content in single quotes.
	TypeSingleQuotedString
	// TypeDoubleQuotedString is content in double quotes.
	TypeDoubleQuotedString
	// TypeBacktick is a backtick command substitution.
	TypeBacktick
	// TypeSubshell is a $(...) command substitution.
	TypeSubshell
	// TypeVariable is a variable reference ($VAR or ${VAR}).
	TypeVariable
	// TypeParenGroup is a parenthesized group (...).
	TypeParenGroup
	// TypeError marks invalid/error regions.
	TypeError
	// TypeWhitespace is whitespace between tokens.
	TypeWhitespace
	// TypeComment is a comment (# to end of line).
	TypeComment
)

// String returns the string name of the semantic type.
func (t SemanticType) String() string {
	switch t {
	case TypeUnknown:
		return "Unknown"
	case TypeCommand:
		return "Command"
	case TypeArgument:
		return "Argument"
	case TypeOperator:
		return "Operator"
	case TypeRedirection:
		return "Redirection"
	case TypeRedirectionTarget:
		return "RedirectionTarget"
	case TypeSingleQuotedString:
		return "SingleQuotedString"
	case TypeDoubleQuotedString:
		return "DoubleQuotedString"
	case TypeBacktick:
		return "Backtick"
	case TypeSubshell:
		return "Subshell"
	case TypeVariable:
		return "Variable"
	case TypeParenGroup:
		return "ParenGroup"
	case TypeError:
		return "Error"
	case TypeWhitespace:
		return "Whitespace"
	case TypeComment:
		return "Comment"
	default:
		return "Unknown"
	}
}

// AnalyzedToken represents a token with its semantic type and position.
type AnalyzedToken struct {
	Type  SemanticType
	Value string
	Start int // Character position (0-indexed)
	End   int // Character position (exclusive)
	Depth int // Nesting depth for quotes/parens/subshells
}

// SyntaxError represents a syntax error with its position.
type SyntaxError struct {
	Start   int
	End     int
	Message string
}

// AnalysisResult contains the result of syntax analysis.
type AnalysisResult struct {
	Tokens []AnalyzedToken
	Errors []SyntaxError
	Valid  bool
}

// Analyze performs syntax analysis on the input string.
// It returns tokens with semantic types, positions, and any syntax errors.
// This is the single source of truth for syntax validation.
func Analyze(input string) *AnalysisResult {
	a := &analyzer{
		input:  []rune(input),
		pos:    0,
		tokens: make([]AnalyzedToken, 0),
		errors: make([]SyntaxError, 0),
	}
	a.analyze()
	return &AnalysisResult{
		Tokens: a.tokens,
		Errors: a.errors,
		Valid:  len(a.errors) == 0,
	}
}

// analyzer holds the state for syntax analysis.
type analyzer struct {
	input  []rune
	pos    int
	tokens []AnalyzedToken
	errors []SyntaxError

	// Track if we're at the start of a command (for Command vs Argument)
	isFirstInCommand bool
	// Track if we just saw a redirection operator
	afterRedirection bool
}

func (a *analyzer) analyze() {
	a.isFirstInCommand = true
	a.afterRedirection = false

	for a.pos < len(a.input) {
		// Skip and record whitespace
		if unicode.IsSpace(a.input[a.pos]) {
			a.consumeWhitespace()
			continue
		}

		// A # at word start (here: any fresh position after whitespace,
		// an operator, or start of input) begins a comment running to the
		// next newline or end of input, matching the lexer. A # inside a
		// word or a substitution body stays literal (parseWord consumes
		// it).
		if a.input[a.pos] == '#' {
			a.consumeComment()
			continue
		}

		// Check for operators first
		if op, length := a.matchOperator(true); length > 0 {
			a.addToken(op, a.pos, a.pos+length, 0)
			a.pos += length

			// Update state based on operator type
			if op == TypeOperator {
				a.isFirstInCommand = true
				a.afterRedirection = false
			} else if op == TypeRedirection {
				a.afterRedirection = true
			}
			continue
		}

		// Parse a word/token
		a.parseWord()
	}

	// Check for trailing operator errors
	if len(a.tokens) > 0 {
		last := a.tokens[len(a.tokens)-1]
		if last.Type == TypeOperator && last.Value != ";" {
			a.errors = append(a.errors, SyntaxError{
				Start:   last.Start,
				End:     last.End,
				Message: "unexpected operator at end",
			})
		}
		if last.Type == TypeRedirection {
			a.errors = append(a.errors, SyntaxError{
				Start:   last.Start,
				End:     last.End,
				Message: "missing redirection target",
			})
		}
	}
}

func (a *analyzer) consumeWhitespace() {
	start := a.pos
	sawNewline := false
	for a.pos < len(a.input) && unicode.IsSpace(a.input[a.pos]) {
		if a.input[a.pos] == '\n' {
			sawNewline = true
		}
		a.pos++
	}
	a.addToken(TypeWhitespace, start, a.pos, 0)

	// A newline separates commands (the lexer emits an implicit ;), so
	// the next word starts a new command.
	if sawNewline {
		a.isFirstInCommand = true
		a.afterRedirection = false
	}
}

// consumeComment consumes a comment from # to the next newline (exclusive)
// or end of input.
func (a *analyzer) consumeComment() {
	start := a.pos
	for a.pos < len(a.input) && a.input[a.pos] != '\n' {
		a.pos++
	}
	a.addToken(TypeComment, start, a.pos, 0)
}

// peek returns the rune at offset from the current position, or 0 when out
// of bounds.
func (a *analyzer) peek(offset int) rune {
	if a.pos+offset < len(a.input) {
		return a.input[a.pos+offset]
	}
	return 0
}

// matchOperator checks if the current position starts with an operator.
// Returns the semantic type and length of the operator, or 0 length if not
// an operator. It compares runes in place: no per-call allocation (this
// runs for every character on every keystroke).
//
// fresh indicates a word-start position (start of input, after whitespace
// or after an operator). The 2>/2>> forms only match there: mid-word the 2
// belongs to the pending word (echo a2>f is word a2 + operator >),
// matching the lexer's rule that 2>/2>> apply only when the pending word
// is exactly "2".
func (a *analyzer) matchOperator(fresh bool) (SemanticType, int) {
	if a.pos >= len(a.input) {
		return TypeUnknown, 0
	}

	switch a.input[a.pos] {
	case '&':
		if a.peek(1) == '&' {
			return TypeOperator, 2
		}
		// A single & is a literal word character, not an operator.
	case '|':
		if a.peek(1) == '|' {
			return TypeOperator, 2
		}
		return TypeOperator, 1
	case ';':
		return TypeOperator, 1
	case '>':
		if a.peek(1) == '>' {
			return TypeRedirection, 2
		}
		return TypeRedirection, 1
	case '<':
		return TypeRedirection, 1
	case '2':
		if fresh && a.peek(1) == '>' {
			if a.peek(2) == '>' {
				return TypeRedirection, 3
			}
			return TypeRedirection, 2
		}
	case '(':
		return TypeParenGroup, 1
	case ')':
		return TypeParenGroup, 1
	}

	return TypeUnknown, 0
}

// parseWord parses a word token, handling quotes, backticks, and variables.
func (a *analyzer) parseWord() {
	start := a.pos
	var builder strings.Builder

	// Track quote depths
	singleQuoteDepth := 0
	doubleQuoteDepth := 0
	backtickDepth := 0
	parenDepth := 0 // For $()

	// Track where quotes started for error reporting
	var quoteStarts []int

	for a.pos < len(a.input) {
		c := a.input[a.pos]

		// Handle escape sequences (outside single quotes)
		if c == '\\' && singleQuoteDepth == 0 && a.pos+1 < len(a.input) {
			next := a.input[a.pos+1]
			// Escaped characters don't count toward quote depth
			builder.WriteRune(c)
			builder.WriteRune(next)
			a.pos += 2
			continue
		}

		// Handle single quotes with depth tracking
		if c == '\'' && doubleQuoteDepth == 0 && backtickDepth == 0 {
			if singleQuoteDepth%2 == 0 {
				// Opening quote
				quoteStarts = append(quoteStarts, a.pos)
			} else {
				// Closing quote
				if len(quoteStarts) > 0 {
					quoteStarts = quoteStarts[:len(quoteStarts)-1]
				}
			}
			singleQuoteDepth++
			builder.WriteRune(c)
			a.pos++
			continue
		}

		// Handle double quotes with depth tracking
		if c == '"' && singleQuoteDepth == 0 && backtickDepth == 0 {
			if doubleQuoteDepth%2 == 0 {
				// Opening quote
				quoteStarts = append(quoteStarts, a.pos)
			} else {
				// Closing quote
				if len(quoteStarts) > 0 {
					quoteStarts = quoteStarts[:len(quoteStarts)-1]
				}
			}
			doubleQuoteDepth++
			builder.WriteRune(c)
			a.pos++
			continue
		}

		// Handle backticks with depth tracking. Parity check: a closed
		// single-quote pair earlier in the word must not disable backtick
		// tracking ('a'`date` still contains a substitution).
		if c == '`' && singleQuoteDepth%2 == 0 {
			if backtickDepth%2 == 0 {
				// Opening backtick
				quoteStarts = append(quoteStarts, a.pos)
			} else {
				// Closing backtick
				if len(quoteStarts) > 0 {
					quoteStarts = quoteStarts[:len(quoteStarts)-1]
				}
			}
			backtickDepth++
			builder.WriteRune(c)
			a.pos++
			continue
		}

		// Handle $() command substitution. Parity check: a closed
		// single-quote pair earlier in the word must not disable
		// substitution tracking ('a'$(date) is one word containing a
		// substitution, not a shattered token stream).
		if c == '$' && singleQuoteDepth%2 == 0 && a.pos+1 < len(a.input) && a.input[a.pos+1] == '(' {
			parenDepth++
			builder.WriteRune(c)
			builder.WriteRune('(')
			a.pos += 2
			continue
		}

		// Handle closing ) for $(). A ) that is quoted inside the
		// substitution body (odd quote parity) is body text and must not
		// close the substitution: echo $(echo ')') is valid.
		if c == ')' && parenDepth > 0 && singleQuoteDepth%2 == 0 && doubleQuoteDepth%2 == 0 {
			parenDepth--
			builder.WriteRune(c)
			a.pos++
			continue
		}

		// Check for word boundaries (whitespace or operators)
		// We're outside quotes when the count is even (0, 2, 4, ...)
		outsideQuotes := singleQuoteDepth%2 == 0 && doubleQuoteDepth%2 == 0 && backtickDepth%2 == 0 && parenDepth == 0
		if outsideQuotes {
			if unicode.IsSpace(c) {
				break
			}
			if _, length := a.matchOperator(false); length > 0 {
				break
			}
		}

		builder.WriteRune(c)
		a.pos++
	}

	value := builder.String()
	if len(value) == 0 {
		return
	}

	// Check for unclosed quotes (odd count)
	if singleQuoteDepth%2 != 0 {
		a.errors = append(a.errors, SyntaxError{
			Start:   start,
			End:     a.pos,
			Message: "unclosed single quote",
		})
	}
	if doubleQuoteDepth%2 != 0 {
		a.errors = append(a.errors, SyntaxError{
			Start:   start,
			End:     a.pos,
			Message: "unclosed double quote",
		})
	}
	if backtickDepth%2 != 0 {
		a.errors = append(a.errors, SyntaxError{
			Start:   start,
			End:     a.pos,
			Message: "unclosed backtick",
		})
	}
	if parenDepth > 0 {
		a.errors = append(a.errors, SyntaxError{
			Start:   start,
			End:     a.pos,
			Message: "unclosed command substitution $(...)",
		})
	}

	// Determine semantic type
	semType := a.determineWordType(value, singleQuoteDepth, doubleQuoteDepth, backtickDepth, parenDepth)

	// Calculate max depth for this token
	maxDepth := max(singleQuoteDepth, doubleQuoteDepth, backtickDepth, parenDepth)

	a.addToken(semType, start, a.pos, maxDepth)

	// Update state
	if a.afterRedirection {
		a.afterRedirection = false
	} else {
		a.isFirstInCommand = false
	}
}

// determineWordType determines the semantic type of a word token.
func (a *analyzer) determineWordType(value string, singleDepth, doubleDepth, backtickDepth, parenDepth int) SemanticType {
	// Check for errors first
	if singleDepth%2 != 0 || doubleDepth%2 != 0 || backtickDepth%2 != 0 || parenDepth > 0 {
		return TypeError
	}

	// Check if it's a redirection target
	if a.afterRedirection {
		return TypeRedirectionTarget
	}

	// Check for quote types (when the entire token is quoted)
	if len(value) >= 2 {
		if value[0] == '\'' && value[len(value)-1] == '\'' {
			return TypeSingleQuotedString
		}
		if value[0] == '"' && value[len(value)-1] == '"' {
			return TypeDoubleQuotedString
		}
		if value[0] == '`' && value[len(value)-1] == '`' {
			return TypeBacktick
		}
	}

	// Check for subshell
	if strings.HasPrefix(value, "$(") && strings.HasSuffix(value, ")") {
		return TypeSubshell
	}

	// Check for variable
	if strings.HasPrefix(value, "$") && len(value) > 1 {
		return TypeVariable
	}

	// Command vs Argument
	if a.isFirstInCommand {
		return TypeCommand
	}
	return TypeArgument
}

func (a *analyzer) addToken(semType SemanticType, start, end, depth int) {
	value := string(a.input[start:end])
	a.tokens = append(a.tokens, AnalyzedToken{
		Type:  semType,
		Value: value,
		Start: start,
		End:   end,
		Depth: depth,
	})
}

func max(values ...int) int {
	if len(values) == 0 {
		return 0
	}
	m := values[0]
	for _, v := range values[1:] {
		if v > m {
			m = v
		}
	}
	return m
}

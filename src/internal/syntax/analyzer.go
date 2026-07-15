// Package syntax provides syntax analysis, highlighting, and diagnostics for shell input.
package syntax

import (
	"fmt"
	"strings"
	"unicode"

	"foundation-shell/internal/lexer"
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
	// TypeCommandSubst is a $(...) command substitution.
	TypeCommandSubst
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
	case TypeCommandSubst:
		return "CommandSubst"
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

	a.checkStructure()
}

// checkStructure reports leading-operator, consecutive-operator,
// missing-redirection-target and trailing-operator errors, matching the
// parser's messages. Whitespace and comment tokens are not significant:
// `echo hello | ` and `echo | # done` are still trailing-pipe errors.
func (a *analyzer) checkStructure() {
	// Indexes of the significant (non-whitespace, non-comment) tokens.
	var sig []int
	for i := range a.tokens {
		if a.tokens[i].Type == TypeWhitespace || a.tokens[i].Type == TypeComment {
			continue
		}
		sig = append(sig, i)
	}
	if len(sig) == 0 {
		// Only whitespace/comments: nothing to check
		return
	}

	// A leading chain operator (|, &&, ||, ;) is an error, matching the
	// parser. Redirections may legally start a command (< in.txt cat).
	first := a.tokens[sig[0]]
	if first.Type == TypeOperator {
		a.errors = append(a.errors, SyntaxError{
			Start:   first.Start,
			End:     first.End,
			Message: "unexpected operator at start: " + first.Value,
		})
		if len(sig) == 1 {
			// A lone operator is fully described by the error above.
			return
		}
	}

	// Consecutive operators, matching the parser. A newline between
	// commands is an implicit ; (the lexer emits one unless the previous
	// token is an operator, which continues the line), so a chain
	// operator that starts a new line after a word is "consecutive" with
	// that implicit ;. A redirection followed by any operator has no
	// target, again matching the parser.
	for k := 1; k < len(sig); k++ {
		prev, cur := a.tokens[sig[k-1]], a.tokens[sig[k]]
		msg := ""
		switch {
		case cur.Type == TypeOperator && prev.Type == TypeOperator:
			msg = fmt.Sprintf("consecutive operators: %s followed by %s", prev.Value, cur.Value)
		case cur.Type == TypeOperator && prev.Type != TypeRedirection && a.newlineBetween(sig[k-1], sig[k]):
			msg = "consecutive operators: ; followed by " + cur.Value
		case prev.Type == TypeRedirection && (cur.Type == TypeOperator || cur.Type == TypeRedirection):
			msg = fmt.Sprintf("missing redirection target: %s followed by operator %s", prev.Value, cur.Value)
		}
		if msg != "" {
			a.errors = append(a.errors, SyntaxError{
				Start:   cur.Start,
				End:     cur.End,
				Message: msg,
			})
		}
	}

	last := a.tokens[sig[len(sig)-1]]
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

// newlineBetween reports whether any whitespace token strictly between
// token indexes i and j contains a newline.
func (a *analyzer) newlineBetween(i, j int) bool {
	for k := i + 1; k < j; k++ {
		if a.tokens[k].Type == TypeWhitespace && strings.ContainsRune(a.tokens[k].Value, '\n') {
			return true
		}
	}
	return false
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

// subContext tracks one open command substitution ($(...) or `...`)
// inside a word, mirroring the lexer's model: kind is '(' or '`', depth is
// the same-type nesting level of a backtick region (always 1 for $(...)),
// and single/double are the nesting depths of quote regions open inside
// the body (guarding the closing delimiter).
type subContext struct {
	kind   rune
	depth  int
	single int
	double int
}

// parseWord parses a word token, handling quotes, backticks, and
// substitutions with the same depth-tracked nesting rule as the lexer
// (lexer.QuoteNestsDeeper): inside an open region, a same-type quote
// character nests one level deeper iff the previous character is
// whitespace and the next character exists, is not whitespace, and is not
// a quote character; otherwise it closes one level.
func (a *analyzer) parseWord() {
	start := a.pos

	// Outer quote regions. At most one is nonzero: inside one type, the
	// other type's characters are literal content.
	singleDepth := 0
	doubleDepth := 0
	// Open command substitutions, innermost last.
	var subStack []subContext

	// Depth bookkeeping for AnalyzedToken.Depth: curDepth is the number
	// of currently open constructs (quote nesting levels, substitutions,
	// and in-body quote levels); maxDepth is the token's high-water mark.
	curDepth, maxDepth := 0, 0
	openLevel := func() {
		curDepth++
		if curDepth > maxDepth {
			maxDepth = curDepth
		}
	}
	closeLevel := func() { curDepth-- }

	// nestOrClose applies the nesting rule to a quote/backtick depth
	// counter that is already >= 1 and returns the new depth.
	nestOrClose := func(depth int) int {
		if lexer.QuoteNestsDeeper(a.input, a.pos) {
			openLevel()
			return depth + 1
		}
		closeLevel()
		return depth - 1
	}

	for a.pos < len(a.input) {
		c := a.input[a.pos]

		inSubstitution := len(subStack) > 0
		var top *subContext
		if inSubstitution {
			top = &subStack[len(subStack)-1]
		}
		effSingle := singleDepth > 0
		if inSubstitution {
			effSingle = top.single > 0
		}

		// Handle escape sequences (outside single-quote context): the
		// escaped character never participates in depth tracking.
		if c == '\\' && !effSingle && a.pos+1 < len(a.input) {
			a.pos += 2
			continue
		}

		// $( opens a command substitution (suppressed in single quotes,
		// active inside double quotes).
		if c == '$' && !effSingle && a.pos+1 < len(a.input) && a.input[a.pos+1] == '(' {
			subStack = append(subStack, subContext{kind: '(', depth: 1})
			openLevel()
			a.pos += 2
			continue
		}

		// ) closes the innermost $(...) unless it is quoted inside the
		// substitution body: echo $(echo ')') is valid, while
		// echo "$(whoami)" still closes at the real ) because the outer
		// quote belongs to the outer context, not the body's.
		if c == ')' && inSubstitution && top.kind == '(' && top.single == 0 && top.double == 0 {
			subStack = subStack[:len(subStack)-1]
			closeLevel()
			a.pos++
			continue
		}

		// Backticks: suppressed in single quotes and inside in-body
		// double quotes; active inside outer double quotes. A backtick
		// inside a backtick region nests or closes by the shared rule.
		if c == '`' && !effSingle && !(inSubstitution && top.double > 0) {
			if inSubstitution && top.kind == '`' {
				top.depth = nestOrClose(top.depth)
				if top.depth == 0 {
					subStack = subStack[:len(subStack)-1]
				}
			} else {
				subStack = append(subStack, subContext{kind: '`', depth: 1})
				openLevel()
			}
			a.pos++
			continue
		}

		// Single quotes, depth-tracked (literal inside double quotes).
		if c == '\'' {
			if inSubstitution {
				if top.double == 0 {
					if top.single == 0 {
						top.single = 1
						openLevel()
					} else {
						top.single = nestOrClose(top.single)
					}
				}
				a.pos++
				continue
			}
			if doubleDepth == 0 {
				if singleDepth == 0 {
					singleDepth = 1
					openLevel()
				} else {
					singleDepth = nestOrClose(singleDepth)
				}
				a.pos++
				continue
			}
			// Inside double quotes: literal content, falls through
		}

		// Double quotes, depth-tracked (literal inside single quotes).
		if c == '"' {
			if inSubstitution {
				if top.single == 0 {
					if top.double == 0 {
						top.double = 1
						openLevel()
					} else {
						top.double = nestOrClose(top.double)
					}
				}
				a.pos++
				continue
			}
			if singleDepth == 0 {
				if doubleDepth == 0 {
					doubleDepth = 1
					openLevel()
				} else {
					doubleDepth = nestOrClose(doubleDepth)
				}
				a.pos++
				continue
			}
			// Inside single quotes: literal content, falls through
		}

		// Word boundaries (whitespace or operators) apply at total
		// depth 0 only.
		if singleDepth == 0 && doubleDepth == 0 && len(subStack) == 0 {
			if unicode.IsSpace(c) {
				break
			}
			if _, length := a.matchOperator(false); length > 0 {
				break
			}
		}

		a.pos++
	}

	if a.pos == start {
		return
	}
	value := string(a.input[start:a.pos])

	// Report the INNERMOST unclosed construct (one error per word,
	// matching the lexer's message for the same input). With nesting an
	// even quote count can be unclosed ('a 'b).
	unclosedMsg := ""
	switch {
	case len(subStack) > 0:
		top := subStack[len(subStack)-1]
		switch {
		case top.single > 0:
			unclosedMsg = "unclosed single quote"
		case top.double > 0:
			unclosedMsg = "unclosed double quote"
		case top.kind == '(':
			unclosedMsg = "unclosed command substitution $(...)"
		default:
			unclosedMsg = "unclosed backtick"
		}
	case singleDepth > 0:
		unclosedMsg = "unclosed single quote"
	case doubleDepth > 0:
		unclosedMsg = "unclosed double quote"
	}
	if unclosedMsg != "" {
		a.errors = append(a.errors, SyntaxError{
			Start:   start,
			End:     a.pos,
			Message: unclosedMsg,
		})
	}

	semType := a.determineWordType(value, unclosedMsg != "")

	a.addToken(semType, start, a.pos, maxDepth)

	// Update state
	if a.afterRedirection {
		a.afterRedirection = false
	} else {
		a.isFirstInCommand = false
	}
}

// determineWordType determines the semantic type of a word token.
func (a *analyzer) determineWordType(value string, unclosed bool) SemanticType {
	// Check for errors first
	if unclosed {
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

	// Check for command substitution
	if strings.HasPrefix(value, "$(") && strings.HasSuffix(value, ")") {
		return TypeCommandSubst
	}

	// Check for variable
	if isVariableWord(value) {
		return TypeVariable
	}

	// Command vs Argument
	if a.isFirstInCommand {
		return TypeCommand
	}
	return TypeArgument
}

// isVariableWord reports whether a word is a variable reference: $ followed
// by a letter/underscore (greedy name characters after), ${...} with a
// matching close brace and a non-empty name, or exactly the special
// parameter $?. Any other $ ($$, $!, $1, $-foo, a lone $, an unclosed ${)
// is a literal word character, so the word is not a variable.
func isVariableWord(value string) bool {
	if len(value) < 2 || value[0] != '$' {
		return false
	}
	if value == "$?" {
		return true
	}
	if value[1] == '{' {
		closeIdx := strings.IndexByte(value[2:], '}')
		return closeIdx > 0
	}
	c := value[1]
	return c == '_' || ('a' <= c && c <= 'z') || ('A' <= c && c <= 'Z')
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

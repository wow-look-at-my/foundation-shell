// Package syntax provides syntax analysis, highlighting, and diagnostics for shell input.
package syntax

import (
	"fmt"
	"strings"

	"foundation-shell/internal/lexer"
)

// SemanticType classifies a token for highlighting and diagnostics.
type SemanticType int

const (
	// TypeUnknown is an unclassified token.
	TypeUnknown SemanticType = iota
	// TypeCommand is the first word of a command.
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
	// TypeParenGroup is a parenthesized group (...). RESERVED: parentheses
	// are ordinary word characters today, so nothing produces this. It is
	// here for the subshell grouping operators.md lists as a future feature.
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

// Analyze performs syntax analysis on the input string, returning tokens
// with semantic types and positions plus any syntax errors.
//
// It does NOT tokenize: lexer.Scan does, and this classifies what Scan
// returns. That is what keeps highlighting, diagnostics and execution from
// disagreeing — there is one tokenizer, and this is one of its two views.
func Analyze(input string) *AnalysisResult {
	scan := lexer.Scan(input)

	tokens := make([]AnalyzedToken, 0, len(scan.Tokens))
	errors := make([]SyntaxError, 0, len(scan.Unclosed))

	// An unterminated construct swallows the rest of the input, so only the
	// LAST word can carry one. Mark it, and report every construct it left
	// open (innermost first) exactly as Scan ordered them.
	unclosedWord := -1
	if len(scan.Unclosed) > 0 {
		for i := range scan.Tokens {
			if scan.Tokens[i].Kind == lexer.KindWord {
				unclosedWord = i
			}
		}
	}
	for _, u := range scan.Unclosed {
		errors = append(errors, SyntaxError{Start: u.Start, End: u.End, Message: u.Message})
	}

	for i, tok := range scan.Tokens {
		tokens = append(tokens, AnalyzedToken{
			Type:  semanticType(tok, i == unclosedWord),
			Value: tok.Raw,
			Start: tok.Start,
			End:   tok.End,
			Depth: tok.Depth,
		})
	}

	errors = append(errors, checkStructure(tokens)...)

	return &AnalysisResult{
		Tokens: tokens,
		Errors: errors,
		Valid:  len(errors) == 0,
	}
}

// semanticType maps one scanned token to its highlighting class. The
// structural part of the answer (command vs argument vs redirection target)
// is already decided by the scanner's Role, so both views agree on it.
func semanticType(tok lexer.Token, unclosed bool) SemanticType {
	switch tok.Kind {
	case lexer.KindWhitespace:
		return TypeWhitespace
	case lexer.KindComment:
		return TypeComment
	case lexer.KindOperator:
		if lexer.IsChainOperator(tok.Content) {
			return TypeOperator
		}
		return TypeRedirection
	}

	if unclosed {
		return TypeError
	}
	if tok.Role == lexer.RoleRedirectionTarget {
		return TypeRedirectionTarget
	}

	// A wholly quoted word is colored by its quote type.
	if v := tok.Raw; len(v) >= 2 {
		switch {
		case v[0] == '\'' && v[len(v)-1] == '\'':
			return TypeSingleQuotedString
		case v[0] == '"' && v[len(v)-1] == '"':
			return TypeDoubleQuotedString
		case v[0] == '`' && v[len(v)-1] == '`':
			return TypeBacktick
		}
	}
	if strings.HasPrefix(tok.Raw, "$(") && strings.HasSuffix(tok.Raw, ")") {
		return TypeCommandSubst
	}
	if isVariableWord(tok.Raw) {
		return TypeVariable
	}

	if tok.Role == lexer.RoleCommand {
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

// checkStructure reports leading-operator, background-&, here-document,
// consecutive-operator, missing-redirection-target and trailing-operator
// errors, matching the parser's messages. Whitespace and comment tokens are
// not significant: `echo hello | ` and `echo | # done` are still
// trailing-pipe errors.
func checkStructure(tokens []AnalyzedToken) []SyntaxError {
	var errs []SyntaxError

	// Indexes of the significant (non-whitespace, non-comment) tokens.
	var sig []int
	for i := range tokens {
		if tokens[i].Type == TypeWhitespace || tokens[i].Type == TypeComment {
			continue
		}
		sig = append(sig, i)
	}
	if len(sig) == 0 {
		// Only whitespace/comments: nothing to check.
		return nil
	}

	// A leading chain operator (|, &&, ||, ;) is an error, matching the
	// parser. Redirections may legally start a command (< in.txt cat).
	first := tokens[sig[0]]
	if first.Type == TypeOperator {
		errs = append(errs, SyntaxError{
			Start:   first.Start,
			End:     first.End,
			Message: "unexpected operator at start: " + first.Value,
		})
		if len(sig) == 1 {
			// A lone operator is fully described by the error above.
			return errs
		}
	}

	// A lone unquoted `&` is an ordinary word here, not a background
	// operator, so it would be absorbed into argv along with everything
	// after it. Flag it where it is written; the parser rejects it with the
	// same message. A redirection target is excluded: `> &` is the
	// fd-duplication guard's case, and that message names it better.
	for _, k := range sig {
		tok := tokens[k]
		if tok.Value != "&" || tok.Type == TypeRedirectionTarget {
			continue
		}
		errs = append(errs, SyntaxError{
			Start:   tok.Start,
			End:     tok.End,
			Message: "background execution is not supported",
		})
	}

	// Consecutive operators, matching the parser. A newline between
	// commands is an implicit ; (the lexer materializes one unless the
	// previous token is a CHAIN operator, which continues the line), so a
	// chain operator that starts a new line after a word is "consecutive"
	// with that implicit ;. A redirection followed by any operator — or by
	// a newline, which the lexer turns into a ; (redirections do NOT
	// continue across lines) — has no target, again matching the parser.
	//
	// Index of the second `<` in a heredoc pair already reported, so `<<<`
	// yields one error instead of one per adjacent pair.
	heredocTail := -1
	for k := 1; k < len(sig); k++ {
		prev, cur := tokens[sig[k-1]], tokens[sig[k]]
		msg := ""
		at := cur
		switch {
		case prev.Type == TypeRedirection && newlineBetween(tokens, sig[k-1], sig[k]):
			// `echo hi ><newline>out.txt`: the lexer emits > ; out.txt, so the
			// parser sees the separator as the redirection target. The
			// caret points at the dangling redirection operator.
			msg = "missing redirection target: " + prev.Value + " followed by operator ;"
			at = prev
		case cur.Type == TypeOperator && prev.Type == TypeOperator:
			msg = fmt.Sprintf("consecutive operators: %s followed by %s", prev.Value, cur.Value)
		case cur.Type == TypeOperator && newlineBetween(tokens, sig[k-1], sig[k]):
			msg = "consecutive operators: ; followed by " + cur.Value
		case prev.Value == "<" && cur.Value == "<":
			// `<<EOF` / `<<<word` lex as consecutive `<`. The caret points at
			// the first one, where the construct starts.
			if sig[k-1] == heredocTail {
				// `<<<` is three `<`, so it forms two adjacent pairs. The
				// first pair already reported this construct.
				continue
			}
			msg = "here-documents are not supported"
			at = prev
			heredocTail = sig[k]
		case prev.Type == TypeRedirection && (cur.Type == TypeOperator || cur.Type == TypeRedirection):
			msg = fmt.Sprintf("missing redirection target: %s followed by operator %s", prev.Value, cur.Value)
		}
		if msg != "" {
			errs = append(errs, SyntaxError{Start: at.Start, End: at.End, Message: msg})
		}
	}

	last := tokens[sig[len(sig)-1]]
	if last.Type == TypeOperator && last.Value != ";" {
		errs = append(errs, SyntaxError{
			Start:   last.Start,
			End:     last.End,
			Message: "unexpected operator at end",
		})
	}
	if last.Type == TypeRedirection {
		errs = append(errs, SyntaxError{
			Start:   last.Start,
			End:     last.End,
			Message: "missing redirection target",
		})
	}

	return errs
}

// newlineBetween reports whether any whitespace token strictly between
// token indexes i and j contains a newline.
func newlineBetween(tokens []AnalyzedToken, i, j int) bool {
	for k := i + 1; k < j; k++ {
		if tokens[k].Type == TypeWhitespace && strings.ContainsRune(tokens[k].Value, '\n') {
			return true
		}
	}
	return false
}

// Package syntax provides syntax analysis, highlighting, and diagnostics for shell input.
package syntax

import (
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

	for _, p := range lexer.Validate(scan.Tokens) {
		errors = append(errors, SyntaxError{
			Start:   p.Start,
			End:     p.End,
			Message: describeProblem(p),
		})
	}

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

// describeProblem renders one structural problem for a caret diagnostic.
// The rules live in lexer.Validate, shared with the parser; only the wording
// differs. These messages omit the offending operator where the caret
// already points at it (diagnostics.md §5.1).
func describeProblem(p lexer.Problem) string {
	switch p.Kind {
	case lexer.ProblemOperatorAtStart:
		return "unexpected operator at start: " + p.Text
	case lexer.ProblemTrailingOperator:
		return "unexpected operator at end"
	case lexer.ProblemConsecutiveOperators:
		return "consecutive operators: " + p.Text + " followed by " + p.Other
	case lexer.ProblemMissingRedirectionTarget:
		if p.Other == "" {
			return "missing redirection target"
		}
		return "missing redirection target: " + p.Text + " followed by operator " + p.Other
	case lexer.ProblemBackgroundUnsupported:
		return "background execution is not supported"
	case lexer.ProblemHeredocUnsupported:
		return "here-documents are not supported"
	}
	return "invalid syntax"
}

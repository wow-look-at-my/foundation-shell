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

// TokenContext is the execution view of a token: what to run, with the
// quoting facts expansion needs. It is projected from Scan's richer Token,
// which also carries positions for highlighting and diagnostics.
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
	// WasEscaped is true when any part of the word carried a backslash
	// escape. Only \$ and \` leave a marker in Content, so a guard that
	// reads Content alone cannot tell `\&` from a bare `&`. The 2>/2>>
	// recognition uses the same flag to keep `\2>f` out of the operator.
	WasEscaped bool
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

// QuoteNestsDeeper implements the depth-tracked quote nesting rule. It
// reports whether an unescaped quote character at runes[i], read while a
// region of the SAME type is already open, opens a nested level instead of
// closing one. It nests one level deeper iff the previous input character is
// whitespace AND the next input character exists, is not whitespace, and is
// not a quote character (', ", or `). In every other case the character
// closes one level.
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

// Tokenize splits the input into the tokens the parser executes. Scan owns
// the tokenizing rules — quoting, substitution, escapes, comments, operators
// — and this function adds only what execution needs and analysis does not:
//
//   - Whitespace and comments are dropped.
//   - An unquoted newline outside substitutions becomes a ; operator token,
//     materialized just before the next token — unless no token has been
//     emitted yet, or the previous token is a CHAIN operator (|, &&, ||, ;),
//     which makes the newline a line continuation. Redirection operators do
//     NOT suppress it: a redirection cannot continue across a newline, so
//     `cmd ><newline>file` lexes as cmd > ; file and the parser rejects it.
//     Leading blank lines and a trailing newline produce nothing.
//   - An unterminated construct is an error rather than a record. Scan lists
//     every one, innermost first; the innermost is what this returns:
//     "unclosed single quote", "unclosed double quote", "unclosed backtick",
//     or "unclosed command substitution $(...)". With nesting an EVEN number
//     of quote characters can still be unclosed ('a 'b).
//
// A word consisting only of quotes ('' or "") produces an empty token with
// the corresponding quoting flags set.
func Tokenize(input string) ([]TokenContext, error) {
	result := Scan(input)
	if len(result.Unclosed) > 0 {
		return nil, errors.New(result.Unclosed[0].Message)
	}
	return Project(result), nil
}

// Project reduces a scan to the execution view: whitespace and comments
// dropped, and an unquoted newline turned into the ; separator it stands
// for. A caller that also wants Validate's problems scans once and calls
// both, instead of scanning twice.
func Project(result *ScanResult) []TokenContext {
	var tokens []TokenContext
	pendingNewline := false

	for _, tok := range result.Tokens {
		switch tok.Kind {
		case KindComment:
			continue
		case KindWhitespace:
			if strings.ContainsRune(tok.Raw, '\n') {
				pendingNewline = true
			}
			continue
		}

		if pendingNewline {
			pendingNewline = false
			prev := len(tokens) - 1
			if prev >= 0 && !(tokens[prev].IsOperator && IsChainOperator(tokens[prev].Content)) {
				tokens = append(tokens, TokenContext{Content: ";", IsOperator: true})
			}
		}

		tokens = append(tokens, TokenContext{
			Content:         tok.Content,
			WasSingleQuoted: tok.WasSingleQuoted,
			WasQuoted:       tok.WasQuoted,
			WasEscaped:      tok.WasEscaped,
			IsOperator:      tok.Kind == KindOperator,
		})
	}

	return tokens
}

// StripEscapeMarkers removes escape markers from a string, used after
// expansion has been performed. It applies to every marked character
// (\$ and \` both use the marker).
func StripEscapeMarkers(s string) string {
	return strings.ReplaceAll(s, string(EscapeMarker), "")
}

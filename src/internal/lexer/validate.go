package lexer

import "strings"

// ProblemKind names a structural problem. Callers match on the kind and
// render their own wording: the parser's messages name the offending
// operator because it has no other way to point at it, while the analyzer's
// caret already does that and its messages stay shorter. Same rule, two
// surfaces (diagnostics.md §5.1 and §5.2).
type ProblemKind int

const (
	// ProblemOperatorAtStart is a chain operator with no left operand.
	ProblemOperatorAtStart ProblemKind = iota
	// ProblemTrailingOperator is a chain operator with no right operand. A
	// trailing ; is valid and never reported.
	ProblemTrailingOperator
	// ProblemConsecutiveOperators is two chain operators in a row, counting
	// the implicit ; an unquoted newline stands for.
	ProblemConsecutiveOperators
	// ProblemMissingRedirectionTarget is a redirection with no filename
	// after it. Other is empty when the redirection simply ends the input.
	ProblemMissingRedirectionTarget
	// ProblemBackgroundUnsupported is a lone unquoted & used as a word.
	ProblemBackgroundUnsupported
	// ProblemHeredocUnsupported is << or <<<, which lex as adjacent <.
	ProblemHeredocUnsupported
)

// Problem is one structural fault, with the span to point a caret at and
// the token text a message may name.
type Problem struct {
	Kind  ProblemKind
	Start int
	End   int
	// Text is the offending token as written.
	Text string
	// Other is the second token's text for a rule about a pair, and empty
	// otherwise.
	Other string
}

// Validate reports the structural faults in a scanned token sequence. It is
// the only implementation of these rules: the parser turns the first
// problem into its sentinel error, and the analyzer renders every problem as
// a caret diagnostic.
//
// Whitespace and comments are not significant — `echo hello | ` and
// `echo | # done` are both trailing-pipe errors — but a whitespace run
// holding a newline IS: the lexer materializes a ; separator for it, so a
// redirection before one has lost its target and a chain operator after one
// follows an operator.
func Validate(tokens []Token) []Problem {
	var problems []Problem

	// Indexes of the significant tokens.
	var sig []int
	for i := range tokens {
		if tokens[i].Kind == KindWhitespace || tokens[i].Kind == KindComment {
			continue
		}
		sig = append(sig, i)
	}
	if len(sig) == 0 {
		return nil
	}

	// A leading chain operator is an error. Redirections may legally start a
	// command: `< in.txt cat` runs cat.
	first := tokens[sig[0]]
	if isChainOperatorToken(first) {
		problems = append(problems, Problem{
			Kind:  ProblemOperatorAtStart,
			Start: first.Start,
			End:   first.End,
			Text:  first.Raw,
		})
		if len(sig) == 1 {
			// A lone operator is fully described by the problem above.
			return problems
		}
	}

	// A lone unquoted & is an ordinary word, not a background operator, so
	// it would be absorbed into argv along with everything after it. A
	// redirection target is excluded: `> &` is the fd-duplication case, and
	// the parser names that one better.
	for _, k := range sig {
		tok := tokens[k]
		if tok.Kind != KindWord || tok.Raw != "&" || tok.Role == RoleRedirectionTarget {
			continue
		}
		problems = append(problems, Problem{
			Kind:  ProblemBackgroundUnsupported,
			Start: tok.Start,
			End:   tok.End,
			Text:  tok.Raw,
		})
	}

	// heredocTail is the index of the second < in a pair already reported,
	// so `<<<` yields one problem instead of one per adjacent pair.
	heredocTail := -1
	for k := 1; k < len(sig); k++ {
		prev, cur := tokens[sig[k-1]], tokens[sig[k]]
		newline := newlineBetween(tokens, sig[k-1], sig[k])

		switch {
		case isRedirectionToken(prev) && newline:
			// `echo hi ><newline>out.txt` lexes as > ; out.txt, so the
			// separator is where the target should have been. The caret
			// points at the dangling redirection.
			problems = append(problems, Problem{
				Kind:  ProblemMissingRedirectionTarget,
				Start: prev.Start,
				End:   prev.End,
				Text:  prev.Raw,
				Other: ";",
			})
		case isChainOperatorToken(cur) && isChainOperatorToken(prev):
			problems = append(problems, Problem{
				Kind:  ProblemConsecutiveOperators,
				Start: cur.Start,
				End:   cur.End,
				Text:  prev.Raw,
				Other: cur.Raw,
			})
		case isChainOperatorToken(cur) && newline:
			problems = append(problems, Problem{
				Kind:  ProblemConsecutiveOperators,
				Start: cur.Start,
				End:   cur.End,
				Text:  ";",
				Other: cur.Raw,
			})
		case prev.Raw == "<" && cur.Raw == "<" && isOperatorToken(prev) && isOperatorToken(cur):
			if sig[k-1] == heredocTail {
				// `<<<` is three <, forming two adjacent pairs; the first
				// pair already reported this construct.
				continue
			}
			problems = append(problems, Problem{
				Kind:  ProblemHeredocUnsupported,
				Start: prev.Start,
				End:   prev.End,
				Text:  prev.Raw,
			})
			heredocTail = sig[k]
		case isRedirectionToken(prev) && isOperatorToken(cur):
			problems = append(problems, Problem{
				Kind:  ProblemMissingRedirectionTarget,
				Start: cur.Start,
				End:   cur.End,
				Text:  prev.Raw,
				Other: cur.Raw,
			})
		}
	}

	last := tokens[sig[len(sig)-1]]
	switch {
	case isChainOperatorToken(last) && last.Raw != ";":
		problems = append(problems, Problem{
			Kind:  ProblemTrailingOperator,
			Start: last.Start,
			End:   last.End,
			Text:  last.Raw,
		})
	case isRedirectionToken(last):
		problems = append(problems, Problem{
			Kind:  ProblemMissingRedirectionTarget,
			Start: last.Start,
			End:   last.End,
			Text:  last.Raw,
		})
	}

	return problems
}

func isOperatorToken(t Token) bool { return t.Kind == KindOperator }

func isChainOperatorToken(t Token) bool {
	return t.Kind == KindOperator && IsChainOperator(t.Content)
}

func isRedirectionToken(t Token) bool {
	return t.Kind == KindOperator && isRedirectionOp(t.Content)
}

// newlineBetween reports whether any whitespace strictly between token
// indexes i and j contains a newline.
func newlineBetween(tokens []Token, i, j int) bool {
	for k := i + 1; k < j; k++ {
		if tokens[k].Kind == KindWhitespace && strings.ContainsRune(tokens[k].Raw, '\n') {
			return true
		}
	}
	return false
}

package lexer

import "errors"

// SubstitutionSpan describes one top-level command-substitution span
// ($(...) or `...`) inside a token's Content as produced by Tokenize.
// Indices are RUNE indices into the content (convert the content with
// []rune before slicing).
type SubstitutionSpan struct {
	// Start is the rune index of the span's first character: the $ of
	// $(...) or the opening backtick.
	Start int
	// End is the rune index one past the closing ) or closing backtick.
	End int
	// Body is the substitution body between the delimiters, verbatim
	// (exactly as Tokenize preserved it): it is re-parsed when the
	// substitution executes.
	Body string
}

// ErrUnclosedSubstitution is returned when a substitution span does not
// close before the end of the content. Content produced by Tokenize always
// has balanced spans, so hitting this indicates hand-built input (or the
// pathological quote-adjacent nesting corner described in the doc comment
// of FindSubstitutionSpans).
var ErrUnclosedSubstitution = errors.New("unclosed command substitution")

// FindSubstitutionSpans scans a value token's content left to right and
// returns its top-level command-substitution spans. It applies the SAME
// rules Tokenize used when it preserved those spans verbatim, reusing the
// shared subContext state machine and QuoteNestsDeeper nesting rule:
//
//   - An EscapeMarker makes the following character (a marked $ or `)
//     literal: it never opens a span.
//   - At top level, $( and ` open a span. Quote characters at top level
//     are literal data: any quote that reached a non-single-quoted token's
//     content was either nested-literal or inside double quotes, and in
//     both cases Tokenize kept substitutions active there. (Single-quoted
//     tokens suppress expansion whole-token and must not be scanned.)
//   - Inside a span, nested $(...) push, backticks nest per
//     QuoteNestsDeeper, quote characters track the body's quote state, and
//     backslashes escape the following character, all exactly as in
//     Tokenize's substitution-body scan. The span ends when its outermost
//     delimiter closes.
//
// The neighbor characters QuoteNestsDeeper inspects come from the token
// content rather than the raw input, so characters Tokenize stripped
// (outer quotes) or resolved (escapes) directly adjacent to a nested
// closing delimiter can, in pathological inputs, change a close into a
// nest; such content fails with ErrUnclosedSubstitution rather than
// silently mis-splitting.
func FindSubstitutionSpans(content string) ([]SubstitutionSpan, error) {
	runes := []rune(content)
	var spans []SubstitutionSpan
	var stack []subContext
	spanStart := -1

	pop := func(i int) {
		stack = stack[:len(stack)-1]
		if len(stack) == 0 {
			bodyStart := spanStart + 1 // opening backtick
			if runes[spanStart] == '$' {
				bodyStart = spanStart + 2 // $(
			}
			spans = append(spans, SubstitutionSpan{
				Start: spanStart,
				End:   i + 1,
				Body:  string(runes[bodyStart:i]),
			})
			spanStart = -1
		}
	}

	for i := 0; i < len(runes); i++ {
		c := runes[i]

		// A marked $ or ` (from \$ / \`) is a literal character.
		if c == EscapeMarker {
			i++
			continue
		}

		inSub := len(stack) > 0
		var top *subContext
		if inSub {
			top = &stack[len(stack)-1]
		}

		// Backslash pairs inside a body are consumed exactly like
		// Tokenize's body scan: an escaped ) ` ' " must not affect state.
		// At top level a backslash is literal data (Tokenize already
		// resolved top-level escapes), so it must NOT hide a following
		// span opener.
		if inSub && c == '\\' && top.single == 0 && i+1 < len(runes) {
			i++
			continue
		}

		// $( opens a substitution: at top level it starts a span; inside a
		// body it pushes unless the body position is single-quoted.
		if c == '$' && i+1 < len(runes) && runes[i+1] == '(' {
			if !inSub || top.single == 0 {
				if !inSub {
					spanStart = i
				}
				stack = append(stack, subContext{kind: '(', depth: 1})
				i++
				continue
			}
		}

		// ) closes the innermost $(...) when the body's quote state is
		// clean.
		if c == ')' && inSub && top.kind == '(' && top.single == 0 && top.double == 0 {
			pop(i)
			continue
		}

		// Backticks: literal when single-quoted or double-quoted inside a
		// body; otherwise they nest/close a backtick region or open one.
		if c == '`' && !(inSub && (top.single > 0 || top.double > 0)) {
			switch {
			case inSub && top.kind == '`' && QuoteNestsDeeper(runes, i):
				top.depth++
			case inSub && top.kind == '`':
				top.depth--
				if top.depth == 0 {
					pop(i)
				}
			default:
				if !inSub {
					spanStart = i
				}
				stack = append(stack, subContext{kind: '`', depth: 1})
			}
			continue
		}

		// Quote characters only matter inside a body, where they guard the
		// closing delimiter (same depth-tracked rule as Tokenize).
		if c == '\'' && inSub && top.double == 0 {
			switch {
			case top.single == 0:
				top.single = 1
			case QuoteNestsDeeper(runes, i):
				top.single++
			default:
				top.single--
			}
			continue
		}
		if c == '"' && inSub && top.single == 0 {
			switch {
			case top.double == 0:
				top.double = 1
			case QuoteNestsDeeper(runes, i):
				top.double++
			default:
				top.double--
			}
			continue
		}
	}

	if len(stack) > 0 {
		return nil, ErrUnclosedSubstitution
	}
	return spans, nil
}

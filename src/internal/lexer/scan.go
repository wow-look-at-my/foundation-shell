package lexer

import (
	"strings"
	"unicode"
)

// Kind classifies what the scanner found at a position, before any
// structural role is assigned.
type Kind int

const (
	// KindWord is a word: a command name, an argument, or a redirection
	// target. Its Content is expansion-ready; its Raw is what was typed.
	KindWord Kind = iota
	// KindOperator is one of | || && ; < > >> 2> 2>>.
	KindOperator
	// KindWhitespace is a run of whitespace between tokens. A run holding a
	// newline separates commands (§3.4).
	KindWhitespace
	// KindComment is a # comment, running to the next newline.
	KindComment
)

// Role is a token's structural position, assigned once for every consumer:
// the parser builds a chain from it and the highlighter colors by it.
type Role int

const (
	// RoleNone is every non-word token.
	RoleNone Role = iota
	// RoleCommand is the first word of a command.
	RoleCommand
	// RoleArgument is any later word of a command.
	RoleArgument
	// RoleRedirectionTarget is the word right after a redirection operator.
	RoleRedirectionTarget
)

// Token is one lexical unit with everything any consumer needs: the text as
// typed (Raw) and where it was (Start, End) for highlighting and caret
// diagnostics, the expansion-ready text (Content) and quoting flags for
// execution, and the structural Role both agree on.
//
// Every rune of the input belongs to exactly one token, whitespace and
// comments included, so Raw concatenated over a scan reproduces the input.
type Token struct {
	// Content is the expansion-ready text of a word: outer quotes removed,
	// escapes resolved, and markers inserted for \$ and \`. For every other
	// Kind it equals Raw.
	Content string
	// Raw is the text exactly as written.
	Raw string
	// Start and End are rune offsets into the input; End is exclusive.
	Start int
	End   int
	Kind  Kind
	Role  Role
	// WasSingleQuoted is true when part of the word was single-quoted. It
	// suppresses all expansion for the whole token.
	WasSingleQuoted bool
	// WasQuoted is true when part of the word was inside any quotes. Quote
	// characters inside a substitution body do NOT set it: they belong to
	// the body, which is re-parsed when the substitution runs.
	WasQuoted bool
	// WasEscaped is true when part of the word carried a backslash escape.
	// Only \$ and \` leave a marker in Content, so a guard reading Content
	// alone cannot tell `\&` from a bare `&`.
	WasEscaped bool
	// Depth is the word's high-water nesting level over quote regions and
	// substitutions. Non-word tokens are 0.
	Depth int
}

// Unclosed is one unterminated construct, with the span of the word that
// left it open.
type Unclosed struct {
	Start   int
	End     int
	Message string
}

// ScanResult is a complete pass over the input.
type ScanResult struct {
	Tokens []Token
	// Unclosed lists every unterminated construct, innermost first within a
	// word. Empty means the input is lexically complete.
	Unclosed []Unclosed
}

// Scan tokenizes input in one pass, keeping every rune. It is the only
// tokenizer: Tokenize projects the execution view out of it, and the syntax
// analyzer projects the highlighting and diagnostic view out of it.
//
// Unlike Tokenize it never fails. An unterminated construct is recorded in
// Unclosed and the word is still emitted, so a caller that wants to report
// every problem can, and a caller that wants the first one takes it.
func Scan(input string) *ScanResult {
	s := &scanner{runes: []rune(input)}
	s.run()
	return &ScanResult{Tokens: s.tokens, Unclosed: s.unclosed}
}

type scanner struct {
	runes    []rune
	tokens   []Token
	unclosed []Unclosed
}

func (s *scanner) run() {
	var current strings.Builder
	wasSingleQuoted := false
	wasQuoted := false
	wordHadEscape := false
	// wordStart is the offset of the current word's first rune, or -1 when
	// no word is open.
	wordStart := -1
	// curDepth counts the constructs open right now; maxDepth is the
	// current word's high-water mark.
	curDepth, maxDepth := 0, 0

	// Nesting depths of the outer quote regions. At most one is nonzero:
	// inside one type, the other type's characters are literal content.
	singleDepth := 0
	doubleDepth := 0
	var subStack []subContext

	runes := s.runes

	beginWord := func(i int) {
		if wordStart < 0 {
			wordStart = i
		}
	}
	openLevel := func() {
		curDepth++
		if curDepth > maxDepth {
			maxDepth = curDepth
		}
	}
	closeLevel := func() { curDepth-- }

	// flushWord ends the current word at end, emitting a token when the word
	// has content or consisted only of quotes (an empty argument). It ALWAYS
	// resets the per-word state: an empty quoted word ('' / "") must not leak
	// its flags into the next word, so `echo '' $HOME` still expands $HOME.
	flushWord := func(end int) {
		if wordStart >= 0 && (current.Len() > 0 || wasQuoted) {
			s.tokens = append(s.tokens, Token{
				Content:         current.String(),
				Raw:             string(runes[wordStart:end]),
				Start:           wordStart,
				End:             end,
				Kind:            KindWord,
				WasSingleQuoted: wasSingleQuoted,
				WasQuoted:       wasQuoted,
				WasEscaped:      wordHadEscape,
				Depth:           maxDepth,
			})
		}
		current.Reset()
		wasSingleQuoted = false
		wasQuoted = false
		wordHadEscape = false
		wordStart = -1
		curDepth, maxDepth = 0, 0
	}

	emitOperator := func(op string, start, end int) {
		s.tokens = append(s.tokens, Token{
			Content: op,
			Raw:     string(runes[start:end]),
			Start:   start,
			End:     end,
			Kind:    KindOperator,
		})
	}

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

		// Backslash escapes (not inside single quotes).
		if c == '\\' && !effSingle && i+1 < len(runes) {
			beginWord(i)
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
				current.WriteString(unescape(next))
			}
			i++
			continue
		}

		// $( opens a command substitution. Active inside double quotes;
		// suppressed in any single-quote context. $(...) regions nest by
		// pushing a new context: their delimiters are asymmetric, so no
		// same-type nesting rule is needed.
		if c == '$' && !effSingle && i+1 < len(runes) && runes[i+1] == '(' {
			beginWord(i)
			subStack = append(subStack, subContext{kind: '(', depth: 1})
			openLevel()
			current.WriteRune('$')
			current.WriteRune('(')
			i++
			continue
		}

		// ) closes the innermost $(...) unless it is quoted inside the body.
		if c == ')' && inSubstitution && top.kind == '(' && top.single == 0 && top.double == 0 {
			beginWord(i)
			subStack = subStack[:len(subStack)-1]
			closeLevel()
			current.WriteRune(c)
			continue
		}

		// Backticks. Suppressed in single quotes and when double-quoted
		// inside a body; active inside outer double quotes. Inside a backtick
		// region a backtick either NESTS or closes one level, so
		// `outer `inner` end` is one substitution.
		if c == '`' && !effSingle && !(inSubstitution && top.double > 0) {
			beginWord(i)
			switch {
			case inSubstitution && top.kind == '`' && QuoteNestsDeeper(runes, i):
				top.depth++
				openLevel()
			case inSubstitution && top.kind == '`':
				top.depth--
				closeLevel()
				if top.depth == 0 {
					subStack = subStack[:len(subStack)-1]
				}
			default:
				subStack = append(subStack, subContext{kind: '`', depth: 1})
				openLevel()
			}
			current.WriteRune(c)
			continue
		}

		// Single quotes.
		if c == '\'' {
			if inSubstitution {
				// Body text, preserved verbatim; depth-tracked so a quoted )
				// or ` does not close the substitution. Does not set the
				// token's quoting flags.
				beginWord(i)
				if top.double == 0 {
					switch {
					case top.single == 0:
						top.single = 1
						openLevel()
					case QuoteNestsDeeper(runes, i):
						top.single++
						openLevel()
					default:
						top.single--
						closeLevel()
					}
				}
				current.WriteRune(c)
				continue
			}
			if doubleDepth == 0 {
				beginWord(i)
				switch {
				case singleDepth == 0:
					// Opening quote of a region: stripped from content.
					singleDepth = 1
					openLevel()
					wasSingleQuoted = true
					wasQuoted = true
				case QuoteNestsDeeper(runes, i):
					// Nested opener: stays literal in the content.
					singleDepth++
					openLevel()
					current.WriteRune(c)
				default:
					singleDepth--
					closeLevel()
					if singleDepth > 0 {
						// Nested closer: stays literal in the content.
						current.WriteRune(c)
					}
					// Outermost closer (depth 0): stripped.
				}
				continue
			}
			// Inside double quotes: literal, falls through.
		}

		// Double quotes.
		if c == '"' {
			if inSubstitution {
				beginWord(i)
				if top.single == 0 {
					switch {
					case top.double == 0:
						top.double = 1
						openLevel()
					case QuoteNestsDeeper(runes, i):
						top.double++
						openLevel()
					default:
						top.double--
						closeLevel()
					}
				}
				current.WriteRune(c)
				continue
			}
			if singleDepth == 0 {
				beginWord(i)
				switch {
				case doubleDepth == 0:
					doubleDepth = 1
					openLevel()
					wasQuoted = true
				case QuoteNestsDeeper(runes, i):
					doubleDepth++
					openLevel()
					current.WriteRune(c)
				default:
					doubleDepth--
					closeLevel()
					if doubleDepth > 0 {
						current.WriteRune(c)
					}
				}
				continue
			}
			// Inside single quotes: literal, falls through.
		}

		atTopLevel := singleDepth == 0 && doubleDepth == 0 && !inSubstitution

		// A # at word start, outside any substitution, starts a comment
		// running to the next newline. foo#bar stays one word. A # inside a
		// substitution body is body text: comment rules apply only when the
		// body is re-parsed at execution time.
		if c == '#' && atTopLevel && current.Len() == 0 && !wasQuoted {
			start := i
			for i+1 < len(runes) && runes[i+1] != '\n' {
				i++
			}
			s.tokens = append(s.tokens, Token{
				Content: string(runes[start : i+1]),
				Raw:     string(runes[start : i+1]),
				Start:   start,
				End:     i + 1,
				Kind:    KindComment,
			})
			continue
		}

		// Operators without surrounding whitespace. An unquoted, unescaped
		// operator character outside any substitution ends the current word
		// and lexes an operator by maximal munch over {||, &&, 2>>, 2>, >>,
		// |, ;, <, >}. Quoted operator characters never reach here (the quote
		// handlers consume them); escaped ones go to the escape handler.
		if atTopLevel {
			switch c {
			case '&':
				// A single & is NOT an operator: a&b stays one word. Only &&
				// is recognized. The parser rejects a lone & as a word.
				if i+1 < len(runes) && runes[i+1] == '&' {
					flushWord(i)
					emitOperator("&&", i, i+2)
					i++
					continue
				}
			case '|':
				flushWord(i)
				if i+1 < len(runes) && runes[i+1] == '|' {
					emitOperator("||", i, i+2)
					i++
				} else {
					emitOperator("|", i, i+1)
				}
				continue
			case ';':
				flushWord(i)
				emitOperator(";", i, i+1)
				continue
			case '<':
				flushWord(i)
				emitOperator("<", i, i+1)
				continue
			case '>':
				op := ">"
				end := i + 1
				if i+1 < len(runes) && runes[i+1] == '>' {
					op = ">>"
					end = i + 2
					i++
				}
				// 2> / 2>> apply only when the pending word is exactly an
				// unquoted, unescaped "2": that 2 is consumed into the
				// operator, so `echo a2>f` keeps word a2 with operator >.
				start := end - len(op)
				if current.Len() == 1 && current.String() == "2" && !wasQuoted && !wordHadEscape {
					start = wordStart
					op = "2" + op
					current.Reset()
					wordStart = -1
					curDepth, maxDepth = 0, 0
				} else {
					flushWord(start)
				}
				emitOperator(op, start, end)
				continue
			}
		}

		// Whitespace separates tokens. A run becomes ONE token, and a run
		// holding a newline separates commands.
		if unicode.IsSpace(c) && atTopLevel {
			flushWord(i)
			start := i
			for i+1 < len(runes) && unicode.IsSpace(runes[i+1]) {
				i++
			}
			s.tokens = append(s.tokens, Token{
				Content: string(runes[start : i+1]),
				Raw:     string(runes[start : i+1]),
				Start:   start,
				End:     i + 1,
				Kind:    KindWhitespace,
			})
			continue
		}

		beginWord(i)
		current.WriteRune(c)
	}

	// Record every unterminated construct, INNERMOST first: an open
	// substitution, or a quote open inside its body, is always inner to an
	// outer quote region, because quotes inside a body only touch the body's
	// own context. Each message appears at most once per word — a
	// doubly-nested region like 'a 'b is still ONE unclosed single quote.
	end := len(runes)
	start := wordStart
	if start < 0 {
		start = end
	}
	var seen []string
	add := func(msg string) {
		for _, existing := range seen {
			if existing == msg {
				return
			}
		}
		seen = append(seen, msg)
		s.unclosed = append(s.unclosed, Unclosed{Start: start, End: end, Message: msg})
	}
	for k := len(subStack) - 1; k >= 0; k-- {
		sc := subStack[k]
		// A body quote region opened inside its substitution, so it is inner
		// to it: report it first.
		if sc.single > 0 {
			add("unclosed single quote")
		}
		if sc.double > 0 {
			add("unclosed double quote")
		}
		if sc.kind == '(' {
			add("unclosed command substitution $(...)")
		} else {
			add("unclosed backtick")
		}
	}
	if singleDepth > 0 {
		add("unclosed single quote")
	}
	if doubleDepth > 0 {
		add("unclosed double quote")
	}

	flushWord(end)
	assignRoles(s.tokens)
}

// unescape maps the character after a backslash to its literal text. Only \$
// and \` keep a marker, which stops the expander from reading them as
// syntax; StripEscapeMarkers removes it after expansion.
func unescape(next rune) string {
	switch next {
	case '$':
		return string(EscapeMarker) + "$"
	case '`':
		return string(EscapeMarker) + "`"
	case 'n':
		return "\n"
	case 't':
		return "\t"
	default:
		// \\ \" \' \<space> and anything else: the character itself.
		return string(next)
	}
}

// assignRoles labels each word with its structural position. Both consumers
// read this instead of re-deriving it: the parser to build the chain, the
// highlighter to color command names differently from arguments.
//
// A chain operator starts a new command, and so does a newline (the lexer
// treats an unquoted newline as a soft separator, §3.4). A redirection
// operator claims the next word as its target, which is not a command name
// even at the start of a line: `< in.txt cat` runs cat.
func assignRoles(tokens []Token) {
	firstInCommand := true
	afterRedirection := false

	for i := range tokens {
		switch tokens[i].Kind {
		case KindComment:
			continue
		case KindWhitespace:
			if strings.ContainsRune(tokens[i].Raw, '\n') {
				firstInCommand = true
				afterRedirection = false
			}
			continue
		case KindOperator:
			if isRedirectionOp(tokens[i].Content) {
				afterRedirection = true
			} else {
				firstInCommand = true
				afterRedirection = false
			}
			continue
		}

		switch {
		case afterRedirection:
			tokens[i].Role = RoleRedirectionTarget
			afterRedirection = false
		case firstInCommand:
			tokens[i].Role = RoleCommand
			firstInCommand = false
		default:
			tokens[i].Role = RoleArgument
		}
	}
}

// isRedirectionOp reports whether an operator's text redirects a stream.
func isRedirectionOp(op string) bool {
	switch op {
	case "<", ">", ">>", "2>", "2>>":
		return true
	}
	return false
}

// IsChainOperator reports whether an operator's text is one of the CHAIN
// operators. Only these suppress the newline separator (line continuation,
// §3.4); a redirection must not silently take the next line's first word as
// its target.
func IsChainOperator(op string) bool {
	switch op {
	case "|", "&&", "||", ";":
		return true
	}
	return false
}

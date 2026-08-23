// Package parser orchestrates lexer tokenization, expansion, token classification, and command chain building.
package parser

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"foundation-shell/internal/expander"
	"foundation-shell/internal/lexer"
	"foundation-shell/internal/token"
)

var (
	// ErrEmptyInput is returned when the input is empty or contains only whitespace.
	ErrEmptyInput = errors.New("empty input")
	// ErrOperatorAtStart is returned when an operator appears at the start of input.
	ErrOperatorAtStart = errors.New("unexpected operator at start")
	// ErrMissingRedirectionTarget is returned when a redirection has no target file.
	ErrMissingRedirectionTarget = errors.New("missing redirection target")
	// ErrEmptyRedirectionTarget is returned when a redirection target expands
	// to an empty string (e.g. `> $UNSET`). The message is canonical: the
	// spec pins the exact string "empty redirection target".
	ErrEmptyRedirectionTarget = errors.New("empty redirection target")
	// ErrFdDuplicationUnsupported is returned when an unquoted redirection
	// target starts with & (e.g. `2>&1`, which lexes as `2>` + `&1`). The
	// message is canonical: the spec pins the exact string.
	ErrFdDuplicationUnsupported = errors.New("file descriptor duplication is not supported")
	// ErrBackgroundUnsupported is returned when a lone unquoted & appears as
	// a word (`cmd &`, `cmd & other`). This shell has no background jobs, so
	// the & lexes as an ordinary word (lexer.md §3.3.2). Without this guard
	// the & and EVERY word after it become arguments to the command, which
	// then runs in the foreground: `server &` blocks forever instead of
	// returning, and `cmd_a & cmd_b` never runs cmd_b. The message is
	// canonical: the spec pins the exact string.
	ErrBackgroundUnsupported = errors.New("background execution is not supported")
	// ErrHeredocUnsupported is returned for `<<` and `<<<`, which lex as
	// consecutive `<` operators. The generic missing-target message sends a
	// reader hunting for a filename that was never the point. The message is
	// canonical: the spec pins the exact string.
	ErrHeredocUnsupported = errors.New("here-documents are not supported")
	// ErrAssignmentThenUse is returned when one input assigns a variable and
	// a LATER command in the same input expands it. The whole input is
	// expanded in ONE pass before anything runs (expansion.md §Special
	// Parameters), so that expansion reads the value from BEFORE the input —
	// normally empty. Without this guard `OUT=$(cmd); echo $OUT` yields an
	// empty string and says nothing, while `printenv OUT` in the same input
	// prints the real value. The message is canonical: the spec pins the
	// exact string.
	ErrAssignmentThenUse = errors.New("variable is assigned and used in the same input")
	// ErrTrailingOperator is returned when the input ends with an operator.
	ErrTrailingOperator = errors.New("unexpected operator at end")
	// ErrConsecutiveOperators is returned when two chain operators appear consecutively.
	ErrConsecutiveOperators = errors.New("consecutive operators")
	// ErrEmptyCommand is returned when a command is empty.
	ErrEmptyCommand = errors.New("empty command")
)

// CommandSpec represents a single command with its arguments and I/O redirections.
type CommandSpec struct {
	Args         []string
	InputFile    string
	OutputFile   string
	ErrorFile    string
	AppendOutput bool
	AppendError  bool
	// IsAssignment marks a standalone assignment (execution.md §Standalone
	// Assignment): the command's SOLE word (redirections do not count) was
	// not single-quoted anywhere and its expanded value matches NAME=VALUE
	// with a valid variable name. The executor performs the assignment
	// (Args[0] split at the first '=') instead of running a command.
	IsAssignment bool
}

// Chain represents a sequence of commands connected by operators.
type Chain struct {
	Commands  []*CommandSpec
	Operators []token.TokenType // len = len(Commands) - 1
}

// classifiedToken represents a token after expansion and classification.
type classifiedToken struct {
	tokenType token.TokenType
	value     string
	// rawValue is the token content AS WRITTEN (the lexer's pre-expansion
	// Content). The fd-duplication guard inspects it: `2>&1` is rejected on
	// the literal `&1`, while a target that becomes `&1` only through
	// expansion (`> $X` with X='&1') is a legal filename
	// (redirection.md §9.5).
	rawValue string
	// wasQuoted is true when any part of the original word was quoted
	// (used e.g. to allow a quoted '&1' as a redirection target while
	// rejecting the unquoted fd-duplication form).
	wasQuoted bool
	// wasSingleQuoted is true when any part of the original word was
	// single-quoted (whole-token granularity, lexer.md §4.4.2). A
	// single-quoted part anywhere disqualifies the word from standalone
	// assignment recognition.
	wasSingleQuoted bool
	// wasEscaped is true when any part of the original word carried a
	// backslash escape. The background guard needs it: `\&` and `&` reach
	// the parser with identical content.
	wasEscaped bool
}

// assignmentPattern matches an expanded word that forms a standalone
// assignment: a valid variable name followed by '='. Everything after the
// first '=' is the value (possibly empty, possibly containing more '=').
var assignmentPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*=`)

// problemError renders one structural problem from lexer.Validate as this
// package's error. The rules live there, shared with the syntax analyzer;
// only the wording differs. These messages name the offending operator
// because a parse error has no caret to point with (diagnostics.md §5.2).
func problemError(p lexer.Problem) error {
	switch p.Kind {
	case lexer.ProblemOperatorAtStart:
		return fmt.Errorf("%w: %s", ErrOperatorAtStart, p.Text)
	case lexer.ProblemTrailingOperator:
		return fmt.Errorf("%w: %s", ErrTrailingOperator, p.Text)
	case lexer.ProblemConsecutiveOperators:
		return fmt.Errorf("%w: %s followed by %s", ErrConsecutiveOperators, p.Text, p.Other)
	case lexer.ProblemMissingRedirectionTarget:
		if p.Other == "" {
			return fmt.Errorf("%w: %s", ErrMissingRedirectionTarget, p.Text)
		}
		return fmt.Errorf("%w: %s followed by operator %s", ErrMissingRedirectionTarget, p.Text, p.Other)
	case lexer.ProblemBackgroundUnsupported:
		return ErrBackgroundUnsupported
	case lexer.ProblemHeredocUnsupported:
		return ErrHeredocUnsupported
	}
	return fmt.Errorf("internal error: unknown problem kind %d", p.Kind)
}

// parseOperator checks if a string is an operator and returns its token type.
// Returns (tokenType, true) if it's an operator, (0, false) otherwise.
func parseOperator(s string) (token.TokenType, bool) {
	switch s {
	case "|":
		return token.Pipe, true
	case "&&":
		return token.And, true
	case "||":
		return token.Or, true
	case ";":
		return token.Semicolon, true
	case "<":
		return token.RedirectStdIn, true
	case ">":
		return token.RedirectStdOut, true
	case ">>":
		return token.RedirectStdOutAppend, true
	case "2>":
		return token.RedirectStdErr, true
	case "2>>":
		return token.RedirectStdErrAppend, true
	default:
		return 0, false
	}
}

// isChainOperator returns true if the token type is a chain operator (|, &&, ||, ;).
func isChainOperator(t token.TokenType) bool {
	return t == token.Pipe || t == token.And || t == token.Or || t == token.Semicolon
}

// isRedirectionOperator returns true if the token type is a redirection operator.
func isRedirectionOperator(t token.TokenType) bool {
	return t == token.RedirectStdIn || t == token.RedirectStdOut ||
		t == token.RedirectStdOutAppend || t == token.RedirectStdErr ||
		t == token.RedirectStdErrAppend
}

// Options configures parsing.
type Options struct {
	// Executor expands command substitutions ($(...) and `...`). When nil,
	// substitution spans are left verbatim.
	Executor expander.SubshellExecutor
	// LastStatus reports the exit status of the most recent command for $?
	// expansion. When nil, $? expands to 0.
	LastStatus func() int
}

// Parse parses the input string into a command chain.
// It performs lexer tokenization, expansion, token classification, and chain building.
// This version does not expand command substitutions ($(...) or backticks),
// and $? expands to 0.
func Parse(input string) (*Chain, error) {
	return ParseWithOptions(input, Options{})
}

// ParseWithExecutor parses the input string into a command chain with optional subshell expansion.
// If executor is non-nil, command substitutions ($(...) and `...`) will be expanded.
// $? expands to 0.
func ParseWithExecutor(input string, executor expander.SubshellExecutor) (*Chain, error) {
	return ParseWithOptions(input, Options{Executor: executor})
}

// ParseWithOptions parses the input string into a command chain with full
// control over expansion: command substitution via opts.Executor and $?
// expansion via opts.LastStatus.
func ParseWithOptions(input string, opts Options) (*Chain, error) {
	// Step 1: Scan the input ONCE, then read both views of it. Scan is the
	// only tokenizer; Validate is the only implementation of the structural
	// rules, shared with the syntax analyzer so a caret diagnostic and a
	// parse error can never disagree about what is legal.
	scan := lexer.Scan(input)
	if len(scan.Unclosed) > 0 {
		return nil, fmt.Errorf("tokenization error: %w", errors.New(scan.Unclosed[0].Message))
	}
	if problems := lexer.Validate(scan.Tokens); len(problems) > 0 {
		return nil, problemError(problems[0])
	}

	tokenContexts := lexer.Project(scan)
	if len(tokenContexts) == 0 {
		return nil, ErrEmptyInput
	}

	// Step 2: Expand and classify tokens
	var classified []classifiedToken
	isFirstInCommand := true

	for _, tc := range tokenContexts {
		// Operators come ONLY from the lexer's marking: a token is an
		// operator iff tc.IsOperator. Quoted or escaped operator characters
		// ('|', "|", \|, 'a|b') arrive as value tokens and stay literal data.
		if tc.IsOperator {
			opType, isOp := parseOperator(tc.Content)
			if !isOp {
				// Unreachable: the lexer only emits operators from the set
				// parseOperator knows. Guard against future drift.
				return nil, fmt.Errorf("internal error: lexer emitted unknown operator %q", tc.Content)
			}
			classified = append(classified, classifiedToken{
				tokenType: opType,
				value:     tc.Content,
			})
			// Reset position tracking when we hit a chain operator
			if isChainOperator(opType) {
				isFirstInCommand = true
			}
			continue
		}

		// It's a value token - expand if not single-quoted
		expandedValue := tc.Content
		if !tc.WasSingleQuoted {
			// Tilde expansion is suppressed when ANY part of the word was
			// quoted: quoting the tilde ("~", '~', ~"/x") is deliberate.
			// Variable and command expansion still apply to double-quoted
			// words.
			if !tc.WasQuoted {
				expandedValue = expander.ExpandTilde(expandedValue)
			}
			// Single pass: variables (incl. $?) expand in the literal text,
			// top-level substitution spans execute recursively via the
			// executor, and their output is spliced without re-scanning.
			expanded, expandErr := expander.ExpandToken(expandedValue, expander.Options{
				Executor:   opts.Executor,
				LastStatus: opts.LastStatus,
			})
			if expandErr != nil {
				return nil, fmt.Errorf("command substitution error: %w", expandErr)
			}
			expandedValue = expanded
		}
		// Strip escape markers UNCONDITIONALLY, single-quoted tokens
		// included: a mixed word like 'a'\$b has WasSingleQuoted set (no
		// expansion) but still carries a marker from the unquoted \$.
		expandedValue = lexer.StripEscapeMarkers(expandedValue)

		// Classify as Command or CommandArgument
		var tokenType token.TokenType
		if isFirstInCommand {
			tokenType = token.Command
			isFirstInCommand = false
		} else {
			tokenType = token.CommandArgument
		}

		classified = append(classified, classifiedToken{
			tokenType:       tokenType,
			value:           expandedValue,
			rawValue:        tc.Content,
			wasQuoted:       tc.WasQuoted,
			wasSingleQuoted: tc.WasSingleQuoted,
			wasEscaped:      tc.WasEscaped,
		})
	}

	// Step 3: Validate syntax and build chain
	if err := checkAssignmentThenUse(classified); err != nil {
		return nil, err
	}
	return buildChain(classified)
}

// buildChain builds a command chain from classified tokens.
//
// lexer.Validate has already run, so the operator SEQUENCE is known good:
// no leading or trailing chain operator, no consecutive pair, and every
// redirection has a following word. What is left here needs the expanded
// values, which Validate never sees: an empty command, a target that
// expanded to nothing, and the fd-duplication guard on the written word.
func buildChain(tokens []classifiedToken) (*Chain, error) {
	if len(tokens) == 0 {
		return nil, ErrEmptyInput
	}

	chain := &Chain{
		Commands:  make([]*CommandSpec, 0),
		Operators: make([]token.TokenType, 0),
	}

	currentCommand := &CommandSpec{
		Args: make([]string, 0),
	}

	// firstArgSingleQuoted remembers whether the current command's FIRST
	// word had any single-quoted part; standalone assignment recognition
	// (execution.md §Standalone Assignment) needs it at finalize time.
	firstArgSingleQuoted := false

	// finalize marks a completed command as a standalone assignment when
	// its sole word is an unsingle-quoted NAME=... form.
	finalize := func(cmd *CommandSpec) *CommandSpec {
		if len(cmd.Args) == 1 && !firstArgSingleQuoted && assignmentPattern.MatchString(cmd.Args[0]) {
			cmd.IsAssignment = true
		}
		return cmd
	}

	// Set when a trailing semicolon was consumed: the chain is complete and
	// the post-loop "dangling operator" checks must not fire.
	trailingSemicolonConsumed := false

	for i := 0; i < len(tokens); i++ {
		tok := tokens[i]

		if isChainOperator(tok.tokenType) {
			// Validate: command must not be empty
			if len(currentCommand.Args) == 0 {
				return nil, ErrEmptyCommand
			}

			// Add current command to chain
			chain.Commands = append(chain.Commands, finalize(currentCommand))

			// A trailing semicolon is valid and simply consumed (the lexer
			// also emits one for a trailing newline). Any OTHER trailing
			// operator was already rejected by Validate.
			if i+1 >= len(tokens) {
				trailingSemicolonConsumed = true
				continue // loop ends; no operator is recorded
			}

			chain.Operators = append(chain.Operators, tok.tokenType)

			// Start a new command
			currentCommand = &CommandSpec{
				Args: make([]string, 0),
			}
			firstArgSingleQuoted = false
			continue
		}

		if isRedirectionOperator(tok.tokenType) {
			// Validate guaranteed a following word, so the index is safe and
			// the token is not an operator.
			nextTok := tokens[i+1]

			// A syntactically present redirection must have a non-empty
			// target after expansion: `> $UNSET` is an error, not a
			// silently dropped redirection.
			if nextTok.value == "" {
				return nil, ErrEmptyRedirectionTarget
			}

			// Fd-duplication syntax (2>&1) lexes as `2>` + word `&1`.
			// Reject unquoted &-prefixed targets loudly instead of
			// creating a file literally named "&1". The guard inspects the
			// PRE-expansion token content (redirection.md §9.5): a quoted
			// '&1' is a legal filename, and so is a target that becomes
			// `&1` only through expansion (`> $X` with X='&1').
			if !nextTok.wasQuoted && strings.HasPrefix(nextTok.rawValue, "&") {
				word := lexer.StripEscapeMarkers(nextTok.rawValue)
				return nil, fmt.Errorf("%w: %s", ErrFdDuplicationUnsupported, word)
			}

			// Apply the redirection
			switch tok.tokenType {
			case token.RedirectStdIn:
				currentCommand.InputFile = nextTok.value
			case token.RedirectStdOut:
				currentCommand.OutputFile = nextTok.value
				currentCommand.AppendOutput = false
			case token.RedirectStdOutAppend:
				currentCommand.OutputFile = nextTok.value
				currentCommand.AppendOutput = true
			case token.RedirectStdErr:
				currentCommand.ErrorFile = nextTok.value
				currentCommand.AppendError = false
			case token.RedirectStdErrAppend:
				currentCommand.ErrorFile = nextTok.value
				currentCommand.AppendError = true
			}

			// Skip the target token
			i++
			continue
		}

		// Value token - add to current command's args
		if len(currentCommand.Args) == 0 {
			firstArgSingleQuoted = tok.wasSingleQuoted
		}
		currentCommand.Args = append(currentCommand.Args, tok.value)
	}

	// Add the last command if it has any args
	switch {
	case trailingSemicolonConsumed:
		// Chain already complete; nothing to append or validate here.
	case len(currentCommand.Args) == 0:
		if len(chain.Commands) == 0 || len(chain.Operators) == 0 {
			return nil, ErrEmptyCommand
		}
		// Reachable via e.g. `cmd ; > file`: an operator followed by a
		// redirection-only command. Report the dangling operator with the
		// same `: <op>` context every other trailing-operator error carries.
		return nil, fmt.Errorf("%w: %s", ErrTrailingOperator, chain.Operators[len(chain.Operators)-1])
	default:
		chain.Commands = append(chain.Commands, finalize(currentCommand))
	}

	// Validate invariant: operators.size() == commands.size() - 1
	if len(chain.Commands) > 0 && len(chain.Operators) != len(chain.Commands)-1 {
		return nil, fmt.Errorf("invalid chain: %d commands should have %d operators, but has %d",
			len(chain.Commands), len(chain.Commands)-1, len(chain.Operators))
	}

	return chain, nil
}

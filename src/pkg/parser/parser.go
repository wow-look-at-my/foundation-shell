// Package parser orchestrates lexer tokenization, expansion, token classification, and command chain building.
package parser

import (
	"errors"
	"fmt"
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
	// wasQuoted is true when any part of the original word was quoted
	// (used e.g. to allow a quoted '&1' as a redirection target while
	// rejecting the unquoted fd-duplication form).
	wasQuoted bool
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

// Parse parses the input string into a command chain.
// It performs lexer tokenization, expansion, token classification, and chain building.
// This version does not expand command substitutions ($(...) or backticks).
func Parse(input string) (*Chain, error) {
	return ParseWithExecutor(input, nil)
}

// ParseWithExecutor parses the input string into a command chain with optional subshell expansion.
// If executor is non-nil, command substitutions ($(...) and `...`) will be expanded.
func ParseWithExecutor(input string, executor expander.SubshellExecutor) (*Chain, error) {
	// Step 1: Tokenize input
	tokenContexts, err := lexer.Tokenize(input)
	if err != nil {
		return nil, fmt.Errorf("tokenization error: %w", err)
	}

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
			// Single pass: variables expand in the literal text, top-level
			// substitution spans execute recursively via the executor, and
			// their output is spliced without re-scanning.
			expandedValue, err = expander.ExpandToken(expandedValue, expander.Options{Executor: executor})
			if err != nil {
				return nil, fmt.Errorf("command substitution error: %w", err)
			}
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
			tokenType: tokenType,
			value:     expandedValue,
			wasQuoted: tc.WasQuoted,
		})
	}

	// Step 3: Validate syntax and build chain
	return buildChain(classified)
}

// buildChain builds a command chain from classified tokens.
func buildChain(tokens []classifiedToken) (*Chain, error) {
	if len(tokens) == 0 {
		return nil, ErrEmptyInput
	}

	// Check for chain operator at start (|, &&, ||)
	// Redirection operators at start are valid: `< input.txt cat`
	if isChainOperator(tokens[0].tokenType) {
		return nil, fmt.Errorf("%w: %s", ErrOperatorAtStart, tokens[0].value)
	}

	chain := &Chain{
		Commands:  make([]*CommandSpec, 0),
		Operators: make([]token.TokenType, 0),
	}

	currentCommand := &CommandSpec{
		Args: make([]string, 0),
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
			chain.Commands = append(chain.Commands, currentCommand)

			// Check if there's anything after this operator. A single
			// trailing semicolon is valid and simply consumed (the lexer
			// also emits one for a trailing newline); any other trailing
			// operator is an error.
			if i+1 >= len(tokens) {
				if tok.tokenType == token.Semicolon {
					trailingSemicolonConsumed = true
					continue // loop ends; no operator is recorded
				}
				return nil, fmt.Errorf("%w: %s", ErrTrailingOperator, tok.value)
			}

			chain.Operators = append(chain.Operators, tok.tokenType)

			// Check for consecutive chain operators
			nextTok := tokens[i+1]
			if isChainOperator(nextTok.tokenType) {
				return nil, fmt.Errorf("%w: %s followed by %s", ErrConsecutiveOperators, tok.value, nextTok.value)
			}

			// Start a new command
			currentCommand = &CommandSpec{
				Args: make([]string, 0),
			}
			continue
		}

		if isRedirectionOperator(tok.tokenType) {
			// Redirection needs a target
			if i+1 >= len(tokens) {
				return nil, fmt.Errorf("%w: %s", ErrMissingRedirectionTarget, tok.value)
			}

			nextTok := tokens[i+1]
			// Target must be a value, not an operator
			if nextTok.tokenType.IsOperator() {
				return nil, fmt.Errorf("%w: %s followed by operator %s", ErrMissingRedirectionTarget, tok.value, nextTok.value)
			}

			// A syntactically present redirection must have a non-empty
			// target after expansion: `> $UNSET` is an error, not a
			// silently dropped redirection.
			if nextTok.value == "" {
				return nil, ErrEmptyRedirectionTarget
			}

			// Fd-duplication syntax (2>&1) lexes as `2>` + word `&1`.
			// Reject unquoted &-prefixed targets loudly instead of
			// creating a file literally named "&1". A quoted '&1'
			// remains a legal filename.
			if !nextTok.wasQuoted && strings.HasPrefix(nextTok.value, "&") {
				return nil, fmt.Errorf("%w: %s", ErrFdDuplicationUnsupported, nextTok.value)
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
		chain.Commands = append(chain.Commands, currentCommand)
	}

	// Validate invariant: operators.size() == commands.size() - 1
	if len(chain.Commands) > 0 && len(chain.Operators) != len(chain.Commands)-1 {
		return nil, fmt.Errorf("invalid chain: %d commands should have %d operators, but has %d",
			len(chain.Commands), len(chain.Commands)-1, len(chain.Operators))
	}

	return chain, nil
}

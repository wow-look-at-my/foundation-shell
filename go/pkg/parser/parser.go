// Package parser orchestrates lexer tokenization, expansion, token classification, and command chain building.
package parser

import (
	"errors"
	"fmt"

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

// isChainOperator returns true if the token type is a chain operator (|, &&, ||).
func isChainOperator(t token.TokenType) bool {
	return t == token.Pipe || t == token.And || t == token.Or
}

// isRedirectionOperator returns true if the token type is a redirection operator.
func isRedirectionOperator(t token.TokenType) bool {
	return t == token.RedirectStdIn || t == token.RedirectStdOut ||
		t == token.RedirectStdOutAppend || t == token.RedirectStdErr ||
		t == token.RedirectStdErrAppend
}

// Parse parses the input string into a command chain.
// It performs lexer tokenization, expansion, token classification, and chain building.
func Parse(input string) (*Chain, error) {
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
		// Try to parse as operator first (operators are not expanded)
		if opType, isOp := parseOperator(tc.Content); isOp {
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
			expandedValue = expander.ExpandTilde(expandedValue)
			expandedValue = expander.ExpandEnvironment(expandedValue)
		}
		// Strip escape markers after expansion
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

	for i := 0; i < len(tokens); i++ {
		tok := tokens[i]

		if isChainOperator(tok.tokenType) {
			// Validate: command must not be empty
			if len(currentCommand.Args) == 0 {
				return nil, ErrEmptyCommand
			}

			// Add current command to chain
			chain.Commands = append(chain.Commands, currentCommand)
			chain.Operators = append(chain.Operators, tok.tokenType)

			// Check if there's anything after this operator
			if i+1 >= len(tokens) {
				return nil, fmt.Errorf("%w: %s", ErrTrailingOperator, tok.value)
			}

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
	if len(currentCommand.Args) == 0 {
		// This shouldn't happen with valid input, but check anyway
		if len(chain.Commands) == 0 {
			return nil, ErrEmptyCommand
		}
		// Remove the trailing operator since there's no command after it
		if len(chain.Operators) > 0 {
			return nil, ErrTrailingOperator
		}
	} else {
		chain.Commands = append(chain.Commands, currentCommand)
	}

	// Validate invariant: operators.size() == commands.size() - 1
	if len(chain.Commands) > 0 && len(chain.Operators) != len(chain.Commands)-1 {
		return nil, fmt.Errorf("invalid chain: %d commands should have %d operators, but has %d",
			len(chain.Commands), len(chain.Commands)-1, len(chain.Operators))
	}

	return chain, nil
}

// Package token provides lexical token types and token structures for shell parsing.
package token

// TokenType represents the type of a lexical token.
type TokenType int

const (
	// Value types - tokens that carry a string value

	// Command represents a command name (the first word in a command).
	Command TokenType = iota + 1
	// CommandArgument represents arguments to a command.
	CommandArgument

	// Operator types - tokens that represent shell operators

	// Pipe represents the pipe operator (|).
	Pipe
	// And represents the logical AND operator (&&).
	And
	// Or represents the logical OR operator (||).
	Or
	// Semicolon represents the command separator (;).
	Semicolon
	// RedirectStdIn represents input redirection (<).
	RedirectStdIn
	// RedirectStdOut represents output redirection (>).
	RedirectStdOut
	// RedirectStdOutAppend represents appending output redirection (>>).
	RedirectStdOutAppend
	// RedirectStdErr represents stderr redirection (2>).
	RedirectStdErr
	// RedirectStdErrAppend represents appending stderr redirection (2>>).
	RedirectStdErrAppend
)

// IsValue returns true if the token type carries a string value.
func (t TokenType) IsValue() bool {
	return t == Command || t == CommandArgument
}

// IsOperator returns true if the token type is an operator.
func (t TokenType) IsOperator() bool {
	switch t {
	case Pipe, And, Or, Semicolon, RedirectStdIn, RedirectStdOut, RedirectStdOutAppend, RedirectStdErr, RedirectStdErrAppend:
		return true
	default:
		return false
	}
}

// String returns the string representation of the token type.
func (t TokenType) String() string {
	switch t {
	case Command:
		return "Command"
	case CommandArgument:
		return "CommandArgument"
	case Pipe:
		return "|"
	case And:
		return "&&"
	case Or:
		return "||"
	case Semicolon:
		return ";"
	case RedirectStdIn:
		return "<"
	case RedirectStdOut:
		return ">"
	case RedirectStdOutAppend:
		return ">>"
	case RedirectStdErr:
		return "2>"
	case RedirectStdErrAppend:
		return "2>>"
	default:
		return "Unknown"
	}
}

package token

import (
	"errors"
	"fmt"
)

var (
	// ErrInvalidTokenType is returned when creating a token with an invalid type.
	ErrInvalidTokenType = errors.New("invalid token type")
	// ErrEmptyValue is returned when creating a value token with an empty value.
	ErrEmptyValue = errors.New("value token cannot have an empty value")
)

// Token represents a lexical token from shell input.
type Token struct {
	Type  TokenType
	Value string // Empty for operator tokens
}

// NewValueToken creates a new token that carries a string value.
// It returns an error if the type is not a value type or if the value is empty.
func NewValueToken(typ TokenType, value string) (Token, error) {
	if !typ.IsValue() {
		return Token{}, fmt.Errorf("%w: expected value type, got %s", ErrInvalidTokenType, typ)
	}
	if value == "" {
		return Token{}, ErrEmptyValue
	}
	return Token{Type: typ, Value: value}, nil
}

// NewOperatorToken creates a new operator token.
// It returns an error if the type is not an operator type.
func NewOperatorToken(typ TokenType) (Token, error) {
	if !typ.IsOperator() {
		return Token{}, fmt.Errorf("%w: expected operator type, got %s", ErrInvalidTokenType, typ)
	}
	return Token{Type: typ, Value: ""}, nil
}

// String returns a human-readable representation of the token.
func (t Token) String() string {
	if t.Type.IsValue() {
		return fmt.Sprintf("%s(%q)", t.Type, t.Value)
	}
	return t.Type.String()
}

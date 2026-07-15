package token

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTokenType_IsValue(t *testing.T) {
	tests := []struct {
		name     string
		typ      TokenType
		expected bool
	}{
		{"Command is value type", Command, true},
		{"CommandArgument is value type", CommandArgument, true},
		{"Pipe is not value type", Pipe, false},
		{"And is not value type", And, false},
		{"Or is not value type", Or, false},
		{"RedirectStdIn is not value type", RedirectStdIn, false},
		{"RedirectStdOut is not value type", RedirectStdOut, false},
		{"RedirectStdOutAppend is not value type", RedirectStdOutAppend, false},
		{"RedirectStdErr is not value type", RedirectStdErr, false},
		{"RedirectStdErrAppend is not value type", RedirectStdErrAppend, false},
		{"Zero value is not value type", TokenType(0), false},
		{"Unknown type is not value type", TokenType(100), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.typ.IsValue()
			assert.Equal(t, tt.expected, got)

		})
	}
}

func TestTokenType_IsOperator(t *testing.T) {
	tests := []struct {
		name     string
		typ      TokenType
		expected bool
	}{
		{"Pipe is operator", Pipe, true},
		{"And is operator", And, true},
		{"Or is operator", Or, true},
		{"RedirectStdIn is operator", RedirectStdIn, true},
		{"RedirectStdOut is operator", RedirectStdOut, true},
		{"RedirectStdOutAppend is operator", RedirectStdOutAppend, true},
		{"RedirectStdErr is operator", RedirectStdErr, true},
		{"RedirectStdErrAppend is operator", RedirectStdErrAppend, true},
		{"Command is not operator", Command, false},
		{"CommandArgument is not operator", CommandArgument, false},
		{"Zero value is not operator", TokenType(0), false},
		{"Unknown type is not operator", TokenType(100), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.typ.IsOperator()
			assert.Equal(t, tt.expected, got)

		})
	}
}

func TestTokenType_MutuallyExclusive(t *testing.T) {
	// Test that value types and operator types are mutually exclusive
	allTypes := []TokenType{
		Command, CommandArgument,
		Pipe, And, Or,
		RedirectStdIn, RedirectStdOut, RedirectStdOutAppend,
		RedirectStdErr, RedirectStdErrAppend,
	}

	for _, typ := range allTypes {
		isValue := typ.IsValue()
		isOperator := typ.IsOperator()

		assert.False(t, isValue && isOperator)

		assert.False(t, !isValue && !isOperator)

	}
}

func TestTokenType_String(t *testing.T) {
	tests := []struct {
		typ      TokenType
		expected string
	}{
		{Command, "Command"},
		{CommandArgument, "CommandArgument"},
		{Pipe, "|"},
		{And, "&&"},
		{Or, "||"},
		{RedirectStdIn, "<"},
		{RedirectStdOut, ">"},
		{RedirectStdOutAppend, ">>"},
		{RedirectStdErr, "2>"},
		{RedirectStdErrAppend, "2>>"},
		{TokenType(0), "Unknown"},
		{TokenType(100), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := tt.typ.String()
			assert.Equal(t, tt.expected, got)

		})
	}
}

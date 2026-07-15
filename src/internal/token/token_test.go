package token

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestNewValueToken(t *testing.T) {
	tests := []struct {
		name      string
		typ       TokenType
		value     string
		wantErr   error
		wantValue string
	}{
		{
			name:      "Command with value",
			typ:       Command,
			value:     "ls",
			wantErr:   nil,
			wantValue: "ls",
		},
		{
			name:      "CommandArgument with value",
			typ:       CommandArgument,
			value:     "-la",
			wantErr:   nil,
			wantValue: "-la",
		},
		{
			name:      "Command with spaces in value",
			typ:       Command,
			value:     "hello world",
			wantErr:   nil,
			wantValue: "hello world",
		},
		{
			name:    "Command with empty value",
			typ:     Command,
			value:   "",
			wantErr: ErrEmptyValue,
		},
		{
			name:    "CommandArgument with empty value",
			typ:     CommandArgument,
			value:   "",
			wantErr: ErrEmptyValue,
		},
		{
			name:    "Operator type for value token",
			typ:     Pipe,
			value:   "test",
			wantErr: ErrInvalidTokenType,
		},
		{
			name:    "And operator type for value token",
			typ:     And,
			value:   "test",
			wantErr: ErrInvalidTokenType,
		},
		{
			name:    "Zero type for value token",
			typ:     TokenType(0),
			value:   "test",
			wantErr: ErrInvalidTokenType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tok, err := NewValueToken(tt.typ, tt.value)

			if tt.wantErr != nil {
				require.NotNil(t, err)

				assert.True(t, errors.Is(err, tt.wantErr))

				return
			}

			require.Nil(t, err)

			assert.Equal(t, tt.typ, tok.Type)

			assert.Equal(t, tt.wantValue, tok.Value)

		})
	}
}

func TestNewOperatorToken(t *testing.T) {
	tests := []struct {
		name    string
		typ     TokenType
		wantErr error
	}{
		{"Pipe operator", Pipe, nil},
		{"And operator", And, nil},
		{"Or operator", Or, nil},
		{"RedirectStdIn operator", RedirectStdIn, nil},
		{"RedirectStdOut operator", RedirectStdOut, nil},
		{"RedirectStdOutAppend operator", RedirectStdOutAppend, nil},
		{"RedirectStdErr operator", RedirectStdErr, nil},
		{"RedirectStdErrAppend operator", RedirectStdErrAppend, nil},
		{"Command type for operator token", Command, ErrInvalidTokenType},
		{"CommandArgument type for operator token", CommandArgument, ErrInvalidTokenType},
		{"Zero type for operator token", TokenType(0), ErrInvalidTokenType},
		{"Unknown type for operator token", TokenType(100), ErrInvalidTokenType},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tok, err := NewOperatorToken(tt.typ)

			if tt.wantErr != nil {
				require.NotNil(t, err)

				assert.True(t, errors.Is(err, tt.wantErr))

				return
			}

			require.Nil(t, err)

			assert.Equal(t, tt.typ, tok.Type)

			assert.Equal(t, "", tok.Value)

		})
	}
}

func TestToken_String(t *testing.T) {
	tests := []struct {
		name     string
		token    Token
		expected string
	}{
		{
			name:     "Command token",
			token:    Token{Type: Command, Value: "ls"},
			expected: `Command("ls")`,
		},
		{
			name:     "CommandArgument token",
			token:    Token{Type: CommandArgument, Value: "-la"},
			expected: `CommandArgument("-la")`,
		},
		{
			name:     "Command with special chars",
			token:    Token{Type: Command, Value: `hello "world"`},
			expected: `Command("hello \"world\"")`,
		},
		{
			name:     "Pipe operator",
			token:    Token{Type: Pipe},
			expected: "|",
		},
		{
			name:     "And operator",
			token:    Token{Type: And},
			expected: "&&",
		},
		{
			name:     "Or operator",
			token:    Token{Type: Or},
			expected: "||",
		},
		{
			name:     "RedirectStdIn operator",
			token:    Token{Type: RedirectStdIn},
			expected: "<",
		},
		{
			name:     "RedirectStdOut operator",
			token:    Token{Type: RedirectStdOut},
			expected: ">",
		},
		{
			name:     "RedirectStdOutAppend operator",
			token:    Token{Type: RedirectStdOutAppend},
			expected: ">>",
		},
		{
			name:     "RedirectStdErr operator",
			token:    Token{Type: RedirectStdErr},
			expected: "2>",
		},
		{
			name:     "RedirectStdErrAppend operator",
			token:    Token{Type: RedirectStdErrAppend},
			expected: "2>>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.token.String()
			assert.Equal(t, tt.expected, got)

		})
	}
}

func TestTokenCreation_Integration(t *testing.T) {
	// Test a realistic token sequence: ls -la | grep foo > output.txt
	tokens := []Token{}

	// ls
	tok, err := NewValueToken(Command, "ls")
	require.Nil(t, err)

	tokens = append(tokens, tok)

	// -la
	tok, err = NewValueToken(CommandArgument, "-la")
	require.Nil(t, err)

	tokens = append(tokens, tok)

	// |
	tok, err = NewOperatorToken(Pipe)
	require.Nil(t, err)

	tokens = append(tokens, tok)

	// grep
	tok, err = NewValueToken(Command, "grep")
	require.Nil(t, err)

	tokens = append(tokens, tok)

	// foo
	tok, err = NewValueToken(CommandArgument, "foo")
	require.Nil(t, err)

	tokens = append(tokens, tok)

	// >
	tok, err = NewOperatorToken(RedirectStdOut)
	require.Nil(t, err)

	tokens = append(tokens, tok)

	// output.txt
	tok, err = NewValueToken(CommandArgument, "output.txt")
	require.Nil(t, err)

	tokens = append(tokens, tok)

	// Verify token count
	assert.Equal(t, 7, len(tokens))

	// Verify types
	expectedTypes := []TokenType{
		Command, CommandArgument, Pipe, Command, CommandArgument, RedirectStdOut, CommandArgument,
	}
	for i, expected := range expectedTypes {
		assert.Equal(t, expected, tokens[i].Type)

	}

	// Verify values
	expectedValues := []string{"ls", "-la", "", "grep", "foo", "", "output.txt"}
	for i, expected := range expectedValues {
		assert.Equal(t, expected, tokens[i].Value)

	}
}

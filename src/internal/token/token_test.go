package token

import (
	"errors"
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
			if got := tt.typ.IsValue(); got != tt.expected {
				t.Errorf("TokenType(%d).IsValue() = %v, want %v", tt.typ, got, tt.expected)
			}
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
			if got := tt.typ.IsOperator(); got != tt.expected {
				t.Errorf("TokenType(%d).IsOperator() = %v, want %v", tt.typ, got, tt.expected)
			}
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

		if isValue && isOperator {
			t.Errorf("TokenType %s is both value and operator", typ)
		}
		if !isValue && !isOperator {
			t.Errorf("TokenType %s is neither value nor operator", typ)
		}
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
			if got := tt.typ.String(); got != tt.expected {
				t.Errorf("TokenType(%d).String() = %q, want %q", tt.typ, got, tt.expected)
			}
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
				if err == nil {
					t.Fatalf("NewValueToken(%v, %q) expected error containing %v, got nil", tt.typ, tt.value, tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("NewValueToken(%v, %q) error = %v, want error containing %v", tt.typ, tt.value, err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("NewValueToken(%v, %q) unexpected error: %v", tt.typ, tt.value, err)
			}

			if tok.Type != tt.typ {
				t.Errorf("Token.Type = %v, want %v", tok.Type, tt.typ)
			}
			if tok.Value != tt.wantValue {
				t.Errorf("Token.Value = %q, want %q", tok.Value, tt.wantValue)
			}
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
				if err == nil {
					t.Fatalf("NewOperatorToken(%v) expected error containing %v, got nil", tt.typ, tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("NewOperatorToken(%v) error = %v, want error containing %v", tt.typ, err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("NewOperatorToken(%v) unexpected error: %v", tt.typ, err)
			}

			if tok.Type != tt.typ {
				t.Errorf("Token.Type = %v, want %v", tok.Type, tt.typ)
			}
			if tok.Value != "" {
				t.Errorf("Token.Value = %q, want empty string", tok.Value)
			}
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
			if got := tt.token.String(); got != tt.expected {
				t.Errorf("Token.String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestTokenCreation_Integration(t *testing.T) {
	// Test a realistic token sequence: ls -la | grep foo > output.txt
	tokens := []Token{}

	// ls
	tok, err := NewValueToken(Command, "ls")
	if err != nil {
		t.Fatalf("Failed to create Command token: %v", err)
	}
	tokens = append(tokens, tok)

	// -la
	tok, err = NewValueToken(CommandArgument, "-la")
	if err != nil {
		t.Fatalf("Failed to create CommandArgument token: %v", err)
	}
	tokens = append(tokens, tok)

	// |
	tok, err = NewOperatorToken(Pipe)
	if err != nil {
		t.Fatalf("Failed to create Pipe token: %v", err)
	}
	tokens = append(tokens, tok)

	// grep
	tok, err = NewValueToken(Command, "grep")
	if err != nil {
		t.Fatalf("Failed to create Command token: %v", err)
	}
	tokens = append(tokens, tok)

	// foo
	tok, err = NewValueToken(CommandArgument, "foo")
	if err != nil {
		t.Fatalf("Failed to create CommandArgument token: %v", err)
	}
	tokens = append(tokens, tok)

	// >
	tok, err = NewOperatorToken(RedirectStdOut)
	if err != nil {
		t.Fatalf("Failed to create RedirectStdOut token: %v", err)
	}
	tokens = append(tokens, tok)

	// output.txt
	tok, err = NewValueToken(CommandArgument, "output.txt")
	if err != nil {
		t.Fatalf("Failed to create CommandArgument token: %v", err)
	}
	tokens = append(tokens, tok)

	// Verify token count
	if len(tokens) != 7 {
		t.Errorf("Expected 7 tokens, got %d", len(tokens))
	}

	// Verify types
	expectedTypes := []TokenType{
		Command, CommandArgument, Pipe, Command, CommandArgument, RedirectStdOut, CommandArgument,
	}
	for i, expected := range expectedTypes {
		if tokens[i].Type != expected {
			t.Errorf("Token %d: expected type %v, got %v", i, expected, tokens[i].Type)
		}
	}

	// Verify values
	expectedValues := []string{"ls", "-la", "", "grep", "foo", "", "output.txt"}
	for i, expected := range expectedValues {
		if tokens[i].Value != expected {
			t.Errorf("Token %d: expected value %q, got %q", i, expected, tokens[i].Value)
		}
	}
}

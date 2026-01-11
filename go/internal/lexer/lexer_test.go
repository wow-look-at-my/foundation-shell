package lexer

import (
	"testing"
)

func TestTokenize_BasicWhitespaceSplitting(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []TokenContext
	}{
		{
			name:  "simple command with args",
			input: "echo hello world",
			expected: []TokenContext{
				{Content: "echo", WasSingleQuoted: false},
				{Content: "hello", WasSingleQuoted: false},
				{Content: "world", WasSingleQuoted: false},
			},
		},
		{
			name:  "multiple spaces between words",
			input: "echo    hello   world",
			expected: []TokenContext{
				{Content: "echo", WasSingleQuoted: false},
				{Content: "hello", WasSingleQuoted: false},
				{Content: "world", WasSingleQuoted: false},
			},
		},
		{
			name:  "tabs and spaces",
			input: "echo\thello\t\tworld",
			expected: []TokenContext{
				{Content: "echo", WasSingleQuoted: false},
				{Content: "hello", WasSingleQuoted: false},
				{Content: "world", WasSingleQuoted: false},
			},
		},
		{
			name:  "leading and trailing whitespace",
			input: "  echo hello  ",
			expected: []TokenContext{
				{Content: "echo", WasSingleQuoted: false},
				{Content: "hello", WasSingleQuoted: false},
			},
		},
		{
			name:     "single word",
			input:    "echo",
			expected: []TokenContext{{Content: "echo", WasSingleQuoted: false}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Tokenize(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertTokensEqual(t, tt.expected, result)
		})
	}
}

func TestTokenize_DoubleQuotes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []TokenContext
	}{
		{
			name:  "simple double quoted string",
			input: `echo "hello world"`,
			expected: []TokenContext{
				{Content: "echo", WasSingleQuoted: false},
				{Content: "hello world", WasSingleQuoted: false},
			},
		},
		{
			name:  "double quotes preserve multiple spaces",
			input: `echo "hello    world"`,
			expected: []TokenContext{
				{Content: "echo", WasSingleQuoted: false},
				{Content: "hello    world", WasSingleQuoted: false},
			},
		},
		{
			name:  "empty double quoted string",
			input: `echo ""`,
			expected: []TokenContext{
				{Content: "echo", WasSingleQuoted: false},
			},
		},
		{
			name:  "adjacent double quoted strings",
			input: `echo "hello""world"`,
			expected: []TokenContext{
				{Content: "echo", WasSingleQuoted: false},
				{Content: "helloworld", WasSingleQuoted: false},
			},
		},
		{
			name:  "double quotes with single quote inside",
			input: `echo "it's fine"`,
			expected: []TokenContext{
				{Content: "echo", WasSingleQuoted: false},
				{Content: "it's fine", WasSingleQuoted: false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Tokenize(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertTokensEqual(t, tt.expected, result)
		})
	}
}

func TestTokenize_SingleQuotes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []TokenContext
	}{
		{
			name:  "simple single quoted string",
			input: "echo 'hello world'",
			expected: []TokenContext{
				{Content: "echo", WasSingleQuoted: false},
				{Content: "hello world", WasSingleQuoted: true},
			},
		},
		{
			name:  "single quotes preserve everything literally",
			input: `echo 'hello $VAR \n world'`,
			expected: []TokenContext{
				{Content: "echo", WasSingleQuoted: false},
				{Content: `hello $VAR \n world`, WasSingleQuoted: true},
			},
		},
		{
			name:  "single quotes with double quote inside",
			input: `echo 'say "hello"'`,
			expected: []TokenContext{
				{Content: "echo", WasSingleQuoted: false},
				{Content: `say "hello"`, WasSingleQuoted: true},
			},
		},
		{
			name:  "single quotes preserve backslashes",
			input: `echo 'back\\slash'`,
			expected: []TokenContext{
				{Content: "echo", WasSingleQuoted: false},
				{Content: `back\\slash`, WasSingleQuoted: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Tokenize(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertTokensEqual(t, tt.expected, result)
		})
	}
}

func TestTokenize_Escapes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []TokenContext
	}{
		{
			name:  "escaped space joins words",
			input: `echo hello\ world`,
			expected: []TokenContext{
				{Content: "echo", WasSingleQuoted: false},
				{Content: "hello world", WasSingleQuoted: false},
			},
		},
		{
			name:  "escaped backslash",
			input: `echo hello\\world`,
			expected: []TokenContext{
				{Content: "echo", WasSingleQuoted: false},
				{Content: `hello\world`, WasSingleQuoted: false},
			},
		},
		{
			name:  "escaped dollar sign",
			input: `echo \$HOME`,
			expected: []TokenContext{
				{Content: "echo", WasSingleQuoted: false},
				{Content: string(EscapeMarker) + "$HOME", WasSingleQuoted: false},
			},
		},
		{
			name:  "escaped double quote",
			input: `echo \"hello\"`,
			expected: []TokenContext{
				{Content: "echo", WasSingleQuoted: false},
				{Content: `"hello"`, WasSingleQuoted: false},
			},
		},
		{
			name:  "escaped single quote",
			input: `echo \'hello\'`,
			expected: []TokenContext{
				{Content: "echo", WasSingleQuoted: false},
				{Content: `'hello'`, WasSingleQuoted: false},
			},
		},
		{
			name:  "escape newline sequence",
			input: `echo hello\nworld`,
			expected: []TokenContext{
				{Content: "echo", WasSingleQuoted: false},
				{Content: "hello\nworld", WasSingleQuoted: false},
			},
		},
		{
			name:  "escape tab sequence",
			input: `echo hello\tworld`,
			expected: []TokenContext{
				{Content: "echo", WasSingleQuoted: false},
				{Content: "hello\tworld", WasSingleQuoted: false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Tokenize(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertTokensEqual(t, tt.expected, result)
		})
	}
}

func TestTokenize_MixedQuotes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []TokenContext
	}{
		{
			name:  "escaped quote inside double quotes",
			input: `echo "it's a \"test\""`,
			expected: []TokenContext{
				{Content: "echo", WasSingleQuoted: false},
				{Content: `it's a "test"`, WasSingleQuoted: false},
			},
		},
		{
			name:  "mixed single and double quotes",
			input: `echo "hello" 'world'`,
			expected: []TokenContext{
				{Content: "echo", WasSingleQuoted: false},
				{Content: "hello", WasSingleQuoted: false},
				{Content: "world", WasSingleQuoted: true},
			},
		},
		{
			name:  "concatenated quoted strings",
			input: `echo "hello"'world'`,
			expected: []TokenContext{
				{Content: "echo", WasSingleQuoted: false},
				{Content: "helloworld", WasSingleQuoted: true},
			},
		},
		{
			name:  "unquoted and quoted concatenation",
			input: `echo hello"world"`,
			expected: []TokenContext{
				{Content: "echo", WasSingleQuoted: false},
				{Content: "helloworld", WasSingleQuoted: false},
			},
		},
		{
			name:  "complex mixed quoting",
			input: `echo 'single'"double"unquoted`,
			expected: []TokenContext{
				{Content: "echo", WasSingleQuoted: false},
				{Content: "singledoubleunquoted", WasSingleQuoted: true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Tokenize(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertTokensEqual(t, tt.expected, result)
		})
	}
}

func TestTokenize_EmptyAndEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []TokenContext
	}{
		{
			name:     "empty input",
			input:    "",
			expected: nil,
		},
		{
			name:     "only whitespace",
			input:    "   \t\n  ",
			expected: nil,
		},
		{
			name:  "only quotes with no content",
			input: `''`,
			// Empty single-quoted string produces no token since content is empty
			expected: nil,
		},
		{
			name:  "backslash at end of input",
			input: `echo hello\`,
			// Trailing backslash with nothing to escape is kept as literal backslash
			expected: []TokenContext{
				{Content: "echo", WasSingleQuoted: false},
				{Content: `hello\`, WasSingleQuoted: false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Tokenize(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertTokensEqual(t, tt.expected, result)
		})
	}
}

func TestTokenize_UnclosedQuotes(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectedErr string
	}{
		{
			name:        "unclosed single quote",
			input:       "echo 'hello",
			expectedErr: "unclosed single quote",
		},
		{
			name:        "unclosed double quote",
			input:       `echo "hello`,
			expectedErr: "unclosed double quote",
		},
		{
			name:        "unclosed single quote at start",
			input:       "'hello world",
			expectedErr: "unclosed single quote",
		},
		{
			name:        "unclosed double quote at start",
			input:       `"hello world`,
			expectedErr: "unclosed double quote",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Tokenize(tt.input)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if err.Error() != tt.expectedErr {
				t.Fatalf("expected error %q, got %q", tt.expectedErr, err.Error())
			}
		})
	}
}

func TestTokenize_EscapesInsideDoubleQuotes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []TokenContext
	}{
		{
			name:  "escaped quote inside double quotes",
			input: `echo "say \"hello\""`,
			expected: []TokenContext{
				{Content: "echo", WasSingleQuoted: false},
				{Content: `say "hello"`, WasSingleQuoted: false},
			},
		},
		{
			name:  "escaped backslash inside double quotes",
			input: `echo "path\\to\\file"`,
			expected: []TokenContext{
				{Content: "echo", WasSingleQuoted: false},
				{Content: `path\to\file`, WasSingleQuoted: false},
			},
		},
		{
			name:  "escaped dollar inside double quotes",
			input: `echo "cost is \$100"`,
			expected: []TokenContext{
				{Content: "echo", WasSingleQuoted: false},
				{Content: "cost is " + string(EscapeMarker) + "$100", WasSingleQuoted: false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Tokenize(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertTokensEqual(t, tt.expected, result)
		})
	}
}

func TestTokenize_RealWorldCommands(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []TokenContext
	}{
		{
			name:  "git commit command",
			input: `git commit -m "Fix bug in parser"`,
			expected: []TokenContext{
				{Content: "git", WasSingleQuoted: false},
				{Content: "commit", WasSingleQuoted: false},
				{Content: "-m", WasSingleQuoted: false},
				{Content: "Fix bug in parser", WasSingleQuoted: false},
			},
		},
		{
			name:  "grep with pattern",
			input: `grep -r 'func main' ./src`,
			expected: []TokenContext{
				{Content: "grep", WasSingleQuoted: false},
				{Content: "-r", WasSingleQuoted: false},
				{Content: "func main", WasSingleQuoted: true},
				{Content: "./src", WasSingleQuoted: false},
			},
		},
		{
			name:  "echo with variable",
			input: `echo "Hello $USER"`,
			expected: []TokenContext{
				{Content: "echo", WasSingleQuoted: false},
				{Content: "Hello $USER", WasSingleQuoted: false},
			},
		},
		{
			name:  "find command with complex pattern",
			input: `find . -name "*.go" -type f`,
			expected: []TokenContext{
				{Content: "find", WasSingleQuoted: false},
				{Content: ".", WasSingleQuoted: false},
				{Content: "-name", WasSingleQuoted: false},
				{Content: "*.go", WasSingleQuoted: false},
				{Content: "-type", WasSingleQuoted: false},
				{Content: "f", WasSingleQuoted: false},
			},
		},
		{
			name:  "path with spaces",
			input: `ls "/path/to/my files/"`,
			expected: []TokenContext{
				{Content: "ls", WasSingleQuoted: false},
				{Content: "/path/to/my files/", WasSingleQuoted: false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Tokenize(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertTokensEqual(t, tt.expected, result)
		})
	}
}

func TestStripEscapeMarkers(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no markers",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "single marker",
			input:    string(EscapeMarker) + "$HOME",
			expected: "$HOME",
		},
		{
			name:     "multiple markers",
			input:    string(EscapeMarker) + "$VAR1 and " + string(EscapeMarker) + "$VAR2",
			expected: "$VAR1 and $VAR2",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StripEscapeMarkers(tt.input)
			if result != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// assertTokensEqual compares two slices of TokenContext for equality.
func assertTokensEqual(t *testing.T, expected, actual []TokenContext) {
	t.Helper()

	if len(expected) != len(actual) {
		t.Fatalf("token count mismatch: expected %d, got %d\nexpected: %+v\nactual: %+v",
			len(expected), len(actual), expected, actual)
	}

	for i := range expected {
		if expected[i].Content != actual[i].Content {
			t.Errorf("token[%d] content mismatch: expected %q, got %q",
				i, expected[i].Content, actual[i].Content)
		}
		if expected[i].WasSingleQuoted != actual[i].WasSingleQuoted {
			t.Errorf("token[%d] WasSingleQuoted mismatch: expected %v, got %v",
				i, expected[i].WasSingleQuoted, actual[i].WasSingleQuoted)
		}
	}
}

package parser

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Item: tilde expansion is suppressed for any quoted word; unquoted tildes
// still expand.
func TestParse_TildeQuotingSuppression(t *testing.T) {
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", "/home/testuser")
	defer os.Setenv("HOME", oldHome)

	tests := []struct {
		name  string
		input string
		arg   string
	}{
		{"bare tilde expands", "echo ~", "/home/testuser"},
		{"tilde path expands", "echo ~/x", "/home/testuser/x"},
		{"double-quoted tilde literal", `echo "~"`, "~"},
		{"single-quoted tilde literal", `echo '~'`, "~"},
		{"double-quoted tilde path literal", `echo "~/x"`, "~/x"},
		{"partially quoted word literal", `echo ~"/x"`, "~/x"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chain, err := Parse(tt.input)
			require.NoError(t, err)
			require.Len(t, chain.Commands, 1)
			require.Len(t, chain.Commands[0].Args, 2)
			assert.Equal(t, tt.arg, chain.Commands[0].Args[1])
		})
	}
}

// Double-quoted words must still expand variables even though tilde
// expansion is suppressed for them.
func TestParse_DoubleQuotedStillExpandsVariables(t *testing.T) {
	os.Setenv("PARSER_TEST_NAME", "world")
	defer os.Unsetenv("PARSER_TEST_NAME")

	chain, err := Parse(`echo "~ $PARSER_TEST_NAME"`)
	require.NoError(t, err)
	assert.Equal(t, "~ world", chain.Commands[0].Args[1])
}

// Item: escape-marker stripping is unconditional for every value token,
// single-quoted included. A mixed word like 'a'\$HOME is WasSingleQuoted
// (no expansion) but still carries a marker from the unquoted \$.
func TestParse_EscapeMarkerStrippingIsUnconditional(t *testing.T) {
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", "/home/testuser")
	defer os.Setenv("HOME", oldHome)

	tests := []struct {
		name  string
		input string
		arg   string
	}{
		// Mixed single-quoted + escaped-dollar word: expansion suppressed,
		// marker must still be stripped.
		{"single-quoted prefix with escaped dollar", `echo 'a'\$HOME`, "a$HOME"},
		// Escapes INSIDE single quotes are raw content (no markers arise):
		// the backslash survives literally.
		{"backslash dollar inside single quotes", `echo '\$HOME'`, `\$HOME`},
		// Plain escaped dollar outside quotes.
		{"escaped dollar unquoted", `echo \$HOME`, "$HOME"},
		// Escaped dollar inside double quotes.
		{"escaped dollar double-quoted", `echo "\$HOME"`, "$HOME"},
		// Escaped backtick stays a literal backtick character.
		{"escaped backticks unquoted", "echo \\`x\\`", "`x`"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chain, err := Parse(tt.input)
			require.NoError(t, err)
			require.Len(t, chain.Commands, 1)
			require.Len(t, chain.Commands[0].Args, 2)
			assert.Equal(t, tt.arg, chain.Commands[0].Args[1])
		})
	}
}

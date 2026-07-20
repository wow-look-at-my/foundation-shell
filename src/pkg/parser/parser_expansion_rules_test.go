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

// Item: $? expands to the injected last exit status; the back-compat
// wrappers default it to 0. Other special parameters stay literal.
func TestParse_LastStatusExpansion(t *testing.T) {
	status := func(n int) func() int { return func() int { return n } }

	tests := []struct {
		name  string
		input string
		opts  Options
		arg   string
	}{
		{"bare status", "echo $?", Options{LastStatus: status(7)}, "7"},
		{"status zero", "echo $?", Options{LastStatus: status(0)}, "0"},
		{"large status", "echo $?", Options{LastStatus: status(127)}, "127"},
		{"nil status defaults to 0", "echo $?", Options{}, "0"},
		{"status in word", "echo rc=$?.", Options{LastStatus: status(3)}, "rc=3."},
		{"double-quoted status expands", `echo "$?"`, Options{LastStatus: status(9)}, "9"},
		{"single-quoted status suppressed", `echo '$?'`, Options{LastStatus: status(9)}, "$?"},
		{"escaped status suppressed", `echo \$?`, Options{LastStatus: status(9)}, "$?"},
		{"dollar dollar stays literal", "echo $$", Options{LastStatus: status(9)}, "$$"},
		{"dollar bang stays literal", "echo $!", Options{LastStatus: status(9)}, "$!"},
		{"positional stays literal", "echo $1", Options{LastStatus: status(9)}, "$1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chain, err := ParseWithOptions(tt.input, tt.opts)
			require.NoError(t, err)
			require.Len(t, chain.Commands, 1)
			require.Len(t, chain.Commands[0].Args, 2)
			assert.Equal(t, tt.arg, chain.Commands[0].Args[1])
		})
	}
}

// The Parse wrapper also defaults $? to 0.
func TestParse_WrapperDefaultsStatusToZero(t *testing.T) {
	chain, err := Parse("echo $?")
	require.NoError(t, err)
	assert.Equal(t, "0", chain.Commands[0].Args[1])
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

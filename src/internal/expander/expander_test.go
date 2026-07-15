package expander

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExpandTilde(t *testing.T) {
	// Save and restore HOME
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	os.Setenv("HOME", "/home/testuser")

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", ""},
		{"just tilde", "~", "/home/testuser"},
		{"tilde with slash", "~/", "/home/testuser/"},
		{"tilde with path", "~/docs", "/home/testuser/docs"},
		{"tilde with deep path", "~/a/b/c", "/home/testuser/a/b/c"},
		{"tilde user not supported", "~user", "~user"},
		{"tilde user with path", "~user/docs", "~user/docs"},
		{"no tilde", "/home/other", "/home/other"},
		{"tilde in middle", "/path/~", "/path/~"},
		{"tilde with dots", "~/.config", "/home/testuser/.config"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExpandTilde(tt.input)
			assert.Equal(t, tt.expected, result)

		})
	}
}

func TestExpandTildeNoHome(t *testing.T) {
	// Save and restore HOME
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	os.Unsetenv("HOME")

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"tilde without HOME", "~", "~"},
		{"tilde path without HOME", "~/docs", "~/docs"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExpandTilde(tt.input)
			assert.Equal(t, tt.expected, result)

		})
	}
}

func TestExpandEnvironment(t *testing.T) {
	// Save and restore test vars
	originalTestVar := os.Getenv("TEST_VAR")
	originalTestVar2 := os.Getenv("TEST_VAR2")
	defer func() {
		if originalTestVar == "" {
			os.Unsetenv("TEST_VAR")
		} else {
			os.Setenv("TEST_VAR", originalTestVar)
		}
		if originalTestVar2 == "" {
			os.Unsetenv("TEST_VAR2")
		} else {
			os.Setenv("TEST_VAR2", originalTestVar2)
		}
	}()

	os.Setenv("TEST_VAR", "hello")
	os.Setenv("TEST_VAR2", "world")

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", ""},
		{"no variables", "hello world", "hello world"},
		{"simple var", "$TEST_VAR", "hello"},
		{"var with text after", "$TEST_VAR!", "hello!"},
		{"var in middle", "say $TEST_VAR please", "say hello please"},
		{"multiple vars", "$TEST_VAR $TEST_VAR2", "hello world"},
		{"braced var", "${TEST_VAR}", "hello"},
		{"braced var with text", "${TEST_VAR}suffix", "hellosuffix"},
		{"nonexistent var", "$NONEXISTENT_VAR_12345", ""},
		{"nonexistent braced var", "${NONEXISTENT_VAR_12345}", ""},
		{"dollar at end", "test$", "test$"},
		{"double dollar", "$$", "$$"},
		{"dollar with number", "$123", "$123"},
		{"underscore var", "$_VAR", ""}, // _VAR doesn't exist
		{"var starting with underscore", "$_TEST", ""},
		{"empty braces", "${}", "${}"},
		{"unclosed brace", "${TEST_VAR", "${TEST_VAR"},
		{"dollar space", "$ TEST_VAR", "$ TEST_VAR"},
		{"adjacent vars", "$TEST_VAR$TEST_VAR2", "helloworld"},
		{"braced adjacent vars", "${TEST_VAR}${TEST_VAR2}", "helloworld"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExpandEnvironment(tt.input)
			assert.Equal(t, tt.expected, result)

		})
	}
}

func TestExpandEnvironmentEscaped(t *testing.T) {
	// Save and restore HOME
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	os.Setenv("HOME", "/home/testuser")

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"escaped dollar", "\x01$HOME", "$HOME"},
		{"escaped dollar in middle", "path/\x01$HOME/file", "path/$HOME/file"},
		{"escaped and unescaped", "\x01$HOME and $HOME", "$HOME and /home/testuser"},
		{"multiple escaped", "\x01$ONE\x01$TWO", "$ONE$TWO"},
		{"escape marker alone", "\x01", "\x01"},
		{"escape marker not before dollar", "\x01X", "\x01X"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExpandEnvironment(tt.input)
			assert.Equal(t, tt.expected, result)

		})
	}
}

func TestExpand(t *testing.T) {
	// Save and restore HOME
	originalHome := os.Getenv("HOME")
	originalTestVar := os.Getenv("TEST_VAR")
	defer func() {
		os.Setenv("HOME", originalHome)
		if originalTestVar == "" {
			os.Unsetenv("TEST_VAR")
		} else {
			os.Setenv("TEST_VAR", originalTestVar)
		}
	}()

	os.Setenv("HOME", "/home/testuser")
	os.Setenv("TEST_VAR", "value")

	tests := []struct {
		name            string
		input           string
		wasSingleQuoted bool
		expected        string
	}{
		{"tilde expansion", "~", false, "/home/testuser"},
		{"env expansion", "$TEST_VAR", false, "value"},
		{"both expansions", "~/$TEST_VAR", false, "/home/testuser/value"},
		{"single quoted tilde", "~", true, "~"},
		{"single quoted env", "$TEST_VAR", true, "$TEST_VAR"},
		{"single quoted both", "~/$TEST_VAR", true, "~/$TEST_VAR"},
		{"empty string", "", false, ""},
		{"empty single quoted", "", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Expand(tt.input, tt.wasSingleQuoted)
			assert.Equal(t, tt.expected, result)

		})
	}
}

func TestExpandWithRealHome(t *testing.T) {
	// Test with actual HOME value
	home := os.Getenv("HOME")
	if home == "" {
		t.Skip("HOME not set")
	}

	result := ExpandTilde("~")
	assert.Equal(t, home, result)

	result = ExpandTilde("~/test")
	expected := home + "/test"
	assert.Equal(t, expected, result)

}

func TestExpandEnvironmentSpecialCases(t *testing.T) {
	os.Setenv("A", "a")
	os.Setenv("AB", "ab")
	os.Setenv("ABC", "abc")
	defer func() {
		os.Unsetenv("A")
		os.Unsetenv("AB")
		os.Unsetenv("ABC")
	}()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"greedy matching", "$ABC", "abc"},
		{"braced short var", "${A}BC", "aBC"},
		// $A1 expands variable A1 (not A followed by 1), which is empty
		{"var name with digit", "$A1", ""},
		// Use braces to get A followed by literal 1
		{"braced var with literal suffix", "${A}1", "a1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExpandEnvironment(tt.input)
			assert.Equal(t, tt.expected, result)

		})
	}
}

func TestIsVarChar(t *testing.T) {
	// Test internal helper function behavior through ExpandEnvironment
	os.Setenv("VAR_123", "test")
	os.Setenv("_START", "underscore")
	os.Setenv("A", "a")
	defer func() {
		os.Unsetenv("VAR_123")
		os.Unsetenv("_START")
		os.Unsetenv("A")
	}()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"var with underscore", "$VAR_123", "test"},
		{"var starting underscore", "$_START", "underscore"},
		// Variable names are ASCII-only: a multibyte UTF-8 sequence ends
		// the name instead of being pulled in byte by byte.
		{"multibyte rune ends the name", "$Aé", "aé"},
		{"multibyte rune after braced var", "${A}é", "aé"},
		{"cjk after var name", "$A漢", "a漢"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExpandEnvironment(tt.input)
			assert.Equal(t, tt.expected, result)

		})
	}
}

// Command-substitution expansion (ExpandToken) is tested in
// expander_token_test.go.

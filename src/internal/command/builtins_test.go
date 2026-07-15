package command

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsBuiltin(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{"cd", true},
		{"pwd", true},
		{"exit", true},
		{"clear", true},
		{"help", true},
		{"export", true},
		{"ls", false},
		{"cat", false},
		{"grep", false},
		{"echo", false}, // echo is external, not a builtin
		{"", false},
		{"CD", false}, // case sensitive
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsBuiltin(tt.name)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestBuiltinCd(t *testing.T) {
	// Save original directory to restore after test
	originalDir, err := os.Getwd()
	require.Nil(t, err)

	defer os.Chdir(originalDir)

	t.Run("change to existing directory", func(t *testing.T) {
		// Resolve symlinks because macOS has /var -> /private/var
		tempDir, _ := filepath.EvalSymlinks(t.TempDir())

		var stdout, stderr bytes.Buffer
		code, err := ExecuteBuiltin("cd", []string{tempDir}, nil, &stdout, &stderr)
		require.Nil(t, err)
		assert.Equal(t, 0, code)

		cwd, _ := os.Getwd()
		assert.Equal(t, tempDir, cwd)
	})

	t.Run("change to nonexistent directory", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code, err := ExecuteBuiltin("cd", []string{"/nonexistent/directory/path"}, nil, &stdout, &stderr)
		// Ordinary builtin failure: status 1, prefix-free message with the
		// bare os reason, no error.
		require.Nil(t, err)
		assert.Equal(t, 1, code)
		assert.Equal(t, "cd: /nonexistent/directory/path: no such file or directory\n", stderr.String())
	})

	t.Run("cd with no args goes to HOME", func(t *testing.T) {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			t.Skip("could not determine home directory")
		}

		var stdout, stderr bytes.Buffer
		code, err := ExecuteBuiltin("cd", []string{}, nil, &stdout, &stderr)
		require.Nil(t, err)
		assert.Equal(t, 0, code)

		cwd, _ := os.Getwd()
		assert.Equal(t, homeDir, cwd)
	})
}

func TestBuiltinPwd(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code, err := ExecuteBuiltin("pwd", []string{}, nil, &stdout, &stderr)
	require.Nil(t, err)
	assert.Equal(t, 0, code)

	expectedCwd, _ := os.Getwd()
	output := strings.TrimSpace(stdout.String())
	assert.Equal(t, expectedCwd, output)
}

func TestBuiltinClear(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code, err := ExecuteBuiltin("clear", []string{}, nil, &stdout, &stderr)
	require.Nil(t, err)
	assert.Equal(t, 0, code)

	expected := "\033[2J\033[H"
	assert.Equal(t, expected, stdout.String())
}

func TestBuiltinHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code, err := ExecuteBuiltin("help", []string{}, nil, &stdout, &stderr)
	require.Nil(t, err)
	assert.Equal(t, 0, code)

	output := stdout.String()
	assert.NotEqual(t, 0, len(output))

	// Every builtin, standalone assignment, the real operators, the
	// redirections, comments, and $? must all be documented.
	expectedKeywords := []string{
		"cd", "pwd", "exit", "clear", "help", "export", "NAME=VALUE",
		"Foundation Shell",
		"|", "&&", "||", ";",
		"< file", "> file", ">> file", "2> file", "2>> file",
		"#", "$?",
	}
	for _, keyword := range expectedKeywords {
		assert.Contains(t, output, keyword)
	}

	// No unimplemented features: there is no & background operator. Strip
	// ANSI-colored "&&" mentions first by checking no line advertises a
	// bare "&".
	assert.NotContains(t, output, "background")
}

func TestExecuteBuiltinNonexistent(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code, err := ExecuteBuiltin("nonexistent", []string{}, nil, &stdout, &stderr)
	assert.NotNil(t, err)
	assert.Equal(t, 1, code)
	assert.Contains(t, err.Error(), "not a builtin")
}

func TestBuiltinsMapCompleteness(t *testing.T) {
	expectedBuiltins := []string{"cd", "pwd", "exit", "clear", "help", "export"}

	for _, name := range expectedBuiltins {
		_, ok := Builtins[name]
		assert.True(t, ok)
	}

	// Verify expected count: exactly these six, nothing else.
	assert.Equal(t, len(expectedBuiltins), len(Builtins))
}

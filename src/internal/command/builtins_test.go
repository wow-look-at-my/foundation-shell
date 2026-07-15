package command

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
		{"ls", false},
		{"cat", false},
		{"grep", false},
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
		err := ExecuteBuiltin("cd", []string{tempDir}, nil, &stdout, &stderr)
		assert.Nil(t, err)

		cwd, _ := os.Getwd()
		assert.Equal(t, tempDir, cwd)

	})

	t.Run("change to nonexistent directory", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		err := ExecuteBuiltin("cd", []string{"/nonexistent/directory/path"}, nil, &stdout, &stderr)
		assert.NotNil(t, err)

	})

	t.Run("cd with no args goes to HOME", func(t *testing.T) {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			t.Skip("could not determine home directory")
		}

		var stdout, stderr bytes.Buffer
		err = ExecuteBuiltin("cd", []string{}, nil, &stdout, &stderr)
		assert.Nil(t, err)

		cwd, _ := os.Getwd()
		assert.Equal(t, homeDir, cwd)

	})
}

func TestBuiltinPwd(t *testing.T) {
	var stdout, stderr bytes.Buffer

	err := ExecuteBuiltin("pwd", []string{}, nil, &stdout, &stderr)
	assert.Nil(t, err)

	expectedCwd, _ := os.Getwd()
	output := strings.TrimSpace(stdout.String())
	assert.Equal(t, expectedCwd, output)

}

func TestBuiltinClear(t *testing.T) {
	var stdout, stderr bytes.Buffer

	err := ExecuteBuiltin("clear", []string{}, nil, &stdout, &stderr)
	assert.Nil(t, err)

	expected := "\033[2J\033[H"
	assert.Equal(t, expected, stdout.String())

}

func TestBuiltinHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer

	err := ExecuteBuiltin("help", []string{}, nil, &stdout, &stderr)
	assert.Nil(t, err)

	output := stdout.String()
	assert.NotEqual(t, 0, len(output))

	// Check that help mentions key commands
	expectedKeywords := []string{"cd", "pwd", "exit", "clear", "help", "Foundation Shell"}
	for _, keyword := range expectedKeywords {
		assert.Contains(t, output, keyword)

	}
}

func TestBuiltinExitIsRecognized(t *testing.T) {
	// We can't easily test exit because it calls os.Exit(),
	// but we can verify it's recognized as a builtin
	assert.True(t, IsBuiltin("exit"))

	// Verify the function exists in the map
	fn, ok := Builtins["exit"]
	assert.True(t, ok)

	assert.NotNil(t, fn)

}

func TestExecuteBuiltinNonexistent(t *testing.T) {
	var stdout, stderr bytes.Buffer

	err := ExecuteBuiltin("nonexistent", []string{}, nil, &stdout, &stderr)
	assert.NotNil(t, err)

	assert.Contains(t, err.Error(), "not a builtin")

}

func TestBuiltinsMapCompleteness(t *testing.T) {
	expectedBuiltins := []string{"cd", "pwd", "exit", "clear", "help"}

	for _, name := range expectedBuiltins {
		_, ok := Builtins[name]
		assert.True(t, ok)

	}

	// Verify expected count
	assert.Equal(t, len(expectedBuiltins), len(Builtins))

}

package command

import (
	"bytes"
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
			if got := IsBuiltin(tt.name); got != tt.expected {
				t.Errorf("IsBuiltin(%q) = %v, want %v", tt.name, got, tt.expected)
			}
		})
	}
}

func TestBuiltinCd(t *testing.T) {
	// Save original directory to restore after test
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	defer os.Chdir(originalDir)

	t.Run("change to existing directory", func(t *testing.T) {
		// Resolve symlinks because macOS has /var -> /private/var
		tempDir, _ := filepath.EvalSymlinks(t.TempDir())

		var stdout, stderr bytes.Buffer
		err := ExecuteBuiltin("cd", []string{tempDir}, nil, &stdout, &stderr)
		if err != nil {
			t.Errorf("cd to existing directory failed: %v", err)
		}

		cwd, _ := os.Getwd()
		if cwd != tempDir {
			t.Errorf("expected cwd to be %q, got %q", tempDir, cwd)
		}
	})

	t.Run("change to nonexistent directory", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		err := ExecuteBuiltin("cd", []string{"/nonexistent/directory/path"}, nil, &stdout, &stderr)
		if err == nil {
			t.Error("expected error for nonexistent directory, got nil")
		}
	})

	t.Run("cd with no args goes to HOME", func(t *testing.T) {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			t.Skip("could not determine home directory")
		}

		var stdout, stderr bytes.Buffer
		err = ExecuteBuiltin("cd", []string{}, nil, &stdout, &stderr)
		if err != nil {
			t.Errorf("cd with no args failed: %v", err)
		}

		cwd, _ := os.Getwd()
		if cwd != homeDir {
			t.Errorf("expected cwd to be %q, got %q", homeDir, cwd)
		}
	})
}

func TestBuiltinPwd(t *testing.T) {
	var stdout, stderr bytes.Buffer

	err := ExecuteBuiltin("pwd", []string{}, nil, &stdout, &stderr)
	if err != nil {
		t.Errorf("pwd failed: %v", err)
	}

	expectedCwd, _ := os.Getwd()
	output := strings.TrimSpace(stdout.String())
	if output != expectedCwd {
		t.Errorf("pwd output = %q, want %q", output, expectedCwd)
	}
}

func TestBuiltinClear(t *testing.T) {
	var stdout, stderr bytes.Buffer

	err := ExecuteBuiltin("clear", []string{}, nil, &stdout, &stderr)
	if err != nil {
		t.Errorf("clear failed: %v", err)
	}

	expected := "\033[2J\033[H"
	if stdout.String() != expected {
		t.Errorf("clear output = %q, want %q", stdout.String(), expected)
	}
}

func TestBuiltinHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer

	err := ExecuteBuiltin("help", []string{}, nil, &stdout, &stderr)
	if err != nil {
		t.Errorf("help failed: %v", err)
	}

	output := stdout.String()
	if len(output) == 0 {
		t.Error("help output is empty")
	}

	// Check that help mentions key commands
	expectedKeywords := []string{"cd", "pwd", "exit", "clear", "help", "Foundation Shell"}
	for _, keyword := range expectedKeywords {
		if !strings.Contains(output, keyword) {
			t.Errorf("help output missing expected keyword %q", keyword)
		}
	}
}

func TestBuiltinExitIsRecognized(t *testing.T) {
	// We can't easily test exit because it calls os.Exit(),
	// but we can verify it's recognized as a builtin
	if !IsBuiltin("exit") {
		t.Error("exit should be recognized as a builtin")
	}

	// Verify the function exists in the map
	fn, ok := Builtins["exit"]
	if !ok {
		t.Error("exit function should exist in Builtins map")
	}
	if fn == nil {
		t.Error("exit function should not be nil")
	}
}

func TestExecuteBuiltinNonexistent(t *testing.T) {
	var stdout, stderr bytes.Buffer

	err := ExecuteBuiltin("nonexistent", []string{}, nil, &stdout, &stderr)
	if err == nil {
		t.Error("expected error for nonexistent builtin, got nil")
	}
	if !strings.Contains(err.Error(), "not a builtin") {
		t.Errorf("error message should mention 'not a builtin', got: %v", err)
	}
}

func TestBuiltinsMapCompleteness(t *testing.T) {
	expectedBuiltins := []string{"cd", "pwd", "exit", "clear", "help"}

	for _, name := range expectedBuiltins {
		if _, ok := Builtins[name]; !ok {
			t.Errorf("expected builtin %q not found in Builtins map", name)
		}
	}

	// Verify expected count
	if len(Builtins) != len(expectedBuiltins) {
		t.Errorf("expected %d builtins, got %d", len(expectedBuiltins), len(Builtins))
	}
}

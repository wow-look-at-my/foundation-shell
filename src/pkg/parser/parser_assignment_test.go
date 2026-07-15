package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Standalone assignment recognition (execution.md §Standalone Assignment):
// sole word, not single-quoted anywhere, expanded value matches NAME=VALUE.
func TestParse_AssignmentMarking(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		isAssignment bool
		arg          string // expected Args[0]
	}{
		{"plain", "A=hello", true, "A=hello"},
		{"empty value", "A=", true, "A="},
		{"value contains equals", "A=B=C", true, "A=B=C"},
		{"underscore name", "_A1=x", true, "_A1=x"},
		{"double-quoted value", `A="B"`, true, "A=B"},
		{"whole word double-quoted", `"A=B"`, true, "A=B"},
		{"quoted value with space", `A="hello world"`, true, "A=hello world"},
		{"single-quoted whole word", `'A=B'`, false, "A=B"},
		{"single-quoted value", `A='B'`, false, "A=B"},
		{"single-quoted name", `'A'=B`, false, "A=B"},
		{"invalid name digit", "1X=y", false, "1X=y"},
		{"invalid name dash", "A-B=z", false, "A-B=z"},
		{"no equals", "AB", false, "AB"},
		{"leading equals", "=x", false, "=x"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chain, err := Parse(tt.input)
			require.NoError(t, err)
			require.Len(t, chain.Commands, 1)
			cmd := chain.Commands[0]
			require.Len(t, cmd.Args, 1)
			assert.Equal(t, tt.arg, cmd.Args[0])
			assert.Equal(t, tt.isAssignment, cmd.IsAssignment,
				"IsAssignment for %q", tt.input)
		})
	}
}

// More than one word is never an assignment (prefix assignments are
// unsupported), and only the marked command in a chain is an assignment.
func TestParse_AssignmentRequiresSoleWord(t *testing.T) {
	chain, err := Parse("A=x cmd")
	require.NoError(t, err)
	require.Len(t, chain.Commands, 1)
	assert.False(t, chain.Commands[0].IsAssignment)
	assert.Equal(t, []string{"A=x", "cmd"}, chain.Commands[0].Args)

	chain, err = Parse("echo a=b")
	require.NoError(t, err)
	assert.False(t, chain.Commands[0].IsAssignment)
}

// Redirections do not count as words: an assignment with a redirection is
// still an assignment.
func TestParse_AssignmentWithRedirection(t *testing.T) {
	chain, err := Parse("A=b > /tmp/somewhere.txt")
	require.NoError(t, err)
	require.Len(t, chain.Commands, 1)
	cmd := chain.Commands[0]
	assert.True(t, cmd.IsAssignment)
	assert.Equal(t, "/tmp/somewhere.txt", cmd.OutputFile)
}

// Assignments are recognized per command within a chain.
func TestParse_AssignmentInChain(t *testing.T) {
	chain, err := Parse("A=1 ; echo done ; B=2")
	require.NoError(t, err)
	require.Len(t, chain.Commands, 3)
	assert.True(t, chain.Commands[0].IsAssignment)
	assert.False(t, chain.Commands[1].IsAssignment)
	assert.True(t, chain.Commands[2].IsAssignment)
}

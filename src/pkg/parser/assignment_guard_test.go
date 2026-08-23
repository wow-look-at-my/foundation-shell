package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The whole input expands in one pass before anything runs, so a variable
// assigned in it still holds its pre-input value everywhere in it. Left
// alone, `OUT=$(cmd); echo $OUT` prints an empty line and reports success.
func TestParse_AssignmentThenUseRejected(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"standalone then echo", "X=hi; echo $X"},
		{"capture then use", "OUT=$(echo captured); echo $OUT"},
		{"export then use", "export X=hi; echo $X"},
		{"braced reference", "X=hi; echo ${X}"},
		{"reference inside double quotes", `X=hi; echo "value is $X"`},
		{"reference in a later pipeline", "X=hi; echo $X | cat"},
		{"reference after &&", "X=hi && echo $X"},
		{"reference in a redirection target", "X=out; echo hi > $X"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.input)
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrAssignmentThenUse)
			assert.Contains(t, err.Error(), "variable is assigned and used in the same input")
		})
	}
}

// The guard names the variable, so a caller knows which one to split out.
func TestParse_AssignmentThenUseNamesTheVariable(t *testing.T) {
	_, err := Parse("BUILD_DIR=/tmp/x; echo $BUILD_DIR")
	require.Error(t, err)
	assert.Equal(t, "variable is assigned and used in the same input: BUILD_DIR", err.Error())
}

// Everything that does NOT read the stale value must still parse.
func TestParse_AssignmentWithoutLaterUseIsFine(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"assignment alone", "X=hi"},
		{"assignment then unrelated command", "X=hi; echo done"},
		{"assignment then a different variable", "X=hi; echo $Y"},
		{"child reads the environment itself", "X=hi; printenv X"},
		// The value really is set, so a single-quoted $X handed to a child
		// resolves in the child. This is the documented way to use it.
		{"single-quoted reference is data", `X=hi; sh -c 'echo $X'`},
		{"escaped reference is literal", `X=hi; echo \$X`},
		{"self reference in the assignment", "X=$X"},
		{"use before assign", "echo $X; X=hi"},
		{"reference with no assignment anywhere", "echo $HOME"},
		{"assignment-looking argument is not an assignment", "echo A=1 $A"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.input)
			assert.NoError(t, err)
		})
	}
}

// A prefix assignment is not an assignment here (two words make it a command
// name), so it must not arm the guard either.
func TestParse_PrefixAssignmentDoesNotArmTheGuard(t *testing.T) {
	_, err := Parse("X=hi true; echo $X")
	assert.NoError(t, err)
}

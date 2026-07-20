package command

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// withLastStatus stubs the shell's last-status source for the duration of a
// test and restores it afterwards.
func withLastStatus(t *testing.T, v int) {
	t.Helper()
	old := LastStatus
	LastStatus = func() int { return v }
	t.Cleanup(func() { LastStatus = old })
}

// exit returns the ErrExit sentinel — never os.Exit — so pipelines,
// substitutions, and the REPL can each interpret it.
func TestBuiltinExit_SentinelWithCode(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code, err := ExecuteBuiltin("exit", []string{"7"}, nil, &stdout, &stderr)

	assert.Equal(t, 7, code)
	require.IsType(t, ErrExit{}, err)
	assert.Equal(t, 7, err.(ErrExit).Code)
	assert.Empty(t, stderr.String())
}

// No argument: exit with the shell's last recorded status.
func TestBuiltinExit_NoArgUsesLastStatus(t *testing.T) {
	withLastStatus(t, 42)

	var stdout, stderr bytes.Buffer
	code, err := ExecuteBuiltin("exit", nil, nil, &stdout, &stderr)

	assert.Equal(t, 42, code)
	require.IsType(t, ErrExit{}, err)
	assert.Equal(t, 42, err.(ErrExit).Code)
}

// No argument with no wired status source: 0.
func TestBuiltinExit_NoArgDefaultsToZero(t *testing.T) {
	withLastStatus(t, 0)

	var stdout, stderr bytes.Buffer
	code, err := ExecuteBuiltin("exit", nil, nil, &stdout, &stderr)
	assert.Equal(t, 0, code)
	require.IsType(t, ErrExit{}, err)
}

// Numeric arguments are taken modulo 256, non-negative (spec examples).
func TestBuiltinExit_Modulo256(t *testing.T) {
	tests := []struct {
		arg  string
		want int
	}{
		{"0", 0},
		{"7", 7},
		{"255", 255},
		{"256", 0},
		{"-1", 255},
		{"300", 44},
	}
	for _, tt := range tests {
		t.Run(tt.arg, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code, err := ExecuteBuiltin("exit", []string{tt.arg}, nil, &stdout, &stderr)
			assert.Equal(t, tt.want, code)
			require.IsType(t, ErrExit{}, err)
			assert.Equal(t, tt.want, err.(ErrExit).Code)
		})
	}
}

// A non-numeric argument is a usage error: status 2, canonical message, NO
// sentinel — the shell keeps running.
func TestBuiltinExit_NonNumericDoesNotExit(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code, err := ExecuteBuiltin("exit", []string{"abc"}, nil, &stdout, &stderr)

	assert.Equal(t, 2, code)
	assert.Nil(t, err)
	assert.Equal(t, "exit: abc: numeric argument required\n", stderr.String())
}

// Extra arguments are ignored: only the first is examined.
func TestBuiltinExit_ExtraArgsIgnored(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code, err := ExecuteBuiltin("exit", []string{"3", "junk", "more"}, nil, &stdout, &stderr)
	assert.Equal(t, 3, code)
	require.IsType(t, ErrExit{}, err)
}

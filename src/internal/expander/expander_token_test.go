package expander

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockExecutor implements SubshellExecutor for testing.
type mockExecutor struct {
	outputs map[string]string
	calls   []string
	err     error
}

func newMockExecutor() *mockExecutor {
	return &mockExecutor{
		outputs: make(map[string]string),
		calls:   make([]string, 0),
	}
}

func (m *mockExecutor) Execute(command string) (string, int, error) {
	m.calls = append(m.calls, command)
	if m.err != nil {
		return "", 1, m.err
	}
	if output, ok := m.outputs[command]; ok {
		return output, 0, nil
	}
	return "", 0, nil
}

func TestExpandToken_Substitution(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		outputs  map[string]string
		expected string
		// calls, when non-nil, asserts the exact bodies handed to the
		// executor (in order).
		calls []string
	}{
		{
			name:     "no substitution",
			input:    "hello world",
			outputs:  map[string]string{},
			expected: "hello world",
			calls:    []string{},
		},
		{
			name:     "simple dollar paren",
			input:    "$(whoami)",
			outputs:  map[string]string{"whoami": "testuser\n"},
			expected: "testuser",
		},
		{
			name:     "dollar paren with surrounding text",
			input:    "hello $(whoami) there",
			outputs:  map[string]string{"whoami": "testuser\n"},
			expected: "hello testuser there",
		},
		{
			name:     "simple backtick",
			input:    "`whoami`",
			outputs:  map[string]string{"whoami": "testuser\n"},
			expected: "testuser",
		},
		{
			name:     "multiple substitutions",
			input:    "$(cmd1) and $(cmd2)",
			outputs:  map[string]string{"cmd1": "one\n", "cmd2": "two\n"},
			expected: "one and two",
			calls:    []string{"cmd1", "cmd2"},
		},
		{
			// A nested substitution is ONE top-level span: the whole body
			// goes to the executor, which recurses through the parser. The
			// expander itself never peels the inner $().
			name:     "nested dollar paren goes to the executor whole",
			input:    "$(echo $(whoami))",
			outputs:  map[string]string{"echo $(whoami)": "testuser\n"},
			expected: "testuser",
			calls:    []string{"echo $(whoami)"},
		},
		{
			name:     "trailing newlines stripped",
			input:    "$(cmd)",
			outputs:  map[string]string{"cmd": "output\n\n\n"},
			expected: "output",
		},
		{
			name:     "empty output",
			input:    "$(cmd)",
			outputs:  map[string]string{"cmd": ""},
			expected: "",
		},
		{
			name:     "mixed dollar and backtick",
			input:    "$(cmd1) and `cmd2`",
			outputs:  map[string]string{"cmd1": "one\n", "cmd2": "two\n"},
			expected: "one and two",
		},
		{
			// SECURITY: substitutions appearing in a command's OUTPUT are
			// data, never executed.
			name:     "output containing substitution syntax stays literal",
			input:    "$(cat payload)",
			outputs:  map[string]string{"cat payload": "$(echo pwned)\n"},
			expected: "$(echo pwned)",
			calls:    []string{"cat payload"},
		},
		{
			name:     "output containing backticks stays literal",
			input:    "$(cat payload)",
			outputs:  map[string]string{"cat payload": "`pwned`\n"},
			expected: "`pwned`",
			calls:    []string{"cat payload"},
		},
		{
			// Quoting inside the body is the body's business: the span scan
			// must not end at a quoted ).
			name:     "quoted close paren stays in the body",
			input:    "$(echo ')')",
			outputs:  map[string]string{"echo ')'": ")\n"},
			expected: ")",
			calls:    []string{"echo ')'"},
		},
		{
			name:     "nested backticks via the nesting rule",
			input:    "`echo `echo hi``",
			outputs:  map[string]string{"echo `echo hi`": "hi\n"},
			expected: "hi",
			calls:    []string{"echo `echo hi`"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := newMockExecutor()
			executor.outputs = tt.outputs

			result, err := ExpandToken(tt.input, Options{Executor: executor})
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
			if tt.calls != nil {
				assert.Equal(t, tt.calls, executor.calls)
			}
		})
	}
}

// Variable expansion applies to the literal text around spans, but never to
// body text (the recursive parse expands bodies) and never to spliced
// output.
func TestExpandToken_VariablesAroundSpansOnly(t *testing.T) {
	os.Setenv("EXPAND_TOKEN_VAR", "value")
	defer os.Unsetenv("EXPAND_TOKEN_VAR")

	executor := newMockExecutor()
	executor.outputs["echo $EXPAND_TOKEN_VAR"] = "expanded-by-recursion\n"

	result, err := ExpandToken("$EXPAND_TOKEN_VAR/$(echo $EXPAND_TOKEN_VAR)", Options{Executor: executor})
	require.NoError(t, err)

	// Segment before the span expanded; body handed over verbatim.
	assert.Equal(t, "value/expanded-by-recursion", result)
	assert.Equal(t, []string{"echo $EXPAND_TOKEN_VAR"}, executor.calls)

	// Spliced output containing variable syntax stays literal.
	executor2 := newMockExecutor()
	executor2.outputs["cmd"] = "$EXPAND_TOKEN_VAR\n"
	result2, err := ExpandToken("$(cmd)", Options{Executor: executor2})
	require.NoError(t, err)
	assert.Equal(t, "$EXPAND_TOKEN_VAR", result2)
}

// Without an executor, spans stay verbatim (body text untouched) while the
// surrounding text still expands.
func TestExpandToken_NilExecutorLeavesSpansVerbatim(t *testing.T) {
	os.Setenv("EXPAND_TOKEN_VAR", "value")
	defer os.Unsetenv("EXPAND_TOKEN_VAR")

	result, err := ExpandToken("$EXPAND_TOKEN_VAR-$(echo $EXPAND_TOKEN_VAR)", Options{})
	require.NoError(t, err)
	assert.Equal(t, "value-$(echo $EXPAND_TOKEN_VAR)", result)
}

// Escape-marked $ and ` (from \$ and \`) never start a span. A marked $ is
// resolved to a literal $ by variable expansion; a marked backtick keeps
// its marker for StripEscapeMarkers. Either way nothing executes.
func TestExpandToken_EscapeMarkersBlockSpans(t *testing.T) {
	executor := newMockExecutor()
	executor.outputs["whoami"] = "testuser\n"

	// Marked dollar: $(date) must not execute; the marked $ becomes a
	// literal $ (variable expansion consumes the marker).
	result, err := ExpandToken(EscapeMarker+"$(date)", Options{Executor: executor})
	require.NoError(t, err)
	assert.Equal(t, "$(date)", result)
	assert.Empty(t, executor.calls)

	// Both backticks marked: no substitution at all.
	input := EscapeMarker + "`whoami" + EscapeMarker + "`"
	result, err = ExpandToken(input, Options{Executor: executor})
	require.NoError(t, err)
	assert.Equal(t, input, result)
	assert.Empty(t, executor.calls)

	// A marked backtick next to real backticks does not pair with them.
	executor2 := newMockExecutor()
	executor2.outputs["echo x"] = "x\n"
	result2, err := ExpandToken("`echo x`"+EscapeMarker+"`", Options{Executor: executor2})
	require.NoError(t, err)
	assert.Equal(t, "x"+EscapeMarker+"`", result2)
	assert.Equal(t, []string{"echo x"}, executor2.calls)
}

// Executor errors (a body that fails to parse) propagate: the whole line
// fails.
func TestExpandToken_ExecutorErrorPropagates(t *testing.T) {
	executor := newMockExecutor()
	executor.err = errors.New("parse error inside body")

	_, err := ExpandToken("$(bad body)", Options{Executor: executor})
	require.Error(t, err)
	assert.ErrorIs(t, err, executor.err)
}

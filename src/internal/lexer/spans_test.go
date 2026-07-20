package lexer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindSubstitutionSpans(t *testing.T) {
	marker := string(EscapeMarker)

	tests := []struct {
		name    string
		content string
		want    []SubstitutionSpan
	}{
		{"no spans", "hello world", nil},
		{"empty content", "", nil},
		{"dollar without paren", "$HOME", nil},
		{"simple dollar paren", "$(pwd)", []SubstitutionSpan{{0, 6, "pwd"}}},
		{"text around span", "a$(pwd)b", []SubstitutionSpan{{1, 7, "pwd"}}},
		{
			"two spans",
			"$(a) $(b)",
			[]SubstitutionSpan{{0, 4, "a"}, {5, 9, "b"}},
		},
		{
			"nested dollar paren is ONE top-level span",
			"$(echo $(x))",
			[]SubstitutionSpan{{0, 12, "echo $(x)"}},
		},
		{
			"single-quoted paren does not close the span",
			"$(echo ')')",
			[]SubstitutionSpan{{0, 11, "echo ')'"}},
		},
		{
			"double-quoted paren does not close the span",
			`$(echo "x)y")`,
			[]SubstitutionSpan{{0, 13, `echo "x)y"`}},
		},
		{
			"single quotes suppress nested dollar paren",
			"$(echo '$(pwd)')",
			[]SubstitutionSpan{{0, 16, "echo '$(pwd)'"}},
		},
		{
			"backslash-escaped paren does not close the span",
			`$(echo \))`,
			[]SubstitutionSpan{{0, 10, `echo \)`}},
		},
		{"simple backtick", "`x`", []SubstitutionSpan{{0, 3, "x"}}},
		{"backtick with text", "a`echo hi`b", []SubstitutionSpan{{1, 10, "echo hi"}}},
		{
			"nested backticks via the nesting rule",
			"`echo `echo hi``",
			[]SubstitutionSpan{{0, 16, "echo `echo hi`"}},
		},
		{
			"backtick inside dollar paren",
			"$(echo `x`)",
			[]SubstitutionSpan{{0, 11, "echo `x`"}},
		},
		{
			"mixed dollar and backtick spans",
			"$(a)`b`",
			[]SubstitutionSpan{{0, 4, "a"}, {4, 7, "b"}},
		},
		{
			"marked dollar is literal",
			marker + "$(x)",
			nil,
		},
		{
			"marked backticks are literal",
			marker + "`x" + marker + "`",
			nil,
		},
		{
			"marked backtick does not pair with a real one",
			"`echo x`" + marker + "`",
			[]SubstitutionSpan{{0, 8, "echo x"}},
		},
		{
			// A literal backslash at top level (from \\ in the input) does
			// not hide a following span.
			"top-level backslash before span",
			`\$(pwd)`,
			[]SubstitutionSpan{{1, 7, "pwd"}},
		},
		{
			// Only $( pushes: a bare ( in a body is text, so the first )
			// closes. The trailing ) is plain data.
			"bare parens in body",
			"$((x))",
			[]SubstitutionSpan{{0, 5, "(x"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FindSubstitutionSpans(tt.content)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFindSubstitutionSpans_Unclosed(t *testing.T) {
	for _, content := range []string{"$(x", "`x", "$(echo 'x", "$(a $(b)"} {
		_, err := FindSubstitutionSpans(content)
		assert.ErrorIs(t, err, ErrUnclosedSubstitution, "content %q", content)
	}
}

// Cross-check: spans found in Tokenize-produced content line up with the
// bodies Tokenize preserved verbatim.
func TestFindSubstitutionSpans_MatchesTokenize(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		token  int // index of the token holding the span
		bodies []string
	}{
		{"simple", "echo $(pwd)", 1, []string{"pwd"}},
		{"quoted close paren", "echo $(echo 'a )b')", 1, []string{"echo 'a )b'"}},
		{"nested", "echo $(echo $(echo deep))", 1, []string{"echo $(echo deep)"}},
		{"backtick nesting", "echo `echo `echo hi``", 1, []string{"echo `echo hi`"}},
		{"inside double quotes", `echo "v=$(pwd)"`, 1, []string{"pwd"}},
		{"two spans one word", "echo $(a)-$(b)", 1, []string{"a", "b"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := Tokenize(tt.input)
			require.NoError(t, err)
			require.Greater(t, len(tokens), tt.token)

			spans, err := FindSubstitutionSpans(tokens[tt.token].Content)
			require.NoError(t, err)
			require.Len(t, spans, len(tt.bodies))
			for i, body := range tt.bodies {
				assert.Equal(t, body, spans[i].Body)
			}
		})
	}
}

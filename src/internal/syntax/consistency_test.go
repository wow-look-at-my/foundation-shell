package syntax

import (
	"testing"

	"foundation-shell/internal/lexer"
	"github.com/stretchr/testify/require"
)

// TestLexerAnalyzerConsistency is the spec's consistency promise as a
// cross-check: for a broad input table, lexer.Tokenize error/success must
// agree with syntax.Analyze validity. The analyzer is allowed to be
// STRICTER than the lexer (it also reports structural errors the parser
// would reject, like trailing operators), so the hard invariant is
// one-way: anything the lexer rejects, the analyzer must flag as invalid.
// Every row additionally pins the expected outcome for both engines.
//
// Deliberately excluded (owned by the upcoming depth-tracked quote
// NESTING batch, whose semantics will change): same-type nested quotes
// ('a 'b' c'), cross-type quote adjacency ("a"'b, 'it'\''s), backticks
// wrapped in double quotes, and single quotes inside a double-quoted
// substitution.
func TestLexerAnalyzerConsistency(t *testing.T) {
	tests := []struct {
		input      string
		wantLexErr bool
		wantValid  bool
	}{
		// Plain words and quotes
		{"echo hello", false, true},
		{"echo 'a b'", false, true},
		{`echo "a b"`, false, true},
		{`echo "a" 'b'`, false, true},
		{"echo ''", false, true},
		{`echo ""`, false, true},
		{"''", false, true},

		// Escapes
		{`echo hello\ world`, false, true},
		{`echo \$HOME`, false, true},
		{`echo \|`, false, true},
		{`echo hello\`, false, true},
		{"echo \\`whoami\\`", false, true},

		// Command substitutions, including quoted ) inside the body
		{"echo $(date)", false, true},
		{"echo $(echo $(whoami))", false, true},
		{`echo $(echo "a  b")`, false, true},
		{`echo $(echo "x)y")`, false, true},
		{`echo $(echo ')')`, false, true},
		{"echo $(cat f | grep x; true)", false, true},
		{"echo `date`", false, true},
		{"echo 'a'$(date)", false, true},
		{"prefix$(cmd)suffix", false, true},
		{`echo "$(whoami)"`, false, true},

		// Operators with and without whitespace
		{"echo hello|grep world", false, true},
		{"echo a;echo b", false, true},
		{"a&&b", false, true},
		{"a||b", false, true},
		{"a&b", false, true},
		{"echo hi>out.txt", false, true},
		{"cat<in.txt", false, true},
		{"echo a2>f", false, true},
		{"cmd 2>err.log", false, true},
		{"< in.txt cat", false, true},
		{"echo '|'", false, true},
		{`echo "|"`, false, true},

		// Comments
		{"# just a comment", false, true},
		{"#!/usr/bin/env fsh", false, true},
		{"echo a # rest", false, true},
		{"echo a # unclosed ' quote inside comment", false, true},
		{"foo#bar", false, true},

		// Newlines
		{"echo a\necho b", false, true},
		{"\n\necho a\n", false, true},
		{"echo a &&\necho b", false, true},
		{"echo 'a\nb'", false, true},
		{"echo $(a\nb)", false, true},

		// Unclosed constructs: the lexer errors and the analyzer must
		// agree
		{"echo 'x", true, false},
		{`echo "x`, true, false},
		{"echo $(date", true, false},
		{"echo `date", true, false},
		{`echo "$(a`, true, false},
		{"echo $(echo ')'", true, false},

		// Structural errors: the lexer tokenizes fine, the parser rejects,
		// and the analyzer must flag them (the stricter direction)
		{"echo hello |", false, false},
		{"echo hello | ", false, false},
		{"echo hello &&", false, false},
		{"echo > ", false, false},
		{"echo | # done", false, false},
		{"| foo", false, false},
		{"  && foo", false, false},
		{"|", false, false},
		{"; foo", false, false},

		// Trailing semicolon is valid
		{"echo hello ;", false, true},
		{"echo hello ; ", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			_, lexErr := lexer.Tokenize(tt.input)
			result := Analyze(tt.input)

			require.Equal(t, tt.wantLexErr, lexErr != nil,
				"lexer.Tokenize(%q) error = %v, want error: %v", tt.input, lexErr, tt.wantLexErr)

			require.Equal(t, tt.wantValid, result.Valid,
				"Analyze(%q).Valid = %v (errors: %#v), want %v", tt.input, result.Valid, result.Errors, tt.wantValid)

			// The one-way consistency invariant: anything the lexer
			// rejects, the analyzer must flag.
			if lexErr != nil {
				require.False(t, result.Valid,
					"lexer rejected %q (%v) but the analyzer calls it valid", tt.input, lexErr)
			}
		})
	}
}

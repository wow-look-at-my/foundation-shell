package syntax

import (
	"github.com/stretchr/testify/require"
	"testing"
)

// Golden test pinning the exact multi-error output format: blocks of
// line/carets/message separated by ONE blank line, with exactly one
// trailing newline.
func TestFormatDiagnostics_GoldenTwoErrorFormat(t *testing.T) {
	input := "echo |\necho >"
	errors := []SyntaxError{
		{Start: 5, End: 6, Message: "unexpected operator at end"},
		{Start: 12, End: 13, Message: "missing redirection target"},
	}

	got := FormatDiagnostics(input, errors)

	want := "echo |\n" +
		"     ^\n" +
		"error: unexpected operator at end\n" +
		"\n" +
		"echo >\n" +
		"     ^\n" +
		"error: missing redirection target\n"
	require.Equal(t, want, got)
}

// The same format promise holds for errors produced by the real analyzer:
// an unclosed substitution inside unclosed double quotes yields two blocks.
func TestFormatDiagnostics_GoldenAnalyzerIntegration(t *testing.T) {
	input := `echo "$(a`
	result := Analyze(input)

	require.False(t, result.Valid)

	require.Equal(t, 2, len(result.Errors))

	got := FormatDiagnostics(input, result.Errors)

	want := "echo \"$(a\n" +
		"     ^^^^\n" +
		"error: unclosed double quote\n" +
		"\n" +
		"echo \"$(a\n" +
		"     ^^^^\n" +
		"error: unclosed command substitution $(...)\n"
	require.Equal(t, want, got)
}

// A single error block ends with exactly one trailing newline.
func TestFormatDiagnostics_TrailingNewline(t *testing.T) {
	input := "echo |"
	errors := []SyntaxError{{Start: 5, End: 6, Message: "unexpected operator at end"}}

	got := FormatDiagnostics(input, errors)

	want := "echo |\n" +
		"     ^\n" +
		"error: unexpected operator at end\n"
	require.Equal(t, want, got)
}

// Rune correctness: multi-byte characters on an earlier line must not
// shift the selected line or the caret column. (Byte/rune index mixing
// used to select line 1 and misalign the carets here.)
func TestFormatDiagnostics_MultibyteMultiline(t *testing.T) {
	input := "日本語\ncat \"x"
	result := Analyze(input)

	require.False(t, result.Valid)

	got := FormatDiagnostics(input, result.Errors)

	want := "cat \"x\n" +
		"    ^^\n" +
		"error: unclosed double quote\n"
	require.Equal(t, want, got)
}

// Rune correctness on a single line: carets are counted in runes, so a
// multi-byte character inside the error span still yields span-width
// carets at the right column.
func TestFormatDiagnostics_MultibyteSingleLine(t *testing.T) {
	input := `echo "héllo`
	result := Analyze(input)

	require.False(t, result.Valid)

	got := FormatDiagnostics(input, result.Errors)

	want := "echo \"héllo\n" +
		"     ^^^^^^\n" +
		"error: unclosed double quote\n"
	require.Equal(t, want, got)
}

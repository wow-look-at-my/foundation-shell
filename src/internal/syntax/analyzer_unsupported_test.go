package syntax

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The analyzer owns the caret diagnostics and the highlighter, and the shell
// prefers its message over the parser's. It must therefore reject the same
// constructs the parser rejects, with the same wording.
func TestAnalyze_RejectsBackgroundAmpersand(t *testing.T) {
	result := Analyze("sleep 30 &")

	require.False(t, result.Valid)
	require.Len(t, result.Errors, 1)
	assert.Equal(t, "background execution is not supported", result.Errors[0].Message)
	// The caret points at the & itself, the last character of the input.
	assert.Equal(t, 9, result.Errors[0].Start)
	assert.Equal(t, 10, result.Errors[0].End)
}

func TestAnalyze_RejectsAmpersandBetweenCommands(t *testing.T) {
	result := Analyze("echo a & echo b")

	require.False(t, result.Valid)
	require.Len(t, result.Errors, 1)
	assert.Equal(t, "background execution is not supported", result.Errors[0].Message)
	assert.Equal(t, 7, result.Errors[0].Start)
}

// `<<<` is three `<`, so it forms two adjacent pairs. It must still produce
// ONE error: a doubled diagnostic teaches the reader to skim the output.
func TestAnalyze_HeredocReportedOnce(t *testing.T) {
	for _, input := range []string{"cat <<EOF", "cat << EOF", "cat <<<word"} {
		t.Run(input, func(t *testing.T) {
			result := Analyze(input)

			require.False(t, result.Valid)
			require.Len(t, result.Errors, 1)
			assert.Equal(t, "here-documents are not supported", result.Errors[0].Message)
			// The caret points at the first `<`, where the construct starts.
			assert.Equal(t, 4, result.Errors[0].Start)
		})
	}
}

// The guards must not fire on ordinary input.
func TestAnalyze_LegitimateAmpersandAndRedirectionStayValid(t *testing.T) {
	inputs := []string{
		"echo a&b",
		`echo "a & b"`,
		"echo 'a & b'",
		`echo \&`,
		"true && echo ok",
		"false || echo ok",
		"cat < in.txt",
		"echo hi > out.txt",
	}

	for _, input := range inputs {
		t.Run(input, func(t *testing.T) {
			result := Analyze(input)
			assert.True(t, result.Valid, "errors: %#v", result.Errors)
		})
	}
}

// `> &` is the fd-duplication guard's case; that message names the problem
// better, so the background guard must leave a redirection target alone.
func TestAnalyze_AmpersandRedirectionTargetNotBackground(t *testing.T) {
	result := Analyze("echo hi > &1")

	for _, e := range result.Errors {
		assert.NotEqual(t, "background execution is not supported", e.Message)
	}
}

package parser

import (
	"os"
	"testing"

	"foundation-shell/internal/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Item: operators come ONLY from the lexer's marking. Quoted or escaped
// operator characters are literal data, never live operators.
func TestParse_QuotedOperatorCharactersAreLiterals(t *testing.T) {
	tests := []struct {
		name  string
		input string
		args  []string
	}{
		{"single-quoted pipe", `echo '|'`, []string{"echo", "|"}},
		{"double-quoted pipe", `echo "|"`, []string{"echo", "|"}},
		{"escaped pipe", `echo \|`, []string{"echo", "|"}},
		{"single-quoted semicolon", `echo ';'`, []string{"echo", ";"}},
		{"double-quoted semicolon", `echo ";"`, []string{"echo", ";"}},
		{"escaped semicolon", `echo \;`, []string{"echo", ";"}},
		{"single-quoted and", `echo '&&'`, []string{"echo", "&&"}},
		{"double-quoted or", `echo "||"`, []string{"echo", "||"}},
		{"single-quoted redirect out", `echo '>'`, []string{"echo", ">"}},
		{"escaped redirect out", `echo \>`, []string{"echo", ">"}},
		{"single-quoted redirect in", `echo '<'`, []string{"echo", "<"}},
		{"single-quoted stderr redirect", `echo '2>'`, []string{"echo", "2>"}},
		{"operator inside quoted word", `echo 'a|b'`, []string{"echo", "a|b"}},
		{"quoted gt as grep pattern", `grep '>' data.txt`, []string{"grep", ">", "data.txt"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chain, err := Parse(tt.input)
			require.NoError(t, err)
			require.Len(t, chain.Commands, 1)
			assert.Empty(t, chain.Operators)
			cmd := chain.Commands[0]
			assert.Equal(t, tt.args, cmd.Args)
			// No redirection may have been applied from quoted characters.
			assert.Empty(t, cmd.InputFile)
			assert.Empty(t, cmd.OutputFile)
			assert.Empty(t, cmd.ErrorFile)
		})
	}
}

// Unquoted operator characters still work as operators (control case).
func TestParse_UnquotedOperatorsStillOperate(t *testing.T) {
	chain, err := Parse("echo a | cat")
	require.NoError(t, err)
	require.Len(t, chain.Commands, 2)
	require.Equal(t, []token.TokenType{token.Pipe}, chain.Operators)
}

// Item: a single trailing semicolon is valid and consumed.
func TestParse_TrailingSemicolon(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		commandCount int
		operators    []token.TokenType
	}{
		{"spaced trailing semicolon", "echo hi ;", 1, nil},
		{"adjacent trailing semicolon", "echo hi;", 1, nil},
		{"trailing newline", "echo hi\n", 1, nil},
		{"chain with trailing semicolon", "echo a ; echo b ;", 2, []token.TokenType{token.Semicolon}},
		{"pipeline with trailing semicolon", "echo a | cat ;", 2, []token.TokenType{token.Pipe}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chain, err := Parse(tt.input)
			require.NoError(t, err)
			require.Len(t, chain.Commands, tt.commandCount)
			assert.Equal(t, len(tt.operators), len(chain.Operators))
			for i, op := range tt.operators {
				assert.Equal(t, op, chain.Operators[i])
			}
			// Invariant: len(Operators) == len(Commands) - 1
			assert.Equal(t, len(chain.Commands)-1, len(chain.Operators))
		})
	}
}

// Trailing operators other than ; are still errors, all carrying context.
func TestParse_TrailingOperatorErrors(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr error
		wantMsg string
	}{
		{"trailing pipe", "cmd |", ErrTrailingOperator, "unexpected operator at end: |"},
		{"trailing and", "cmd &&", ErrTrailingOperator, "unexpected operator at end: &&"},
		{"trailing or", "cmd ||", ErrTrailingOperator, "unexpected operator at end: ||"},
		{"consecutive semicolons", "echo a ; ; b", ErrConsecutiveOperators, "consecutive operators: ; followed by ;"},
		{"double trailing semicolon", "echo a ; ;", ErrConsecutiveOperators, "consecutive operators: ; followed by ;"},
		// The post-loop path (operator followed by a redirection-only
		// command) must carry `: <op>` context too.
		{"semicolon then redirection-only command", "echo hi ; > file", ErrTrailingOperator, "unexpected operator at end: ;"},
		{"pipe then redirection-only command", "echo hi | > file", ErrTrailingOperator, "unexpected operator at end: |"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.input)
			require.Error(t, err)
			assert.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.wantMsg, err.Error())
		})
	}
}

// Item: redirection targets must be non-empty after expansion.
func TestParse_EmptyRedirectionTarget(t *testing.T) {
	os.Unsetenv("UNSET_VAR_FOR_REDIR_TEST")

	tests := []struct {
		name  string
		input string
	}{
		{"unset variable target", "echo hi > $UNSET_VAR_FOR_REDIR_TEST"},
		{"empty single-quoted target", "echo hi > ''"},
		{"empty double-quoted target", `echo hi > ""`},
		{"empty input target", "cat < $UNSET_VAR_FOR_REDIR_TEST"},
		{"empty stderr target", "cmd 2> $UNSET_VAR_FOR_REDIR_TEST"},
		{"empty append target", "echo hi >> $UNSET_VAR_FOR_REDIR_TEST"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.input)
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrEmptyRedirectionTarget)
			// Canonical message pinned by the spec.
			assert.Equal(t, "empty redirection target", err.Error())
		})
	}
}

// Item: unquoted &-prefixed redirection targets are rejected loudly
// (2>&1 lexes as `2>` + `&1`); a quoted '&1' stays a legal filename.
func TestParse_FdDuplicationUnsupported(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantMsg string
	}{
		{"stderr to stdout", "echo hi 2>&1", "file descriptor duplication is not supported: &1"},
		{"stdout to stderr", "echo hi >&2", "file descriptor duplication is not supported: &2"},
		{"append form", "echo hi 2>>&1", "file descriptor duplication is not supported: &1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.input)
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrFdDuplicationUnsupported)
			assert.Equal(t, tt.wantMsg, err.Error())
		})
	}
}

func TestParse_QuotedAmpersandTargetIsAFilename(t *testing.T) {
	chain, err := Parse("echo hi > '&1'")
	require.NoError(t, err)
	require.Len(t, chain.Commands, 1)
	assert.Equal(t, "&1", chain.Commands[0].OutputFile)

	chain, err = Parse(`echo hi 2> "&1"`)
	require.NoError(t, err)
	assert.Equal(t, "&1", chain.Commands[0].ErrorFile)
}

// redirection.md §9.5: the fd-duplication guard inspects the target token
// AS WRITTEN (pre-expansion). A target that becomes `&1` only through
// expansion is a legitimate filename; a literal unquoted `&`-prefixed word
// is rejected even when expansion would change it.
func TestParse_FdDuplicationGuardIsPreExpansion(t *testing.T) {
	t.Setenv("FD_GUARD_TEST_VAR", "&1")

	// Expansion-produced &1: legal — the redirection targets a file
	// literally named &1.
	chain, err := Parse("echo hi > $FD_GUARD_TEST_VAR")
	require.NoError(t, err)
	require.Len(t, chain.Commands, 1)
	assert.Equal(t, "&1", chain.Commands[0].OutputFile)

	// Literal &-prefixed word: rejected on the WRITTEN form, with the
	// written word in the message, regardless of what expansion would do.
	t.Setenv("FD_GUARD_SUFFIX", "name")
	_, err = Parse("echo hi > &$FD_GUARD_SUFFIX")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrFdDuplicationUnsupported)
	assert.Equal(t, "file descriptor duplication is not supported: &$FD_GUARD_SUFFIX", err.Error())
}

// lexer.md §3.4: a newline right after a redirection operator is NOT line
// continuation — the lexer materializes the separator and the parser
// reports the dangling redirection. Chain operators DO continue.
func TestParse_NewlineAfterRedirectionIsError(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantMsg string
	}{
		{"stdout redirect", "echo hi >\nout.txt", "missing redirection target: > followed by operator ;"},
		{"stdin redirect", "cat <\nin.txt", "missing redirection target: < followed by operator ;"},
		{"stderr append", "cmd 2>>\nerr.log", "missing redirection target: 2>> followed by operator ;"},
		{"redirect at EOF after newline", "echo hi >\n", "missing redirection target: >"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.input)
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrMissingRedirectionTarget)
			assert.Equal(t, tt.wantMsg, err.Error())
		})
	}

	// Chain operators still continue across the newline.
	chain, err := Parse("echo a &&\necho b")
	require.NoError(t, err)
	require.Len(t, chain.Commands, 2)
}

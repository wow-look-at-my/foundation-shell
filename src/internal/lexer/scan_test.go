package lexer

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Inputs that exercise every branch of the scanner, reused by the property
// tests below.
var scanCorpus = []string{
	"",
	"   ",
	"echo hello",
	"echo   hello   world",
	"echo a&b",
	"echo a&&b",
	"cmd1 | cmd2 && cmd3 || cmd4 ; cmd5",
	"echo hi > out.txt",
	"echo hi 2>> err.log",
	"echo a2>f",
	"cat < in.txt",
	"echo 'single quoted'",
	`echo "double quoted"`,
	"echo 'a 'b' c'",
	`echo "it's a \"test\""`,
	"echo $(echo nested)",
	"echo `date`",
	"echo $(echo $(echo deep))",
	"echo ${HOME}/sub",
	"echo $? $HOME",
	"echo \\$literal \\`tick",
	"# whole line comment",
	"echo hi # trailing comment",
	"echo one\necho two",
	"echo one &&\necho two",
	"cmd ;\n\n cmd2",
	"echo 'unclosed",
	`echo "unclosed`,
	"echo $(unclosed",
	"echo `unclosed",
	"echo ''",
	`echo ""`,
	"\techo\ttabs\t",
	"echo hi\n",
	"\n\necho after blank lines",
}

// The scanner accounts for EVERY rune: words, operators, whitespace and
// comments together reproduce the input exactly. This is what lets one
// tokenizer serve both execution and highlighting — the highlighter can
// paint the token stream and get the user's line back.
func TestScan_TokensReproduceTheInput(t *testing.T) {
	for _, input := range scanCorpus {
		t.Run(input, func(t *testing.T) {
			var b strings.Builder
			for _, tok := range Scan(input).Tokens {
				b.WriteString(tok.Raw)
			}
			assert.Equal(t, input, b.String())
		})
	}
}

// Spans are contiguous, ordered, and index the input they came from.
func TestScan_SpansAreContiguousAndAccurate(t *testing.T) {
	for _, input := range scanCorpus {
		t.Run(input, func(t *testing.T) {
			runes := []rune(input)
			next := 0
			for _, tok := range Scan(input).Tokens {
				require.Equal(t, next, tok.Start, "gap or overlap before %q", tok.Raw)
				require.LessOrEqual(t, tok.End, len(runes))
				require.Less(t, tok.Start, tok.End, "empty span for %q", tok.Raw)
				assert.Equal(t, string(runes[tok.Start:tok.End]), tok.Raw)
				next = tok.End
			}
			assert.Equal(t, len(runes), next, "input not fully covered")
		})
	}
}

// Tokenize is a projection of Scan, so the words it returns are Scan's
// words in order, with the same content and flags. Only whitespace,
// comments and the materialized ; separator differ.
func TestTokenize_ProjectsScanWithoutAlteringWords(t *testing.T) {
	for _, input := range scanCorpus {
		t.Run(input, func(t *testing.T) {
			scan := Scan(input)
			tokens, err := Tokenize(input)
			if len(scan.Unclosed) > 0 {
				require.Error(t, err)
				assert.Equal(t, scan.Unclosed[0].Message, err.Error())
				return
			}
			require.NoError(t, err)

			var fromScan, fromTokenize []string
			for _, tok := range scan.Tokens {
				if tok.Kind == KindWord {
					fromScan = append(fromScan, tok.Content)
				}
			}
			for _, tok := range tokens {
				if !tok.IsOperator {
					fromTokenize = append(fromTokenize, tok.Content)
				}
			}
			assert.Equal(t, fromScan, fromTokenize)
		})
	}
}

// Scan records every unterminated construct rather than stopping at one,
// and Tokenize surfaces the innermost — the first Scan listed.
func TestScan_ReportsEveryUnclosedConstructInnermostFirst(t *testing.T) {
	result := Scan(`echo $(foo "bar`)

	require.Len(t, result.Unclosed, 2)
	assert.Equal(t, "unclosed double quote", result.Unclosed[0].Message)
	assert.Equal(t, "unclosed command substitution $(...)", result.Unclosed[1].Message)

	_, err := Tokenize(`echo $(foo "bar`)
	require.Error(t, err)
	assert.Equal(t, "unclosed double quote", err.Error())
}

// Roles are assigned once, for both consumers.
func TestScan_AssignsRoles(t *testing.T) {
	words := func(input string) []Role {
		var roles []Role
		for _, tok := range Scan(input).Tokens {
			if tok.Kind == KindWord {
				roles = append(roles, tok.Role)
			}
		}
		return roles
	}

	assert.Equal(t, []Role{RoleCommand, RoleArgument}, words("echo hi"))
	assert.Equal(t, []Role{RoleCommand, RoleRedirectionTarget}, words("echo > out"))
	// A redirection may lead a command: the target is still a target, and
	// the command name follows it.
	assert.Equal(t, []Role{RoleRedirectionTarget, RoleCommand}, words("< in.txt cat"))
	assert.Equal(t, []Role{RoleCommand, RoleCommand}, words("true; false"))
	// A newline separates commands the same way a ; does.
	assert.Equal(t, []Role{RoleCommand, RoleCommand}, words("true\nfalse"))
	assert.Equal(t, []Role{RoleCommand, RoleArgument, RoleCommand}, words("echo a | cat"))
}

// A word's Depth is its high-water nesting level, which the highlighter uses
// to shade nested constructs.
func TestScan_TracksWordDepth(t *testing.T) {
	depth := func(input string) int {
		for _, tok := range Scan(input).Tokens {
			if tok.Kind == KindWord {
				return tok.Depth
			}
		}
		t.Fatalf("no word token in %q", input)
		return 0
	}

	assert.Equal(t, 0, depth("plain"))
	assert.Equal(t, 1, depth("'quoted'"))
	assert.Equal(t, 1, depth("$(cmd)"))
	assert.Equal(t, 2, depth("$(echo $(deep))"))
}

package command

import (
	"bytes"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuiltinExport_SetsVariables(t *testing.T) {
	t.Setenv("FSH_TEST_EXPORT_A", "old") // registers cleanup

	var stdout, stderr bytes.Buffer
	code, err := ExecuteBuiltin("export",
		[]string{"FSH_TEST_EXPORT_A=one", "FSH_TEST_EXPORT_B=two"}, nil, &stdout, &stderr)
	t.Setenv("FSH_TEST_EXPORT_B", os.Getenv("FSH_TEST_EXPORT_B")) // cleanup for B

	require.Nil(t, err)
	assert.Equal(t, 0, code)
	assert.Equal(t, "one", os.Getenv("FSH_TEST_EXPORT_A"))
	assert.Equal(t, "two", os.Getenv("FSH_TEST_EXPORT_B"))
	assert.Empty(t, stderr.String())
}

// The FIRST '=' splits name from value: the value may be empty or contain
// more '='.
func TestBuiltinExport_ValueEdgeCases(t *testing.T) {
	t.Setenv("FSH_TEST_EXPORT_E", "sentinel")

	var stdout, stderr bytes.Buffer
	code, err := ExecuteBuiltin("export", []string{"FSH_TEST_EXPORT_E="}, nil, &stdout, &stderr)
	require.Nil(t, err)
	assert.Equal(t, 0, code)
	v, ok := os.LookupEnv("FSH_TEST_EXPORT_E")
	assert.True(t, ok)
	assert.Equal(t, "", v)

	code, err = ExecuteBuiltin("export", []string{"FSH_TEST_EXPORT_E=B=C"}, nil, &stdout, &stderr)
	require.Nil(t, err)
	assert.Equal(t, 0, code)
	assert.Equal(t, "B=C", os.Getenv("FSH_TEST_EXPORT_E"))
}

// A bare valid name (no '=') is a no-op success: every variable is already
// an environment variable in the same-process model.
func TestBuiltinExport_BareNameNoOp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code, err := ExecuteBuiltin("export", []string{"PATH"}, nil, &stdout, &stderr)
	require.Nil(t, err)
	assert.Equal(t, 0, code)
	assert.Empty(t, stderr.String())
	assert.Empty(t, stdout.String())
}

// Invalid names report and CONTINUE; the final status reflects the failure.
func TestBuiltinExport_InvalidNameContinues(t *testing.T) {
	t.Setenv("FSH_TEST_EXPORT_C", "old")
	t.Setenv("FSH_TEST_EXPORT_D", "old")

	var stdout, stderr bytes.Buffer
	code, err := ExecuteBuiltin("export",
		[]string{"FSH_TEST_EXPORT_C=1", "1BAD=2", "FSH_TEST_EXPORT_D=3"}, nil, &stdout, &stderr)

	require.Nil(t, err)
	assert.Equal(t, 1, code)
	assert.Equal(t, "export: invalid name: 1BAD=2\n", stderr.String())
	// Both valid arguments were still processed.
	assert.Equal(t, "1", os.Getenv("FSH_TEST_EXPORT_C"))
	assert.Equal(t, "3", os.Getenv("FSH_TEST_EXPORT_D"))
}

func TestBuiltinExport_InvalidNameForms(t *testing.T) {
	for _, arg := range []string{"=x", "1X=y", "A-B=z", "A B=c"} {
		t.Run(arg, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code, err := ExecuteBuiltin("export", []string{arg}, nil, &stdout, &stderr)
			require.Nil(t, err)
			assert.Equal(t, 1, code)
			assert.Equal(t, "export: invalid name: "+arg+"\n", stderr.String())
		})
	}
}

// Bare export prints the environment sorted by name, one NAME=value entry
// per line. (Compared against the sorted os.Environ() directly: entry
// VALUES may themselves contain newlines, so naive line-splitting cannot
// verify sortedness.)
func TestBuiltinExport_ListsEnvironmentSorted(t *testing.T) {
	t.Setenv("FSH_TEST_EXPORT_LIST", "visible")

	var stdout, stderr bytes.Buffer
	code, err := ExecuteBuiltin("export", nil, nil, &stdout, &stderr)
	require.Nil(t, err)
	assert.Equal(t, 0, code)

	env := os.Environ()
	sort.Strings(env)
	expected := strings.Join(env, "\n") + "\n"
	assert.Equal(t, expected, stdout.String())
	assert.Contains(t, stdout.String(), "FSH_TEST_EXPORT_LIST=visible")
}

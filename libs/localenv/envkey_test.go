package localenv

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvKeyForServerless(t *testing.T) {
	for _, in := range []string{"4", "v4", "V4"} {
		assert.Equal(t, "serverless/serverless-v4", EnvKeyForServerless(in))
	}
}

func TestEnvKeyForSparkVersion(t *testing.T) {
	assert.Equal(t, "dbr/15.4.x-scala2.12", EnvKeyForSparkVersion("15.4.x-scala2.12"))
}

func TestPythonMinorFromRequires(t *testing.T) {
	cases := map[string]string{
		"==3.12.*": "3.12",
		">=3.12":   "3.12",
		"==3.12.3": "3.12",
		"~=3.11":   "3.11",
		// Multi-clause specifiers: the lower bound is the version to install,
		// regardless of clause order. Taking the first number would pick the
		// excluded upper bound (e.g. 3.13 from "<3.13").
		"<3.13,>=3.10":  "3.10",
		">=3.10,<3.13":  "3.10",
		">=3.10, <3.13": "3.10",
		"<4.0,>=3.9":    "3.9",
		"===3.11":       "3.11",
	}
	for in, want := range cases {
		got, err := PythonMinorFromRequires(in)
		require.NoError(t, err)
		assert.Equal(t, want, got, "input %q", in)
	}

	// A bare version with no operator is a valid floor.
	got, err := PythonMinorFromRequires("3.12")
	require.NoError(t, err)
	assert.Equal(t, "3.12", got)

	// No usable floor: only upper-bound / exclusion clauses. Must error rather
	// than select a forbidden/capped version.
	for _, in := range []string{"<3.13", "<=3.12", "!=3.12", "<3.13,!=3.12", "garbage"} {
		_, err := PythonMinorFromRequires(in)
		assert.Error(t, err, "input %q must error", in)
	}
}

package localenv

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvKeyForServerless(t *testing.T) {
	// The documented input is a bare number; a "v"-prefixed form is also accepted.
	// Both map to the "serverless-vN" env key.
	for _, in := range []string{"4", "v4", "V4"} {
		assert.Equal(t, "serverless/serverless-v4", EnvKeyForServerless(in))
	}
	for _, in := range []string{"5", "v5", "V5"} {
		assert.Equal(t, "serverless/serverless-v5", EnvKeyForServerless(in))
	}
}

func TestDefaultServerlessVersionIsBareNumber(t *testing.T) {
	// The default stand-in is stored in the documented bare form; it still maps
	// to the serverless-vN env key via NormalizeServerless.
	assert.Equal(t, "5", defaultServerlessVersion)
	assert.Equal(t, "serverless/serverless-v5", EnvKeyForServerless(defaultServerlessVersion))
}

func TestValidServerlessVersion(t *testing.T) {
	for _, ok := range []string{"5", "4", "v5", "V5", "17", "0"} {
		assert.True(t, ValidServerlessVersion(ok), "%q should be valid", ok)
	}
	for _, bad := range []string{"", "v", "vv5", "5x", "latest", " 5", "5 ", "v5.1", "-5"} {
		assert.False(t, ValidServerlessVersion(bad), "%q should be invalid", bad)
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
		// The effective floor is the HIGHEST lower bound, regardless of order.
		">=3.8,>=3.11": "3.11",
		">=3.11,>=3.8": "3.11",
		// A bare floor alongside an exclusion is still a floor.
		"!=3.11,3.12":   "3.12",
		"3.12,!=3.12.4": "3.12",
		// Bare version with no operator.
		"3.12": "3.12",
		// Whitespace and patch components tolerated.
		">= 3.10 , < 3.13": "3.10",
		// Strict ">" with no patch excludes the whole minor series (PEP 440), so
		// the floor is the next minor up.
		">3.10":       "3.11",
		">3.10,<3.13": "3.11",
		">=3.9,>3.10": "3.11",
		// Strict ">" WITH a patch does not exclude the minor series: 3.10.6
		// satisfies ">3.10.5", so the floor stays 3.10.
		">3.10.5":       "3.10",
		">3.10.5,<3.13": "3.10",
		">=3.10.2":      "3.10",
	}
	for in, want := range cases {
		got, err := PythonMinorFromRequires(in)
		require.NoError(t, err)
		assert.Equal(t, want, got, "input %q", in)
	}

	// No usable floor: only upper-bound / exclusion clauses. Must error rather
	// than select a forbidden/capped version.
	for _, in := range []string{"<3.13", "<=3.12", "!=3.12", "<3.13,!=3.12", "garbage", ""} {
		_, err := PythonMinorFromRequires(in)
		assert.Error(t, err, "input %q must error", in)
	}
}

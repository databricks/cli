package dbconnect

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
	}
	for in, want := range cases {
		got, err := PythonMinorFromRequires(in)
		require.NoError(t, err)
		assert.Equal(t, want, got)
	}
	_, err := PythonMinorFromRequires("garbage")
	assert.Error(t, err)
}

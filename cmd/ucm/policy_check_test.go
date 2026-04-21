package ucm

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCmd_PolicyCheck_ValidFixturePasses(t *testing.T) {
	stdout, stderr, err := runVerb(t, validFixtureDir(t), "policy-check")
	t.Logf("stdout=%q stderr=%q", stdout, stderr)

	require.NoError(t, err)
	assert.Contains(t, stdout, "Policy check OK!")
}

func TestCmd_PolicyCheck_MissingTagFixtureFails(t *testing.T) {
	_, stderr, err := runVerb(t, filepath.Join("testdata", "missing_tag"), "policy-check")

	require.Error(t, err)
	assert.Contains(t, stderr, "requires tag")
}

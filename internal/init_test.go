package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAccBundleInitErrorOnUnknownFields(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	tmpDir := t.TempDir()
	_, _, err := RequireErrorRun(t, "bundle", "init", "./testdata/init/field-does-not-exist", "--output-dir", tmpDir)
	assert.EqualError(t, err, "failed to compute file content for bar.tmpl. variable \"does_not_exist\" not defined")
}

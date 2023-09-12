package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBundleInitErrorOnUnknownFields(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	tmpDir := t.TempDir()
	_, _, err := RequireErrorRun(t, "bundle", "init", "./testdata/init/field-does-not-exist", "--output-dir", tmpDir)
	assert.EqualError(t, err, `failed to compute file content for bar.tmpl. template: :2:2: executing "" at <.does_not_exist>: map has no entry for key "does_not_exist"`)
}

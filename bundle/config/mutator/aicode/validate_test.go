package aicode

import (
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/bundletest"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func bundleForValidate(t *testing.T, codeSourcePath string, gitSource *jobs.GitSource) *bundle.Bundle {
	t.Helper()
	dir := t.TempDir()
	b := &bundle.Bundle{
		BundleRootPath: dir,
		SyncRootPath:   dir,
		Config: config.Root{
			Bundle: config.Bundle{Target: "default"},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"train": {
						JobSettings: jobs.JobSettings{
							GitSource: gitSource,
							Tasks: []jobs.Task{
								{
									TaskKey:       "train",
									AiRuntimeTask: &jobs.AiRuntimeTask{CodeSourcePath: codeSourcePath},
								},
							},
						},
					},
				},
			},
		},
	}
	bundletest.SetLocation(b, ".", []dyn.Location{{File: filepath.Join(dir, "databricks.yml")}})
	return b
}

func TestValidateMissingCodeSourceDir(t *testing.T) {
	b := bundleForValidate(t, "does-not-exist", nil)
	diags := Validate().Apply(t.Context(), b)
	require.Len(t, diags, 1)
	assert.Equal(t, `code_source_path "does-not-exist" not found`, diags[0].Summary)
}

func TestValidateGitSourceConflict(t *testing.T) {
	b := bundleForValidate(t, "src", &jobs.GitSource{GitUrl: "https://example.invalid/repo"})
	diags := Validate().Apply(t.Context(), b)
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Summary, "cannot be combined with git_source")
}

func TestValidateRemoteCodeSourceIsSkipped(t *testing.T) {
	b := bundleForValidate(t, "/Volumes/main/default/code/x.tar.gz", nil)
	diags := Validate().Apply(t.Context(), b)
	assert.Empty(t, diags)
}

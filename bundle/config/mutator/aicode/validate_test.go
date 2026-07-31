package aicode

import (
	"os"
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

// mkCodeDir creates a code_source directory (with one file) under the bundle's
// sync root, so the path resolves to an existing directory this mutator packages.
func mkCodeDir(t *testing.T, b *bundle.Bundle, rel string) {
	t.Helper()
	dir := filepath.Join(b.SyncRootPath, rel)
	require.NoError(t, os.MkdirAll(dir, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "train.py"), []byte("print()\n"), 0o600))
}

// A local path that is not an existing directory is left alone: it flows through
// the standard artifact-upload path (e.g. a pre-built tarball built by an
// `artifacts` block, which does not exist yet at validate time).
func TestValidateNonDirectoryCodeSourceIsSkipped(t *testing.T) {
	// Missing path (nothing on disk yet).
	b := bundleForValidate(t, "does-not-exist", nil)
	assert.Empty(t, Validate().Apply(t.Context(), b))

	// Existing local file (a pre-built tarball), not a directory.
	b = bundleForValidate(t, "code.tgz", nil)
	require.NoError(t, os.WriteFile(filepath.Join(b.SyncRootPath, "code.tgz"), []byte("x"), 0o600))
	assert.Empty(t, Validate().Apply(t.Context(), b))
}

func TestValidateGitSourceConflict(t *testing.T) {
	b := bundleForValidate(t, "src", &jobs.GitSource{GitUrl: "https://example.invalid/repo"})
	mkCodeDir(t, b, "src")
	diags := Validate().Apply(t.Context(), b)
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Summary, "cannot be combined with git_source")
}

func TestValidateRemoteCodeSourceIsSkipped(t *testing.T) {
	b := bundleForValidate(t, "/Volumes/main/default/code/x.tar.gz", nil)
	diags := Validate().Apply(t.Context(), b)
	assert.Empty(t, diags)
}

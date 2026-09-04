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

// A code_source_path escaping the bundle sync root is rejected with a clear
// message (it can't be synced), rather than failing later as an opaque io/fs error.
func TestValidateCodeSourceOutsideBundleRoot(t *testing.T) {
	b := bundleForValidate(t, "../shared", nil)
	diags := Validate().Apply(t.Context(), b)
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Summary, "outside the bundle root")
}

// Source-linked deployment doesn't copy files to the workspace file path, so the
// packaged snapshot would never be uploaded; the combination is rejected.
func TestValidateSourceLinkedConflict(t *testing.T) {
	b := bundleForValidate(t, "src", nil)
	mkCodeDir(t, b, "src")
	enabled := true
	b.Config.Presets.SourceLinkedDeployment = &enabled
	diags := Validate().Apply(t.Context(), b)
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Summary, "source-linked deployment")
}

// A local code_source_path nested under a for_each_task is not packaged by the
// mutator, so it is rejected rather than silently skipped.
func TestValidateForEachTaskCodeSourceRejected(t *testing.T) {
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
							Tasks: []jobs.Task{
								{
									TaskKey: "fanout",
									ForEachTask: &jobs.ForEachTask{
										Task: jobs.Task{
											AiRuntimeTask: &jobs.AiRuntimeTask{CodeSourcePath: "src"},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	bundletest.SetLocation(b, ".", []dyn.Location{{File: filepath.Join(dir, "databricks.yml")}})
	diags := Validate().Apply(t.Context(), b)
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Summary, "for_each_task")
}

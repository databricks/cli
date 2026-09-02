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
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// bundleWithCodeSource builds a bundle rooted at dir whose single AI Runtime task
// points at codeSourcePath.
//
// The end-to-end build/upload behavior (tarball built by artifacts.Build, uploaded by
// libraries) runs the full pipeline and is covered by acceptance tests under
// acceptance/bundle/ai_runtime_task. These unit tests cover the config-only seam:
// which paths are collected, and the artifact synthesis + code_source_path rewrite.
func bundleWithCodeSource(t *testing.T, dir, codeSourcePath string) *bundle.Bundle {
	t.Helper()
	b := &bundle.Bundle{
		BundleRootPath: dir,
		SyncRootPath:   dir,
		Config: config.Root{
			Bundle: config.Bundle{Target: "default"},
			Workspace: config.Workspace{
				CurrentUser: &config.User{User: &iam.User{UserName: "me@databricks.com"}},
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"train": {
						JobSettings: jobs.JobSettings{
							Tasks: []jobs.Task{
								{
									TaskKey:       "train",
									AiRuntimeTask: &jobs.AiRuntimeTask{Experiment: "exp", CodeSourcePath: codeSourcePath},
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

// A local-directory code_source_path is turned into a tgz artifact and the field is
// rewritten to the tarball that artifact builds. path/include are chosen so archive
// entries nest under the directory basename (the runtime's code_source layout), and
// the rewritten path equals the artifact's files.source so the upload rail links them.
func TestPackageCodeSourceSynthesizesArtifact(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "src"), 0o700))
	b := bundleWithCodeSource(t, dir, "./src")

	diags := PackageCodeSource().Apply(t.Context(), b)
	require.Empty(t, diags)

	outRel := ".databricks/air_code_source/air_code_source_src.tar.gz"
	a := b.Config.Artifacts["air_code_source_src"]
	require.NotNil(t, a)
	assert.Equal(t, config.ArtifactTarball, a.Type)
	// path/files are absolute: a synthesized artifact has no location for Prepare to
	// resolve relative paths against.
	assert.Equal(t, dir, a.Path)
	assert.Equal(t, []string{"src"}, a.Include)
	require.Len(t, a.Files, 1)
	assert.Equal(t, filepath.Join(dir, filepath.FromSlash(outRel)), a.Files[0].Source)

	// code_source_path is rewritten to the bundle-relative output, which resolves to
	// the same absolute file so the upload rail links them.
	assert.Equal(t, "./"+outRel, b.Config.Resources.Jobs["train"].Tasks[0].AiRuntimeTask.CodeSourcePath)
}

func TestCollectLocalCodeSourcesFindsLocalDir(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "src"), 0o700))
	b := bundleWithCodeSource(t, dir, "./src")
	sources, diags := collectLocalCodeSources(b)
	require.Empty(t, diags)
	require.Len(t, sources, 1)
	assert.Equal(t, "./src", sources[0].value)
}

func TestCollectLocalCodeSourcesSkipsRemotePaths(t *testing.T) {
	for _, remote := range []string{
		"/Workspace/Users/me/code.tar.gz",
		"/Volumes/main/default/code/existing.tar.gz",
	} {
		b := bundleWithCodeSource(t, t.TempDir(), remote)
		sources, diags := collectLocalCodeSources(b)
		require.Empty(t, diags)
		assert.Empty(t, sources, "remote code_source_path %q must not be collected", remote)
	}
}

// A local path that resolves to a file (not a directory) — e.g. a pre-built
// tarball delivered via an `artifacts` block — is NOT collected: it flows through
// the standard artifact-upload path as a file rather than being packaged here.
func TestCollectLocalCodeSourcesSkipsLocalFile(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "code.tgz"), []byte("x"), 0o600))
	b := bundleWithCodeSource(t, dir, "code.tgz")
	sources, diags := collectLocalCodeSources(b)
	require.Empty(t, diags)
	assert.Empty(t, sources, "a local tarball file must flow through artifact upload, not aicode packaging")
}

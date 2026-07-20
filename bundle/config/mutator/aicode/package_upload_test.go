package aicode

import (
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
// The end-to-end package/upload/rewrite behavior (local dir -> tarball -> upload ->
// rewritten code_source_path) runs the full mutator pipeline (sync file list,
// workspace filer) and is covered by acceptance tests under
// acceptance/bundle/ai_runtime_task. This unit test covers only the pure
// config-collection seam that does not touch the pipeline.
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

func TestCollectLocalCodeSourcesFindsLocalPath(t *testing.T) {
	b := bundleWithCodeSource(t, t.TempDir(), "./src")
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

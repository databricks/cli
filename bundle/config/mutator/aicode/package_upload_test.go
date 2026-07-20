package aicode

import (
	"strings"
	"testing"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/yamlloader"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// configWithCodeSource parses a minimal bundle config whose single AI Runtime task
// points at codeSourcePath, as a dyn value. It reads YAML directly so the config
// retains code_source_path regardless of whether the SDK's typed struct has it.
//
// The end-to-end package/upload/rewrite behavior (local dir -> tarball -> upload ->
// rewritten code_source_path) runs the full mutator pipeline (sync file list,
// workspace filer) and is covered by acceptance tests under
// acceptance/bundle/ai_runtime_task. This unit test covers only the pure
// config-collection seam that does not touch the pipeline.
func configWithCodeSource(t *testing.T, codeSourcePath string) dyn.Value {
	t.Helper()
	yml := `
resources:
  jobs:
    train:
      tasks:
        - task_key: train
          ai_runtime_task:
            experiment: exp
            code_source_path: ` + codeSourcePath + `
`
	v, err := yamlloader.LoadYAML("test.yml", strings.NewReader(yml))
	require.NoError(t, err)
	return v
}

func TestCollectLocalCodeSourcesFindsLocalPath(t *testing.T) {
	sources, diags := collectLocalCodeSources(configWithCodeSource(t, "./src"))
	require.Empty(t, diags)
	require.Len(t, sources, 1)
	assert.Equal(t, "./src", sources[0].value)
}

func TestCollectLocalCodeSourcesSkipsRemotePaths(t *testing.T) {
	for _, remote := range []string{
		"/Workspace/Users/me/code.tar.gz",
		"/Volumes/main/default/code/existing.tar.gz",
	} {
		sources, diags := collectLocalCodeSources(configWithCodeSource(t, remote))
		require.Empty(t, diags)
		assert.Empty(t, sources, "remote code_source_path %q must not be collected", remote)
	}
}

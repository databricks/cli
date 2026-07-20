package aicode

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/yamlloader"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// configForValidate parses a bundle config with one AI Runtime task pointing at
// codeSourcePath (and optionally a job git_source), as a dyn value. Reading YAML
// directly keeps code_source_path regardless of the SDK's typed struct.
func configForValidate(t *testing.T, codeSourcePath string, withGitSource bool) dyn.Value {
	t.Helper()
	git := ""
	if withGitSource {
		git = "\n      git_source:\n        git_url: https://example.invalid/repo\n        git_provider: gitHub"
	}
	yml := `
resources:
  jobs:
    train:` + git + `
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

// noLocations is a stub locations resolver for tests (diagnostics content, not
// locations, is what these assert).
func noLocations(string) []dyn.Location { return nil }

func TestValidateMissingCodeSourceDir(t *testing.T) {
	root := configForValidate(t, "does-not-exist", false)
	diags := validateCodeSources(root, t.TempDir(), false, noLocations)
	require.Len(t, diags, 1)
	assert.Equal(t, `code_source_path "does-not-exist" not found`, diags[0].Summary)
}

func TestValidateGitSourceConflict(t *testing.T) {
	root := configForValidate(t, "src", true)
	diags := validateCodeSources(root, t.TempDir(), false, noLocations)
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Summary, "cannot be combined with git_source")
}

func TestValidateImmutableFolderConflict(t *testing.T) {
	root := configForValidate(t, "src", false)
	diags := validateCodeSources(root, t.TempDir(), true, noLocations)
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Summary, "experimental.immutable_folder")
}

func TestValidateAcceptsExistingDir(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "src"), 0o755))
	root := configForValidate(t, "src", false)
	diags := validateCodeSources(root, dir, false, noLocations)
	assert.Empty(t, diags)
}

func TestValidateRemoteCodeSourceIsSkipped(t *testing.T) {
	root := configForValidate(t, "/Volumes/main/default/code/x.tar.gz", false)
	diags := validateCodeSources(root, t.TempDir(), false, noLocations)
	assert.Empty(t, diags)
}

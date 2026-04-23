package loader_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config"
	"github.com/databricks/cli/ucm/config/loader"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessRootIncludesEmpty(t *testing.T) {
	u := &ucm.Ucm{RootPath: "."}
	diags := ucm.Apply(t.Context(), u, loader.ProcessRootIncludes())
	require.NoError(t, diags.Error())
}

func TestProcessRootIncludesAbsRejected(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("windows absolute path check exercised elsewhere")
	}

	u := &ucm.Ucm{
		RootPath: ".",
		Config:   config.Root{Include: []string{"/tmp/*.yml"}},
	}
	diags := ucm.Apply(t.Context(), u, loader.ProcessRootIncludes())
	require.True(t, diags.HasError())
	assert.ErrorContains(t, diags.Error(), "must be relative paths")
}

func TestProcessRootIncludesSingleGlob(t *testing.T) {
	root := t.TempDir()
	testutil.Touch(t, root, "ucm.yml")
	testutil.Touch(t, root, "a.yml")
	testutil.Touch(t, root, "b.yml")

	u := &ucm.Ucm{RootPath: root, Config: config.Root{Include: []string{"*.yml"}}}
	diags := ucm.Apply(t.Context(), u, loader.ProcessRootIncludes())
	require.NoError(t, diags.Error())
	assert.Equal(t, []string{"a.yml", "b.yml"}, u.Config.Include)
}

func TestProcessRootIncludesMultiGlob(t *testing.T) {
	root := t.TempDir()
	testutil.Touch(t, root, "a1.yml")
	testutil.Touch(t, root, "b1.yml")

	u := &ucm.Ucm{RootPath: root, Config: config.Root{Include: []string{"a*.yml", "b*.yml"}}}
	diags := ucm.Apply(t.Context(), u, loader.ProcessRootIncludes())
	require.NoError(t, diags.Error())
	assert.Equal(t, []string{"a1.yml", "b1.yml"}, u.Config.Include)
}

func TestProcessRootIncludesDedup(t *testing.T) {
	root := t.TempDir()
	testutil.Touch(t, root, "a.yml")

	u := &ucm.Ucm{RootPath: root, Config: config.Root{Include: []string{"*.yml", "*.yml"}}}
	diags := ucm.Apply(t.Context(), u, loader.ProcessRootIncludes())
	require.NoError(t, diags.Error())
	assert.Equal(t, []string{"a.yml"}, u.Config.Include)
}

func TestProcessRootIncludesLiteralMissingIsError(t *testing.T) {
	u := &ucm.Ucm{RootPath: t.TempDir(), Config: config.Root{Include: []string{"notexist.yml"}}}
	diags := ucm.Apply(t.Context(), u, loader.ProcessRootIncludes())
	require.True(t, diags.HasError())
	assert.ErrorContains(t, diags.Error(), "notexist.yml defined in 'include' section does not match any files")
}

func TestProcessRootIncludesGlobMissingIsOk(t *testing.T) {
	u := &ucm.Ucm{RootPath: t.TempDir(), Config: config.Root{Include: []string{"*.yml"}}}
	diags := ucm.Apply(t.Context(), u, loader.ProcessRootIncludes())
	require.NoError(t, diags.Error())
	assert.Empty(t, u.Config.Include)
}

func TestProcessRootIncludesMergesFiles(t *testing.T) {
	root := t.TempDir()
	testutil.WriteFile(t, filepath.Join(root, "ucm.yml"), `
ucm: {name: main}
include:
  - parts/*.yml
`)
	testutil.WriteFile(t, filepath.Join(root, "parts", "a.yml"), `
workspace:
  host: https://a.example.com
`)
	testutil.WriteFile(t, filepath.Join(root, "parts", "b.yml"), `
resources:
  catalogs:
    c1: {name: c1}
`)

	cfg, diags := config.Load(filepath.Join(root, "ucm.yml"))
	require.NoError(t, diags.Error())
	u := &ucm.Ucm{RootPath: root, Config: *cfg}

	out := ucm.Apply(t.Context(), u, loader.ProcessRootIncludes())
	require.NoError(t, out.Error())
	assert.Equal(t, "https://a.example.com", u.Config.Workspace.Host)
	require.NotNil(t, u.Config.Resources.Catalogs)
	assert.Contains(t, u.Config.Resources.Catalogs, "c1")
	assert.Equal(t, []string{filepath.Join("parts", "a.yml"), filepath.Join("parts", "b.yml")}, u.Config.Include)
}

func TestProcessRootIncludesRejectsNonYAML(t *testing.T) {
	root := t.TempDir()
	testutil.Touch(t, root, "a.txt")

	u := &ucm.Ucm{RootPath: root, Config: config.Root{Include: []string{"a.txt"}}}
	diags := ucm.Apply(t.Context(), u, loader.ProcessRootIncludes())
	require.True(t, diags.HasError())
	assert.ErrorContains(t, diags.Error(), "is not a YAML or JSON file")
}

func TestProcessRootIncludesGlobInRootPath(t *testing.T) {
	tests := []struct {
		name string
		root string
		char string
	}{
		{"star", "foo/a*", "*"},
		{"question", "bar/?b", "?"},
		{"left-bracket", "[ab", "["},
		{"right-bracket", "ab]/bax", "]"},
		{"hat", "ab^bax", "^"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			u := &ucm.Ucm{RootPath: tc.root}
			diags := ucm.Apply(t.Context(), u, loader.ProcessRootIncludes())
			require.True(t, diags.HasError())
			require.Len(t, diags, 1)
			assert.Equal(t, diag.Error, diags[0].Severity)
			assert.Contains(t, diags[0].Detail, tc.char)
		})
	}
}

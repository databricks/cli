package artifacts

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/internal/bundletest"
	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandGlobs_Nominal(t *testing.T) {
	tmpDir := t.TempDir()

	testutil.Touch(t, tmpDir, "aa1.txt")
	testutil.Touch(t, tmpDir, "aa2.txt")
	testutil.Touch(t, tmpDir, "bb.txt")
	testutil.Touch(t, tmpDir, "bc.txt")

	b := &bundle.Bundle{
		BundleRootPath: tmpDir,
		Config: config.Root{
			Artifacts: config.Artifacts{
				"test": {
					Files: []config.ArtifactFile{
						{Source: "./aa*.txt"},
						{Source: "./b[bc].txt"},
					},
				},
			},
		},
	}

	bundletest.SetLocation(b, "artifacts", []dyn.Location{{File: filepath.Join(tmpDir, "databricks.yml")}})

	ctx := context.Background()
	diags := bundle.Apply(ctx, b, bundle.Seq(
		// Run prepare first to make paths absolute.
		&prepare{"test"},
		&expandGlobs{"test"},
	))
	require.NoError(t, diags.Error())

	// Assert that the expanded paths are correct.
	a, ok := b.Config.Artifacts["test"]
	if !assert.True(t, ok) {
		return
	}
	assert.Len(t, a.Files, 4)
	assert.Equal(t, filepath.Join(tmpDir, "aa1.txt"), a.Files[0].Source)
	assert.Equal(t, filepath.Join(tmpDir, "aa2.txt"), a.Files[1].Source)
	assert.Equal(t, filepath.Join(tmpDir, "bb.txt"), a.Files[2].Source)
	assert.Equal(t, filepath.Join(tmpDir, "bc.txt"), a.Files[3].Source)
}

func TestExpandGlobs_InvalidPattern(t *testing.T) {
	tmpDir := t.TempDir()

	b := &bundle.Bundle{
		BundleRootPath: tmpDir,
		Config: config.Root{
			Artifacts: config.Artifacts{
				"test": {
					Files: []config.ArtifactFile{
						{Source: "a[.txt"},
						{Source: "./a[.txt"},
						{Source: "../a[.txt"},
						{Source: "subdir/a[.txt"},
					},
				},
			},
		},
	}

	bundletest.SetLocation(b, "artifacts", []dyn.Location{{File: filepath.Join(tmpDir, "databricks.yml")}})

	ctx := context.Background()
	diags := bundle.Apply(ctx, b, bundle.Seq(
		// Run prepare first to make paths absolute.
		&prepare{"test"},
		&expandGlobs{"test"},
	))

	assert.Len(t, diags, 4)
	assert.Equal(t, filepath.Clean("a[.txt")+": syntax error in pattern", diags[0].Summary)
	assert.Equal(t, filepath.Join(tmpDir, "databricks.yml"), diags[0].Locations[0].File)
	assert.Equal(t, "artifacts.test.files[0].source", diags[0].Paths[0].String())
	assert.Equal(t, filepath.Clean("a[.txt")+": syntax error in pattern", diags[1].Summary)
	assert.Equal(t, filepath.Join(tmpDir, "databricks.yml"), diags[1].Locations[0].File)
	assert.Equal(t, "artifacts.test.files[1].source", diags[1].Paths[0].String())
	assert.Equal(t, filepath.Clean("../a[.txt")+": syntax error in pattern", diags[2].Summary)
	assert.Equal(t, filepath.Join(tmpDir, "databricks.yml"), diags[2].Locations[0].File)
	assert.Equal(t, "artifacts.test.files[2].source", diags[2].Paths[0].String())
	assert.Equal(t, filepath.Clean("subdir/a[.txt")+": syntax error in pattern", diags[3].Summary)
	assert.Equal(t, filepath.Join(tmpDir, "databricks.yml"), diags[3].Locations[0].File)
	assert.Equal(t, "artifacts.test.files[3].source", diags[3].Paths[0].String())
}

func TestExpandGlobs_NoMatches(t *testing.T) {
	tmpDir := t.TempDir()

	testutil.Touch(t, tmpDir, "a1.txt")
	testutil.Touch(t, tmpDir, "a2.txt")
	testutil.Touch(t, tmpDir, "b1.txt")
	testutil.Touch(t, tmpDir, "b2.txt")

	b := &bundle.Bundle{
		BundleRootPath: tmpDir,
		Config: config.Root{
			Artifacts: config.Artifacts{
				"test": {
					Files: []config.ArtifactFile{
						{Source: "a*.txt"},
						{Source: "b*.txt"},
						{Source: "c*.txt"},
						{Source: "d*.txt"},
					},
				},
			},
		},
	}

	bundletest.SetLocation(b, "artifacts", []dyn.Location{{File: filepath.Join(tmpDir, "databricks.yml")}})

	ctx := context.Background()
	diags := bundle.Apply(ctx, b, bundle.Seq(
		// Run prepare first to make paths absolute.
		&prepare{"test"},
		&expandGlobs{"test"},
	))

	assert.Len(t, diags, 2)
	assert.Equal(t, "c*.txt: no matching files", diags[0].Summary)
	assert.Equal(t, filepath.Join(tmpDir, "databricks.yml"), diags[0].Locations[0].File)
	assert.Equal(t, "artifacts.test.files[2].source", diags[0].Paths[0].String())
	assert.Equal(t, "d*.txt: no matching files", diags[1].Summary)
	assert.Equal(t, filepath.Join(tmpDir, "databricks.yml"), diags[1].Locations[0].File)
	assert.Equal(t, "artifacts.test.files[3].source", diags[1].Paths[0].String())

	// Assert that the original paths are unchanged.
	a, ok := b.Config.Artifacts["test"]
	if !assert.True(t, ok) {
		return
	}

	assert.Len(t, a.Files, 4)
	assert.Equal(t, "a*.txt", filepath.Base(a.Files[0].Source))
	assert.Equal(t, "b*.txt", filepath.Base(a.Files[1].Source))
	assert.Equal(t, "c*.txt", filepath.Base(a.Files[2].Source))
	assert.Equal(t, "d*.txt", filepath.Base(a.Files[3].Source))
}

package localenv

import (
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPipelineProjectEnvironmentRelativeStandalone(t *testing.T) {
	// uv resolves a relative UV_PROJECT_ENVIRONMENT against the project root.
	dir := writeProject(t)
	pm := &venvRecordingPM{fakePM: fakePM{py: "3.12", dbc: "17.2.0"}}
	p := newServerlessTestPipeline(t, dir)
	p.PM = pm
	ctx := env.Set(t.Context(), EnvUvProjectEnvironment, "custom-env")

	res, err := p.Run(ctx)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "custom-env"), pm.validateVenv)
	assert.Equal(t, filepath.Join(dir, "custom-env"), pm.postProvisionVenv)
	assert.Equal(t, "custom-env", res.VenvPath)
}

func TestPipelineProjectEnvironmentRelativeMember(t *testing.T) {
	// For a workspace member, uv resolves it against the workspace root.
	p, root, _ := newMemberPipeline(t)
	pm := &venvRecordingPM{fakePM: fakePM{py: "3.12", dbc: "17.2.0", root: root}}
	p.PM = pm
	ctx := env.Set(t.Context(), EnvUvProjectEnvironment, "custom-env")

	res, err := p.Run(ctx)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(root, "custom-env"), pm.validateVenv)
	assert.Equal(t, "../../custom-env", res.VenvPath)
}

func TestPipelineProjectEnvironmentAbsolute(t *testing.T) {
	dir := writeProject(t)
	abs := filepath.Join(t.TempDir(), "envs", "proj")
	pm := &venvRecordingPM{fakePM: fakePM{py: "3.12", dbc: "17.2.0"}}
	p := newServerlessTestPipeline(t, dir)
	p.PM = pm
	ctx := env.Set(t.Context(), EnvUvProjectEnvironment, abs)

	res, err := p.Run(ctx)
	require.NoError(t, err)
	assert.Equal(t, abs, pm.validateVenv)
	// venvPath stays relative to the project folder, as the OS resolves it.
	resolvedDir, err := filepath.EvalSymlinks(dir)
	require.NoError(t, err)
	want, err := filepath.Rel(resolvedDir, abs)
	require.NoError(t, err)
	assert.Equal(t, filepath.ToSlash(want), res.VenvPath)
}

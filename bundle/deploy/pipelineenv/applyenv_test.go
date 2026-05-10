package pipelineenv

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvInputHashPyproject(t *testing.T) {
	ctx := t.Context()
	root := t.TempDir()
	pyproject := filepath.Join(root, "pyproject.toml")
	require.NoError(t, os.WriteFile(pyproject, []byte("[project]\nname='x'\n"), 0o600))
	b := newBundle(root)

	h1, err := envInputHash(b)
	require.NoError(t, err)
	assert.NotEmpty(t, h1)

	require.NoError(t, writePrevHash(ctx, b, h1))
	assert.Equal(t, h1, readPrevHash(ctx, b))

	require.NoError(t, os.WriteFile(pyproject, []byte("[project]\nname='y'\n"), 0o600))
	h2, err := envInputHash(b)
	require.NoError(t, err)
	assert.NotEqual(t, h1, h2)
}

func TestEnvInputHashWheelContentChange(t *testing.T) {
	root := t.TempDir()
	wheelPath := filepath.Join(root, "envtest-0.0.1-py3-none-any.whl")
	require.NoError(t, os.WriteFile(wheelPath, []byte("wheel-bytes-v1"), 0o600))

	b := newBundle(root)
	b.Config.Artifacts = config.Artifacts{
		"w": {Type: config.ArtifactPythonWheel, Files: []config.ArtifactFile{{Source: wheelPath}}},
	}

	h1, err := envInputHash(b)
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(wheelPath, []byte("wheel-bytes-v2"), 0o600))
	h2, err := envInputHash(b)
	require.NoError(t, err)
	assert.NotEqual(t, h1, h2)
}

func TestEnvInputHashIgnoresJar(t *testing.T) {
	root := t.TempDir()
	jarPath := filepath.Join(root, "x.jar")
	require.NoError(t, os.WriteFile(jarPath, []byte("jar-bytes"), 0o600))

	b := newBundle(root)
	h1, err := envInputHash(b)
	require.NoError(t, err)

	b.Config.Artifacts = config.Artifacts{
		"j": {Type: config.ArtifactJar, Files: []config.ArtifactFile{{Source: jarPath}}},
	}
	h2, err := envInputHash(b)
	require.NoError(t, err)
	assert.Equal(t, h1, h2)
}

func TestEnvInputHashEmpty(t *testing.T) {
	h, err := envInputHash(newBundle(t.TempDir()))
	require.NoError(t, err)
	assert.NotEmpty(t, h) // sha256 of empty input is still defined
}

func newBundle(root string) *bundle.Bundle {
	return &bundle.Bundle{
		BundleRootPath: root,
		Config: config.Root{
			Bundle: config.Bundle{Target: "test"},
		},
	}
}

func TestPipelinesNeedingEnvApply(t *testing.T) {
	b := &bundle.Bundle{Config: config.Root{Resources: config.Resources{
		Pipelines: map[string]*resources.Pipeline{
			"a_dev_classic":    {BaseResource: resources.BaseResource{ID: "id1"}, CreatePipeline: pipelines.CreatePipeline{Name: "dev_classic", Development: true}},
			"b_dev_no_id":      {CreatePipeline: pipelines.CreatePipeline{Name: "dev_no_id", Development: true}},
			"c_prod_classic":   {BaseResource: resources.BaseResource{ID: "id3"}, CreatePipeline: pipelines.CreatePipeline{Name: "prod_classic"}},
			"d_dev_serverless": {BaseResource: resources.BaseResource{ID: "id4"}, CreatePipeline: pipelines.CreatePipeline{Name: "dev_serverless", Development: true, Serverless: true}},
			"e_nil":            nil,
		},
	}}}

	got := pipelinesNeedingEnvApply(b)
	require.Len(t, got, 1)
	assert.Equal(t, "dev_classic", got[0].Name)
}

func TestIsComputeNotRunning(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil"},
		{name: "plain error", err: assert.AnError},
		{name: "404 unrelated", err: &apierr.APIError{StatusCode: http.StatusNotFound, Message: "Pipeline 7742 does not exist"}},
		{name: "404 compute idle", err: &apierr.APIError{StatusCode: http.StatusNotFound, Message: "Pipeline compute for 7742 is not found. ..."}, want: true},
		{name: "500 with compute msg", err: &apierr.APIError{StatusCode: http.StatusInternalServerError, Message: "Pipeline compute exploded"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, isComputeNotRunning(tc.err))
		})
	}
}

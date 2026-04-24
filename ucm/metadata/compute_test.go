package metadata_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config"
	"github.com/databricks/cli/ucm/deploy"
	"github.com/databricks/cli/ucm/metadata"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestUcm(t *testing.T) *ucm.Ucm {
	t.Helper()
	u := &ucm.Ucm{RootPath: t.TempDir()}
	u.Config.Ucm = config.Ucm{Name: "demo", Target: "dev"}
	return u
}

// seedLocalStateID writes a ucm-state.json under the target's local cache dir
// so Compute's DeploymentID lookup succeeds without running a real Pull.
func seedLocalStateID(t *testing.T, u *ucm.Ucm, id uuid.UUID) {
	t.Helper()
	dir := deploy.LocalStateDir(u)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	blob, err := json.Marshal(map[string]any{
		"version": deploy.StateVersion,
		"seq":     3,
		"id":      id.String(),
	})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, deploy.UcmStateFileName), blob, 0o600))
}

func TestComputeReturnsZeroMetadataOnNilUcm(t *testing.T) {
	md := metadata.Compute(t.Context(), nil)
	assert.Equal(t, metadata.Metadata{}, md)
}

func TestComputePopulatesFields(t *testing.T) {
	u := newTestUcm(t)
	id := uuid.New()
	seedLocalStateID(t, u, id)

	md := metadata.Compute(t.Context(), u)

	assert.Equal(t, metadata.Version, md.Version)
	assert.Equal(t, build.GetInfo().Version, md.CliVersion)
	assert.Equal(t, "demo", md.Ucm.Name)
	assert.Equal(t, "dev", md.Ucm.Target)
	assert.Equal(t, id.String(), md.DeploymentID)
	assert.False(t, md.Timestamp.IsZero())
}

func TestComputeLeavesDeploymentIDEmptyOnMissingState(t *testing.T) {
	u := newTestUcm(t)

	md := metadata.Compute(t.Context(), u)

	assert.Empty(t, md.DeploymentID)
	assert.Equal(t, "demo", md.Ucm.Name)
	assert.Equal(t, "dev", md.Ucm.Target)
}

func TestComputeLeavesDeploymentIDEmptyOnMalformedState(t *testing.T) {
	cases := []struct {
		name      string
		stateJSON string
	}{
		{name: "not_json", stateJSON: "not-json"},
		{name: "id_is_nil_uuid", stateJSON: `{"id":"00000000-0000-0000-0000-000000000000"}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			u := newTestUcm(t)
			dir := deploy.LocalStateDir(u)
			require.NoError(t, os.MkdirAll(dir, 0o755))
			require.NoError(t, os.WriteFile(filepath.Join(dir, deploy.UcmStateFileName), []byte(tc.stateJSON), 0o600))

			md := metadata.Compute(t.Context(), u)

			assert.Empty(t, md.DeploymentID)
		})
	}
}

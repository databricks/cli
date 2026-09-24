package deployplan_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/stretchr/testify/require"
)

func TestPlanUploadManifestRoundTrip(t *testing.T) {
	plan := deployplan.NewPlanDirect()
	plan.Uploads = []deployplan.UploadManifestEntry{{
		Source:       "wheel/source.whl",
		RemoteName:   "source.whl",
		SHA256:       "abc123",
		PatchedWheel: true,
	}}
	data, err := json.Marshal(plan)
	require.NoError(t, err)

	path := filepath.Join(t.TempDir(), "plan.json")
	require.NoError(t, os.WriteFile(path, data, 0o600))
	loaded, err := deployplan.LoadPlanFromFile(path)
	require.NoError(t, err)
	require.Equal(t, plan.Uploads, loaded.Uploads)
}

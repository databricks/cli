package direct

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBindAndUnbindRejectDMSAuthoritativeState(t *testing.T) {
	for _, operation := range []string{"bind", "unbind"} {
		t.Run(operation, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "resources.json")
			state := dstate.NewDatabase("lineage", 1)
			state.Features = map[string]struct{}{dstate.FeatureDeploymentHistory: {}}
			data, err := json.Marshal(state)
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(path, data, 0o600))

			var db DeploymentBundle
			// Reject file-only operations before they need a resource API client.
			if operation == "bind" {
				_, err = db.Bind(t.Context(), nil, nil, path, "resources.jobs.foo", "123")
			} else {
				err = db.Unbind(t.Context(), path, "resources.jobs.foo")
			}
			require.ErrorContains(t, err, operation+" is not supported")
			assert.NoFileExists(t, path+".wal")
			persisted, err := os.ReadFile(path)
			require.NoError(t, err)
			assert.Equal(t, data, persisted)
		})
	}
}

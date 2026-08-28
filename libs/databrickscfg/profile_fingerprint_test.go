package databrickscfg_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/databrickscfg/profilehash"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFileProfilerCapturesFingerprintOfAllProfileFields verifies that loaded profiles
// include known and unknown configuration fields in their fingerprint.
func TestFileProfilerCapturesFingerprintOfAllProfileFields(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), ".databrickscfg")
	t.Setenv("DATABRICKS_CONFIG_FILE", configPath)
	require.NoError(t, os.WriteFile(configPath, []byte(`[TEST]
host = https://workspace.example.com
auth_type = databricks-cli
custom_key = custom-value
`), 0o600))

	want := profilehash.Compute(map[string]string{
		"host":       "https://workspace.example.com",
		"auth_type":  "databricks-cli",
		"custom_key": "custom-value",
	})

	profiles, err := profile.DefaultProfiler.LoadProfiles(t.Context(), profile.WithName("TEST"))
	require.NoError(t, err)
	require.Len(t, profiles, 1)

	assert.Equal(t, want, profiles[0].Fingerprint())
}

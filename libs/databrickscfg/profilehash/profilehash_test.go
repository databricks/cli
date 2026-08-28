package profilehash

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCompute verifies that profile fingerprints are independent of map iteration order
// but change when a parsed profile value changes.
func TestCompute(t *testing.T) {
	base := map[string]string{
		"host":       "https://workspace.example.com",
		"auth_type":  "databricks-cli",
		"custom_key": "custom-value",
	}
	tests := []struct {
		name      string
		values    map[string]string
		wantEqual bool
	}{
		{
			name: "map order does not matter",
			values: map[string]string{
				"custom_key": "custom-value",
				"auth_type":  "databricks-cli",
				"host":       "https://workspace.example.com",
			},
			wantEqual: true,
		},
		{
			name: "changed value changes fingerprint",
			values: map[string]string{
				"host":       "https://workspace.example.com",
				"auth_type":  "databricks-cli",
				"custom_key": "changed-value",
			},
			wantEqual: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantEqual {
				assert.Equal(t, Compute(base), Compute(tt.values))
				return
			}

			assert.NotEqual(t, Compute(base), Compute(tt.values))
		})
	}
}

// TestFromFileIgnoresFormatting verifies that formatting-only profile edits do not
// invalidate cached credentials.
func TestFromFileIgnoresFormatting(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".databrickscfg")
	require.NoError(t, os.WriteFile(path, []byte(`[TEST]
host = https://workspace.example.com
scopes = jobs
`), 0o600))
	first, err := FromFile(path, "TEST")
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(path, []byte(`[TEST]
# Reordering keys and adding comments does not change profile semantics.
scopes=jobs
host=https://workspace.example.com
`), 0o600))
	second, err := FromFile(path, "TEST")
	require.NoError(t, err)

	assert.Equal(t, first, second)
}

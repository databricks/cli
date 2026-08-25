package protos

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBundleDeployExperimentalSaferrFields pins the wire names of the PII-free
// error descriptions. They are the only fields here that a dashboard groups by,
// so a renamed or duplicated tag would silently stop populating a column rather
// than fail anything.
func TestBundleDeployExperimentalSaferrFields(t *testing.T) {
	raw, err := json.Marshal(BundleDeployExperimental{
		DirectMigrateSaferr:        "a",
		DirectMigrateCommitSaferr:  "b",
		DirectMigrateWarningSaferr: "c",
	})
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(raw, &got))

	assert.Equal(t, "a", got["direct_migrate_saferr"])
	assert.Equal(t, "b", got["direct_migrate_commit_saferr"])
	assert.Equal(t, "c", got["direct_migrate_warning_saferr"])
}

// TestBundleDeployExperimentalSaferrOmitted keeps a successful deploy from
// carrying three empty strings.
func TestBundleDeployExperimentalSaferrOmitted(t *testing.T) {
	raw, err := json.Marshal(BundleDeployExperimental{})
	require.NoError(t, err)

	assert.NotContains(t, string(raw), "saferr")
}

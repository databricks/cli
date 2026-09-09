package protos

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBundleDeployExperimentalSafeErrFields pins the wire names of the PII-free
// error descriptions. They are the only fields here that a dashboard groups by,
// so a renamed or duplicated tag would silently stop populating a column rather
// than fail anything.
func TestBundleDeployExperimentalSafeErrFields(t *testing.T) {
	raw, err := json.Marshal(BundleDeployExperimental{
		DirectMigrateSafeErr:        "a",
		DirectMigrateCommitSafeErr:  "b",
		DirectMigrateWarningSafeErr: "c",
		DirectMigratePlanSafeErr:    "d",
	})
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(raw, &got))

	assert.Equal(t, "a", got["direct_migrate_safe_error"])
	assert.Equal(t, "b", got["direct_migrate_commit_safe_error"])
	assert.Equal(t, "c", got["direct_migrate_warning_safe_error"])
	assert.Equal(t, "d", got["direct_migrate_plan_safe_error"])
}

// TestBundleDeployExperimentalSafeErrOmitted keeps a successful deploy from
// carrying three empty strings.
func TestBundleDeployExperimentalSafeErrOmitted(t *testing.T) {
	raw, err := json.Marshal(BundleDeployExperimental{})
	require.NoError(t, err)

	assert.NotContains(t, string(raw), "safe_error")
}

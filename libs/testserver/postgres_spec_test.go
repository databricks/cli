package testserver

import (
	"testing"

	"github.com/databricks/databricks-sdk-go/service/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEchoedSpecCreateDropsInputOnly(t *testing.T) {
	spec := echoedSpec[postgres.ProjectSpec](nil, []byte(`{"spec":{"display_name":"a","pg_version":16}}`), "", projectOneofGroups, "pg_version")
	require.NotNil(t, spec)
	assert.Equal(t, "a", spec.DisplayName)
	assert.Equal(t, 0, spec.PgVersion)
}

func TestEchoedSpecCreateKeepsExplicitFalse(t *testing.T) {
	spec := echoedSpec[postgres.BranchSpec](nil, []byte(`{"spec":{"is_protected":false,"no_expiry":true}}`), "", branchOneofGroups)
	require.NotNil(t, spec)
	assert.Contains(t, spec.ForceSendFields, "IsProtected")
	assert.True(t, spec.NoExpiry)
}

func TestEchoedSpecUpdateOneofReplacesSibling(t *testing.T) {
	stored := echoedSpec[postgres.BranchSpec](nil, []byte(`{"spec":{"is_protected":true,"no_expiry":true}}`), "", branchOneofGroups)
	spec := echoedSpec(stored, []byte(`{"spec":{"ttl":"3600s"}}`), "spec.expiration", branchOneofGroups)
	require.NotNil(t, spec)
	assert.False(t, spec.NoExpiry)
	require.NotNil(t, spec.Ttl)
	assert.True(t, spec.IsProtected, "fields outside the mask keep their stored value")
}

func TestEchoedSpecUpdateMaskedButAbsentKeepsPrevious(t *testing.T) {
	stored := echoedSpec[postgres.ProjectSpec](nil, []byte(`{"spec":{"display_name":"a","history_retention_duration":"604800s"}}`), "", projectOneofGroups)
	spec := echoedSpec(stored, []byte(`{"spec":{"display_name":"b"}}`), "spec", projectOneofGroups)
	require.NotNil(t, spec)
	assert.Equal(t, "b", spec.DisplayName)
	require.NotNil(t, spec.HistoryRetentionDuration)
}

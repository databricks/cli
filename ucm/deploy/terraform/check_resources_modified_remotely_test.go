package terraform_test

import (
	"testing"

	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/deploy/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckResourcesModifiedRemotelyNoOpWithEmptyAllowlist(t *testing.T) {
	u := &ucm.Ucm{}
	diags := ucm.Apply(t.Context(), u, terraform.CheckResourcesModifiedRemotely(nil))
	require.NoError(t, diags.Error())
	assert.Empty(t, diags)
}

func TestCheckResourcesModifiedRemotelyScaffoldCallsWithKindsButHasNoLogicYet(t *testing.T) {
	u := &ucm.Ucm{}
	diags := ucm.Apply(t.Context(), u, terraform.CheckResourcesModifiedRemotely([]string{"catalog"}))
	require.NoError(t, diags.Error())
	assert.Empty(t, diags)
}

func TestCheckResourcesModifiedRemotelyName(t *testing.T) {
	m := terraform.CheckResourcesModifiedRemotely(nil)
	assert.Equal(t, "CheckResourcesModifiedRemotely", m.Name())
}

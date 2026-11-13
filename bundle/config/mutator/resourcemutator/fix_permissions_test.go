package resourcemutator

import (
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetLevelScore(t *testing.T) {
	assert.Equal(t, 17, resources.GetLevelScore("CAN_MANAGE"))
	assert.Equal(t, 0, resources.GetLevelScore("UNKNOWN_PERMISSION"))
	assert.Equal(t, resources.GetLevelScore("CAN_MANAGE"), resources.GetLevelScore("CAN_MANAGE_SOMETHING_ELSE"))
	assert.Equal(t, resources.GetLevelScore("CAN_MANAGE_RUN"), resources.GetLevelScore("CAN_MANAGE_RUN1"))
}

func TestGetMaxLevel(t *testing.T) {
	assert.Equal(t, "IS_OWNER", resources.GetMaxLevel("IS_OWNER", "CAN_MANAGE"))
	assert.Equal(t, "IS_OWNER", resources.GetMaxLevel("CAN_MANAGE", "IS_OWNER"))
	assert.Equal(t, "CAN_MANAGE", resources.GetMaxLevel("CAN_MANAGE", "CAN_EDIT"))
	assert.Equal(t, "CAN_EDIT", resources.GetMaxLevel("CAN_READ", "CAN_EDIT"))

	assert.Equal(t, "CAN_MANAGE", resources.GetMaxLevel("CAN_MANAGE", "CAN_MANAGE"))
	assert.Equal(t, "CAN_READ", resources.GetMaxLevel("CAN_READ", ""))
	assert.Equal(t, "CAN_READ", resources.GetMaxLevel("", "CAN_READ"))
	assert.Empty(t, resources.GetMaxLevel("", ""))

	assert.Equal(t, "UNKNOWN_B", resources.GetMaxLevel("UNKNOWN_A", "UNKNOWN_B"))
	assert.Equal(t, "UNKNOWN_B", resources.GetMaxLevel("UNKNOWN_B", "UNKNOWN_A"))
}

func TestUseMaximumLevelKeepsUnchangedEntries(t *testing.T) {
	loc := []diag.Location{{File: "databricks.yml", Line: 10}}
	viewer := dyn.NewValue(map[string]dyn.Value{"level": dyn.V("CAN_VIEW"), "user_name": dyn.V("viewer@example.com")}, loc)
	raised := dyn.NewValue(map[string]dyn.Value{"level": dyn.V("CAN_VIEW"), "group_name": dyn.V("users")}, loc)
	duplicate := dyn.V(map[string]dyn.Value{"level": dyn.V("CAN_MANAGE"), "group_name": dyn.V("users")})

	result, err := useMaximumLevel(dyn.V([]dyn.Value{viewer, raised, duplicate}), "jobs")
	require.NoError(t, err)

	permissions := result.MustSequence()
	require.Len(t, permissions, 2)
	assert.Equal(t, loc, permissions[0].Locations())
	assert.Empty(t, permissions[1].Locations())
	assert.Equal(t, "CAN_MANAGE", permissions[1].Get("level").MustString())
}

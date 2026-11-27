package resourcemutator

import (
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/stretchr/testify/assert"
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

func TestProcessPermissionsKeepsDeclaredEntries(t *testing.T) {
	viewer := permission{level: "CAN_VIEW", userName: "viewer@example.com"}
	lower := permission{level: "CAN_VIEW", groupName: "users"}
	higher := permission{level: "CAN_MANAGE", groupName: "users"}

	result, sources := processPermissions("me@example.com", "jobs", []permission{viewer, lower, higher})

	assert.Equal(t, []permission{viewer, higher, {level: isOwner, userName: "me@example.com"}}, result)
	// The kept entries come from inputs 0 and 2; the added owner is new.
	assert.Equal(t, []int{0, 2, -1}, sources)
}

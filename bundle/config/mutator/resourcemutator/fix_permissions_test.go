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
	assert.Equal(t, "", resources.GetMaxLevel("", ""))

	assert.Equal(t, "UNKNOWN_B", resources.GetMaxLevel("UNKNOWN_A", "UNKNOWN_B"))
	assert.Equal(t, "UNKNOWN_B", resources.GetMaxLevel("UNKNOWN_B", "UNKNOWN_A"))
}

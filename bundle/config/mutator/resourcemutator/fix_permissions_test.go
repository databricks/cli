package resourcemutator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetLevelScore(t *testing.T) {
	assert.Equal(t, 17, getLevelScore("CAN_MANAGE"))
	assert.Equal(t, 0, getLevelScore("UNKNOWN_PERMISSION"))
	assert.Equal(t, getLevelScore("CAN_MANAGE"), getLevelScore("CAN_MANAGE_SOMETHING_ELSE"))
	assert.Equal(t, getLevelScore("CAN_MANAGE_RUN"), getLevelScore("CAN_MANAGE_RUN1"))
}

func TestGetMaxLevel(t *testing.T) {
	assert.Equal(t, "IS_OWNER", getMaxLevel("IS_OWNER", "CAN_MANAGE"))
	assert.Equal(t, "IS_OWNER", getMaxLevel("CAN_MANAGE", "IS_OWNER"))
	assert.Equal(t, "CAN_MANAGE", getMaxLevel("CAN_MANAGE", "CAN_EDIT"))
	assert.Equal(t, "CAN_EDIT", getMaxLevel("CAN_READ", "CAN_EDIT"))

	assert.Equal(t, "CAN_MANAGE", getMaxLevel("CAN_MANAGE", "CAN_MANAGE"))
	assert.Equal(t, "CAN_READ", getMaxLevel("CAN_READ", ""))
	assert.Equal(t, "CAN_READ", getMaxLevel("", "CAN_READ"))
	assert.Equal(t, "", getMaxLevel("", ""))

	assert.Equal(t, "UNKNOWN_B", getMaxLevel("UNKNOWN_A", "UNKNOWN_B"))
	assert.Equal(t, "UNKNOWN_B", getMaxLevel("UNKNOWN_B", "UNKNOWN_A"))
}

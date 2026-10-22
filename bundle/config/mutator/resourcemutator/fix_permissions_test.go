package resourcemutator

import (
	"strings"
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
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

func TestFixPermissionsResourcePolicy(t *testing.T) {
	owners := map[string]bool{"jobs": true, "pipelines": true, "sql_warehouses": true}
	ignored := map[string]bool{"secret_scopes": true, "cluster_policies": true}
	for key := range config.ResourcesTypes {
		name, ok := strings.CutSuffix(key, ".permissions")
		if !ok {
			continue
		}
		t.Run(name, func(t *testing.T) {
			path := dyn.NewPath(dyn.Key("resources"), dyn.Key(name), dyn.Key("test"), dyn.Key("permissions"))
			result, err := processPermissions("deployer")(path, dyn.V([]dyn.Value{}))
			require.NoError(t, err)
			values, ok := result.AsSequence()
			require.True(t, ok)
			if ignored[name] {
				assert.Empty(t, values)
				return
			}
			require.Len(t, values, 1)
			level := "CAN_MANAGE"
			if owners[name] {
				level = "IS_OWNER"
			}
			assert.Equal(t, level, dyn.GetValue(values[0], "level").MustString())
			assert.Equal(t, "deployer", dyn.GetValue(values[0], "user_name").MustString())
		})
	}
}

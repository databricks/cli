package config

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

func TestResourcesTypesMap(t *testing.T) {
	assert.Greater(t, len(ResourcesTypes), 10, "expected ResourcesTypes to have more than 10 entries")

	typ, ok := ResourcesTypes["jobs"]
	assert.True(t, ok, "resources type for 'jobs' not found in ResourcesTypes map")
	assert.Equal(t, reflect.TypeOf(resources.Job{}), typ, "resources type for 'jobs' mismatch")

	typ, ok = ResourcesTypes["jobs.permissions"]
	assert.True(t, ok, "resources type for 'jobs.permissions' not found in ResourcesTypes map")
	assert.Equal(t, reflect.TypeOf(resources.Permissions[jobs.JobPermissionLevel]{}), typ, "resources type for 'jobs.permissions' mismatch")
}

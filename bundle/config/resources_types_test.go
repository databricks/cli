package config

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/databricks/cli/bundle/config/resources"
)

func TestResourcesTypesMap(t *testing.T) {
	assert.Greater(t, len(ResourcesTypes), 10, "expected ResourcesTypes to have more than 10 entries")

	typ, ok := ResourcesTypes["jobs"]
	assert.True(t, ok, "resources type for 'jobs' not found in ResourcesTypes map")
	assert.Equal(t, reflect.TypeOf(resources.Job{}), typ, "resources type for 'jobs' mismatch")
}

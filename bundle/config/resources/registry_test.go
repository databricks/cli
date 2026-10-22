package resources_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistryMatchesResourceTypes(t *testing.T) {
	typ := reflect.TypeFor[config.Resources]()
	require.Len(t, resources.All, typ.NumField())

	for field := range typ.Fields() {
		name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		description, ok := resources.All[name]
		require.True(t, ok, "missing descriptor for %s", name)
		assert.Equal(t, name, description.PluralName)

		resourceType := field.Type.Elem().Elem()
		resource := reflect.New(resourceType).Interface().(config.ConfigResource)
		assert.Equal(t, resource.ResourceDescription(), description, name)

		_, hasPermissions := resourceType.FieldByName("Permissions")
		assert.Equal(t, hasPermissions, description.Permissions != (resources.PermissionsDescription{}), name)
	}
}

func TestRegistryIdentityAndPermissionRouting(t *testing.T) {
	singulars := make(map[string]string)
	requestTypes := make(map[string]string)
	for name, description := range resources.All {
		assert.NotEmpty(t, description.SingularName, name)
		assert.NotEmpty(t, description.SingularTitle, name)
		assert.NotEmpty(t, description.PluralTitle, name)
		assert.Equal(t, name, description.PluralName)
		assert.NotContains(t, singulars, description.SingularName, name)
		singulars[description.SingularName] = name

		p := description.Permissions
		if p.RequestObjectType == "" {
			assert.Empty(t, p.ResponseObjectType, name)
			assert.Empty(t, p.ObjectIDField, name)
			continue
		}
		assert.NotEmpty(t, p.ResponseObjectType, name)
		assert.NotEmpty(t, p.ObjectIDField, name)
		assert.Equal(t, strings.Trim(p.RequestObjectType, "/"), p.RequestObjectType, name)
		assert.NotContains(t, requestTypes, p.RequestObjectType, name)
		requestTypes[p.RequestObjectType] = name
	}
}

func TestDescriptorCopiesDoNotChangeRegistry(t *testing.T) {
	original := resources.All["jobs"]
	first := config.SupportedResources()
	description := first["jobs"]
	description.Permissions.BundleLevels.CanManage = "CAN_VIEW"
	first["jobs"] = description
	delete(first, "features")

	assert.Equal(t, original, resources.All["jobs"])
	assert.Equal(t, original, (&resources.Job{}).ResourceDescription())
	assert.Equal(t, resources.All, config.SupportedResources())
}

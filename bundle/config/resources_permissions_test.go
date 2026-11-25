package config

import (
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/stretchr/testify/assert"
)

// Shared code finds a type's permissions support only through its declaration, so a type
// with a permissions field that declares nothing would silently get no permissions applied.
func TestPermissionsDeclaredIffPermissionsField(t *testing.T) {
	for _, rt := range AllResourceTypes() {
		_, hasField := ResourcesTypes()[rt.Key+".permissions"]
		_, declares := ResourceTypeAs[resources.PermissionsDeclarer](rt.Key)
		if hasField {
			assert.True(t, declares, "resource type %q has a permissions field but does not implement resources.PermissionsDeclarer", rt.Key)
		} else {
			assert.False(t, declares, "resource type %q implements resources.PermissionsDeclarer but has no permissions field", rt.Key)
		}
	}
}

func TestPermissionsSupportIsConsistent(t *testing.T) {
	for key, d := range ResourceTypesImplementing[resources.PermissionsDeclarer]() {
		s := d.PermissionsSupport()
		usesAPI := s.RequestObjectType != ""
		assert.Equal(t, usesAPI, s.ResponseObjectType != "", "%s: ResponseObjectType must be set exactly when RequestObjectType is", key)
		assert.Equal(t, usesAPI, s.ObjectIDField != "", "%s: ObjectIDField must be set exactly when RequestObjectType is", key)
		assert.False(t, s.HasIsOwner && s.SkipFixup, "%s: HasIsOwner has no effect with SkipFixup", key)
		assert.True(t, usesAPI || s.BundleLevels != resources.BundlePermissionLevels{}, "%s: declares neither a permissions API nor bundle-level permissions", key)
	}
}

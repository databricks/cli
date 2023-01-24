package sync

import (
	"testing"

	"github.com/databricks/databricks-sdk-go/service/scim"
	"github.com/stretchr/testify/assert"
)

func TestPathNestedUnderBasePaths(t *testing.T) {
	me := scim.User{
		UserName: "jane@doe.com",
	}

	// Not nested under allowed base paths.
	assert.Error(t, checkPathNestedUnderBasePaths(&me, "/Repos/jane@doe.com"))
	assert.Error(t, checkPathNestedUnderBasePaths(&me, "/Repos/jane@doe.com/."))
	assert.Error(t, checkPathNestedUnderBasePaths(&me, "/Repos/jane@doe.com/.."))
	assert.Error(t, checkPathNestedUnderBasePaths(&me, "/Repos/john@doe.com"))
	assert.Error(t, checkPathNestedUnderBasePaths(&me, "/Repos/jane@doe.comsuffix/foo"))
	assert.Error(t, checkPathNestedUnderBasePaths(&me, "/Repos/"))
	assert.Error(t, checkPathNestedUnderBasePaths(&me, "/Repos"))
	assert.Error(t, checkPathNestedUnderBasePaths(&me, "/Users/jane@doe.com"))
	assert.Error(t, checkPathNestedUnderBasePaths(&me, "/Users/jane@doe.com/."))
	assert.Error(t, checkPathNestedUnderBasePaths(&me, "/Users/jane@doe.com/.."))
	assert.Error(t, checkPathNestedUnderBasePaths(&me, "/Users/john@doe.com"))
	assert.Error(t, checkPathNestedUnderBasePaths(&me, "/Users/jane@doe.comsuffix/foo"))
	assert.Error(t, checkPathNestedUnderBasePaths(&me, "/Users/"))
	assert.Error(t, checkPathNestedUnderBasePaths(&me, "/Users"))
	assert.Error(t, checkPathNestedUnderBasePaths(&me, "/"))

	// Nested under allowed base paths.
	assert.NoError(t, checkPathNestedUnderBasePaths(&me, "/Repos/jane@doe.com/foo"))
	assert.NoError(t, checkPathNestedUnderBasePaths(&me, "/Repos/jane@doe.com/./foo"))
	assert.NoError(t, checkPathNestedUnderBasePaths(&me, "/Repos/jane@doe.com/foo/bar/qux"))
	assert.NoError(t, checkPathNestedUnderBasePaths(&me, "/Users/jane@doe.com/foo"))
	assert.NoError(t, checkPathNestedUnderBasePaths(&me, "/Users/jane@doe.com/./foo"))
	assert.NoError(t, checkPathNestedUnderBasePaths(&me, "/Users/jane@doe.com/foo/bar/qux"))
}

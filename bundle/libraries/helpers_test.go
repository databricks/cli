package libraries

import (
	"testing"

	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/stretchr/testify/assert"
)

func TestLibraryPath(t *testing.T) {
	path := "/some/path"

	assert.Equal(t, path, libraryPath(&compute.Library{Whl: path}))
	assert.Equal(t, path, libraryPath(&compute.Library{Jar: path}))
	assert.Equal(t, path, libraryPath(&compute.Library{Egg: path}))
	assert.Equal(t, "", libraryPath(&compute.Library{}))
}

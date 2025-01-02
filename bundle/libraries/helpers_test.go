package libraries

import (
	"testing"

	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/stretchr/testify/assert"
)

func TestLibraryPath(t *testing.T) {
	path := "/some/path"

	p, err := libraryPath(&compute.Library{Whl: path})
	assert.Equal(t, path, p)
	assert.NoError(t, err)

	p, err = libraryPath(&compute.Library{Jar: path})
	assert.Equal(t, path, p)
	assert.NoError(t, err)

	p, err = libraryPath(&compute.Library{Egg: path})
	assert.Equal(t, path, p)
	assert.NoError(t, err)

	p, err = libraryPath(&compute.Library{Requirements: path})
	assert.Equal(t, path, p)
	assert.NoError(t, err)

	p, err = libraryPath(&compute.Library{})
	assert.Equal(t, "", p)
	assert.Error(t, err)

	p, err = libraryPath(&compute.Library{Pypi: &compute.PythonPyPiLibrary{Package: "pypipackage"}})
	assert.Equal(t, "", p)
	assert.Error(t, err)
}

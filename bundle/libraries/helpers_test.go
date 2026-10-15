package libraries

import (
	"testing"

	"github.com/databricks/cli/libs/sdktypes/jobs/v2"

	"github.com/stretchr/testify/assert"
)

func TestLibraryPath(t *testing.T) {
	path := "/some/path"

	p, err := libraryPath(&jobs.Library{Whl: new(path)})
	assert.Equal(t, path, p)
	assert.NoError(t, err)

	p, err = libraryPath(&jobs.Library{Jar: new(path)})
	assert.Equal(t, path, p)
	assert.NoError(t, err)

	p, err = libraryPath(&jobs.Library{Egg: new(path)})
	assert.Equal(t, path, p)
	assert.NoError(t, err)

	p, err = libraryPath(&jobs.Library{Requirements: new(path)})
	assert.Equal(t, path, p)
	assert.NoError(t, err)

	p, err = libraryPath(&jobs.Library{})
	assert.Empty(t, p)
	assert.Error(t, err)

	p, err = libraryPath(&jobs.Library{Pypi: &jobs.PythonPyPiLibrary{Package: new("pypipackage")}})
	assert.Empty(t, p)
	assert.Error(t, err)
}

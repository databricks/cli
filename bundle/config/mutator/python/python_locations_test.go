package python

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeLocations(t *testing.T) {
	pythonLocation := diag.Location{File: "foo.py", Line: 1, Column: 1}
	generatedLocation := diag.Location{File: generatedFileName, Line: 1, Column: 1}
	yamlLocation := diag.Location{File: "foo.yml", Line: 1, Column: 1}

	locations := newPythonLocations()
	putPythonLocation(locations, structpath.MustParsePath("foo"), pythonLocation)

	mapper := mergePythonLocations(locations)
	at := func(path string, locs ...diag.Location) []diag.Location {
		var p *structpath.PathNode
		if path != "" {
			p = structpath.MustParsePath(path)
		}
		return mapper(p, locs)
	}

	// pythonLocation is prepended if absent
	assert.Equal(t, []diag.Location{pythonLocation, yamlLocation}, at("foo.baz", yamlLocation))
	// generatedLocation is replaced by pythonLocation
	assert.Equal(t, []diag.Location{pythonLocation, yamlLocation}, at("foo.qux", generatedLocation, yamlLocation))
	assert.Equal(t, []diag.Location{pythonLocation}, at("foo"))
	// if location is unknown, we keep it as-is
	assert.Equal(t, []diag.Location{generatedLocation}, at("bar", generatedLocation))
	assert.Equal(t, []diag.Location{yamlLocation}, at("", yamlLocation))
}

func TestFindLocation(t *testing.T) {
	location0 := diag.Location{File: "foo.py", Line: 1, Column: 1}
	location1 := diag.Location{File: "foo.py", Line: 2, Column: 1}

	locations := newPythonLocations()
	putPythonLocation(locations, structpath.MustParsePath("foo"), location0)
	putPythonLocation(locations, structpath.MustParsePath("foo.bar"), location1)

	actual, exists := findPythonLocation(locations, structpath.MustParsePath("foo.bar"))

	assert.True(t, exists)
	assert.Equal(t, location1, actual)
}

func TestFindLocation_indexPathComponent(t *testing.T) {
	location0 := diag.Location{File: "foo.py", Line: 1, Column: 1}
	location1 := diag.Location{File: "foo.py", Line: 2, Column: 1}
	location2 := diag.Location{File: "foo.py", Line: 3, Column: 1}

	locations := newPythonLocations()
	putPythonLocation(locations, structpath.MustParsePath("foo"), location0)
	putPythonLocation(locations, structpath.MustParsePath("foo.bar"), location1)
	putPythonLocation(locations, structpath.MustParsePath("foo.bar[0]"), location2)

	actual, exists := findPythonLocation(locations, structpath.MustParsePath("foo.bar[0]"))

	assert.True(t, exists)
	assert.Equal(t, location2, actual)
}

func TestFindLocation_closestAncestorLocation(t *testing.T) {
	location0 := diag.Location{File: "foo.py", Line: 1, Column: 1}
	location1 := diag.Location{File: "foo.py", Line: 2, Column: 1}

	locations := newPythonLocations()
	putPythonLocation(locations, structpath.MustParsePath("foo"), location0)
	putPythonLocation(locations, structpath.MustParsePath("foo.bar"), location1)

	actual, exists := findPythonLocation(locations, structpath.MustParsePath("foo.bar.baz"))

	assert.True(t, exists)
	assert.Equal(t, location1, actual)
}

func TestFindLocation_unknownLocation(t *testing.T) {
	location0 := diag.Location{File: "foo.py", Line: 1, Column: 1}
	location1 := diag.Location{File: "foo.py", Line: 2, Column: 1}

	locations := newPythonLocations()
	putPythonLocation(locations, structpath.MustParsePath("foo"), location0)
	putPythonLocation(locations, structpath.MustParsePath("foo.bar"), location1)

	_, exists := findPythonLocation(locations, structpath.MustParsePath("bar"))

	assert.False(t, exists)
}

func TestLoadOutput(t *testing.T) {
	location := diag.Location{File: "my_job.py", Line: 1, Column: 1}
	bundleRoot := t.TempDir()
	output := `{
		"resources": {
			"jobs": {
				"my_job": {
					"name": "my_job",
					"tasks": [
						{
							"task_key": "my_task",
							"notebook_task": {
								"notebook_path": "my_notebook"
							}
						}
					]
				}
			}
		}
	}`

	locations := newPythonLocations()
	putPythonLocation(
		locations,
		structpath.MustParsePath("resources.jobs.my_job"),
		location,
	)

	root, diags := loadOutput(
		bundleRoot,
		bytes.NewReader([]byte(output)),
		locations,
	)

	assert.Equal(t, diag.Diagnostics{}, diags)
	require.Equal(t, []diag.Location{location}, root.LocationsAt(structpath.MustParsePath("resources.jobs.my_job.name")))
}

func TestParsePythonLocations_absolutePath(t *testing.T) {
	// output can contain absolute path that is outside of the bundle root
	expected := diag.Location{File: "/Shared/foo.py", Line: 1, Column: 2}

	input := `{"path": "foo", "file": "/Shared/foo.py", "line": 1, "column": 2}`
	reader := bytes.NewReader([]byte(input))
	locations, err := parsePythonLocations("/tmp/", reader)

	assert.NoError(t, err)

	assert.True(t, locations.keys["foo"].exists)
	assert.Equal(t, expected, locations.keys["foo"].location)
}

func TestParsePythonLocations_relativePath(t *testing.T) {
	// output can contain relative paths, we expect all locations to be absolute
	// at this stage of mutator pipeline
	expected := diag.Location{File: filepath.Clean("/tmp/my_project/foo.py"), Line: 1, Column: 2}

	input := `{"path": "foo", "file": "foo.py", "line": 1, "column": 2}`
	reader := bytes.NewReader([]byte(input))
	locations, err := parsePythonLocations(filepath.Clean("/tmp/my_project"), reader)

	assert.NoError(t, err)

	assert.True(t, locations.keys["foo"].exists)
	assert.Equal(t, expected, locations.keys["foo"].location)
}

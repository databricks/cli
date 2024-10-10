package python

import (
	"bytes"
	"testing"

	"github.com/databricks/cli/libs/dyn"
	assert "github.com/databricks/cli/libs/dyn/dynassert"
)

func TestMergeLocations(t *testing.T) {
	pythonLocation := dyn.Location{File: "foo.py", Line: 1, Column: 1}
	generatedLocation := dyn.Location{File: generatedFileName, Line: 1, Column: 1}
	yamlLocation := dyn.Location{File: "foo.yml", Line: 1, Column: 1}

	locations := newPythonLocations()
	putPythonLocation(locations, dyn.MustPathFromString("foo"), pythonLocation)

	input := dyn.NewValue(
		map[string]dyn.Value{
			"foo": dyn.NewValue(
				map[string]dyn.Value{
					"baz": dyn.NewValue("baz", []dyn.Location{yamlLocation}),
					"qux": dyn.NewValue("baz", []dyn.Location{generatedLocation, yamlLocation}),
				},
				[]dyn.Location{},
			),
			"bar": dyn.NewValue("baz", []dyn.Location{generatedLocation}),
		},
		[]dyn.Location{yamlLocation},
	)

	expected := dyn.NewValue(
		map[string]dyn.Value{
			"foo": dyn.NewValue(
				map[string]dyn.Value{
					// pythonLocation is appended to the beginning of the list if absent
					"baz": dyn.NewValue("baz", []dyn.Location{pythonLocation, yamlLocation}),
					// generatedLocation is replaced by pythonLocation
					"qux": dyn.NewValue("baz", []dyn.Location{pythonLocation, yamlLocation}),
				},
				[]dyn.Location{pythonLocation},
			),
			// if location is unknown, we keep it as-is
			"bar": dyn.NewValue("baz", []dyn.Location{generatedLocation}),
		},
		[]dyn.Location{yamlLocation},
	)

	actual, err := mergePythonLocations(input, locations)

	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestFindLocation(t *testing.T) {
	location0 := dyn.Location{File: "foo.py", Line: 1, Column: 1}
	location1 := dyn.Location{File: "foo.py", Line: 2, Column: 1}

	locations := newPythonLocations()
	putPythonLocation(locations, dyn.MustPathFromString("foo"), location0)
	putPythonLocation(locations, dyn.MustPathFromString("foo.bar"), location1)

	actual, exists := findPythonLocation(locations, dyn.MustPathFromString("foo.bar"))

	assert.True(t, exists)
	assert.Equal(t, location1, actual)
}

func TestFindLocation_indexPathComponent(t *testing.T) {
	location0 := dyn.Location{File: "foo.py", Line: 1, Column: 1}
	location1 := dyn.Location{File: "foo.py", Line: 2, Column: 1}
	location2 := dyn.Location{File: "foo.py", Line: 3, Column: 1}

	locations := newPythonLocations()
	putPythonLocation(locations, dyn.MustPathFromString("foo"), location0)
	putPythonLocation(locations, dyn.MustPathFromString("foo.bar"), location1)
	putPythonLocation(locations, dyn.MustPathFromString("foo.bar[0]"), location2)

	actual, exists := findPythonLocation(locations, dyn.MustPathFromString("foo.bar[0]"))

	assert.True(t, exists)
	assert.Equal(t, location2, actual)
}

func TestFindLocation_closestAncestorLocation(t *testing.T) {
	location0 := dyn.Location{File: "foo.py", Line: 1, Column: 1}
	location1 := dyn.Location{File: "foo.py", Line: 2, Column: 1}

	locations := newPythonLocations()
	putPythonLocation(locations, dyn.MustPathFromString("foo"), location0)
	putPythonLocation(locations, dyn.MustPathFromString("foo.bar"), location1)

	actual, exists := findPythonLocation(locations, dyn.MustPathFromString("foo.bar.baz"))

	assert.True(t, exists)
	assert.Equal(t, location1, actual)
}

func TestFindLocation_unknownLocation(t *testing.T) {
	location0 := dyn.Location{File: "foo.py", Line: 1, Column: 1}
	location1 := dyn.Location{File: "foo.py", Line: 2, Column: 1}

	locations := newPythonLocations()
	putPythonLocation(locations, dyn.MustPathFromString("foo"), location0)
	putPythonLocation(locations, dyn.MustPathFromString("foo.bar"), location1)

	_, exists := findPythonLocation(locations, dyn.MustPathFromString("bar"))

	assert.False(t, exists)
}

func TestParsePythonLocations(t *testing.T) {
	expected := dyn.Location{File: "foo.py", Line: 1, Column: 2}

	input := `{"path": "foo", "file": "foo.py", "line": 1, "column": 2}`
	reader := bytes.NewReader([]byte(input))
	locations, err := parsePythonLocations(reader)

	assert.NoError(t, err)

	assert.True(t, locations.keys["foo"].exists)
	assert.Equal(t, expected, locations.keys["foo"].location)
}

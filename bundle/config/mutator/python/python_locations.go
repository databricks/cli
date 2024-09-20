package python

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"

	"github.com/databricks/cli/libs/dyn"
)

const generatedFileName = "__generated_by_pydabs__.yml"

// pythonLocations is data structure for efficient location lookup for a given path
type pythonLocations struct {
	// descendants referenced by index, e.g. '.foo'
	keys map[string]*pythonLocations

	// descendants referenced by key, e.g. '[0]'
	indexes map[int]*pythonLocations

	// location for the current node if it exists
	location dyn.Location

	// if true, location is present
	exists bool
}

type pythonLocationEntry struct {
	Path   string `json:"path"`
	File   string `json:"file"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
}

// mergePythonLocations applies locations from Python mutator into given dyn.Value
//
// The primary use-case is to merge locations.json with output.json, so that any
// validation errors will point to Python source code instead of generated YAML.
func mergePythonLocations(value dyn.Value, locations *pythonLocations) (dyn.Value, error) {
	return dyn.Walk(value, func(path dyn.Path, value dyn.Value) (dyn.Value, error) {
		if newLocation, ok := findPythonLocation(locations, path); ok {
			var newLocations []dyn.Location

			// the first item in the list is the "last" location used for error reporting
			newLocations = append(newLocations, newLocation)

			for _, location := range value.Locations() {
				if filepath.Base(location.File) == generatedFileName {
					continue
				}

				// don't add duplicates if location already exists
				if location == newLocation {
					continue
				}

				newLocations = append(newLocations, location)
			}

			return value.WithLocations(newLocations), nil
		} else {
			return value, nil
		}
	})
}

// parsePythonLocations parses locations.json from the Python mutator.
//
// locations file is newline-separated JSON objects with pythonLocationEntry structure.
func parsePythonLocations(input io.Reader) (*pythonLocations, error) {
	decoder := json.NewDecoder(input)
	locations := newPythonLocations()

	for decoder.More() {
		var entry pythonLocationEntry

		err := decoder.Decode(&entry)
		if err != nil {
			return nil, fmt.Errorf("failed to parse python location: %s", err)
		}

		path, err := dyn.NewPathFromString(entry.Path)
		if err != nil {
			return nil, fmt.Errorf("failed to parse python location: %s", err)
		}

		location := dyn.Location{
			File:   entry.File,
			Line:   entry.Line,
			Column: entry.Column,
		}

		putPythonLocation(locations, path, location)
	}

	return locations, nil
}

// putPythonLocation puts the location to the trie for the given path
func putPythonLocation(trie *pythonLocations, path dyn.Path, location dyn.Location) {
	var currentNode = trie

	for _, component := range path {
		if key := component.Key(); key != "" {
			if _, ok := currentNode.keys[key]; !ok {
				currentNode.keys[key] = newPythonLocations()
			}

			currentNode = currentNode.keys[key]
		} else {
			index := component.Index()
			if _, ok := currentNode.indexes[index]; !ok {
				currentNode.indexes[index] = newPythonLocations()
			}

			currentNode = currentNode.indexes[index]
		}
	}

	currentNode.location = location
	currentNode.exists = true
}

// newPythonLocations creates a new trie node
func newPythonLocations() *pythonLocations {
	return &pythonLocations{
		keys:    make(map[string]*pythonLocations),
		indexes: make(map[int]*pythonLocations),
	}
}

// findPythonLocation finds the location or closest ancestor location in the trie for the given path
// if no ancestor or exact location is found, false is returned.
func findPythonLocation(locations *pythonLocations, path dyn.Path) (dyn.Location, bool) {
	var currentNode = locations
	var lastLocation = locations.location
	var exists = locations.exists

	for _, component := range path {
		if key := component.Key(); key != "" {
			if _, ok := currentNode.keys[key]; !ok {
				break
			}

			currentNode = currentNode.keys[key]
		} else {
			index := component.Index()
			if _, ok := currentNode.indexes[index]; !ok {
				break
			}

			currentNode = currentNode.indexes[index]
		}

		if currentNode.exists {
			lastLocation = currentNode.location
			exists = true
		}
	}

	return lastLocation, exists
}

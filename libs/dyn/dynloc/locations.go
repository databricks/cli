package dynloc

import (
	"fmt"
	"path/filepath"
	"slices"
	"sort"

	"github.com/databricks/cli/libs/dyn"
	"golang.org/x/exp/maps"
)

const (
	// Version is the version of the location information structure.
	// Increment if the structure changes.
	Version = 1
)

// Locations is a structure that holds location information for (a subset of) a [dyn.Value] value.
type Locations struct {
	// Version is the version of the location information.
	Version int `json:"version"`

	// Files is a list of file paths.
	Files []string `json:"files"`

	// Locations maps the string representation of a [dyn.Path] to a list of 3-tuples that represent the index
	// of the file in the [Files] array, followed by the line and column number.
	// A single [dyn.Path] can have multiple locations (e.g. the effective location and original definition).
	Locations map[string][][]int `json:"locations"`

	// fileToIndex maps file paths to their index in the [Files] array.
	// This is used to avoid duplicate entries in the [Files] array and keep the
	// map with locations as compact as possible.
	fileToIndex map[string]int

	// basePath is the base path used to compute relative paths.
	basePath string
}

func (l *Locations) gatherLocations(v dyn.Value) (map[string][]dyn.Location, error) {
	locs := map[string][]dyn.Location{}
	patterns := []dyn.Pattern{
		dyn.NewPattern(dyn.AnyKey()),                                                                          // Top level fields
		dyn.NewPattern(dyn.Key("resources"), dyn.AnyKey()),                                                    // Resource groups ("resources.jobs")
		dyn.NewPattern(dyn.Key("resources"), dyn.AnyKey(), dyn.AnyKey()),                                      // Resources for all types ("resources.jobs.my_job")
		dyn.NewPattern(dyn.Key("resources"), dyn.Key("jobs"), dyn.AnyKey(), dyn.Key("tasks")),                 // Job tasks ("resources.jobs.my_job.tasks")
		dyn.NewPattern(dyn.Key("resources"), dyn.Key("jobs"), dyn.AnyKey(), dyn.Key("tasks"), dyn.AnyIndex()), // Job task items ("resources.jobs.my_job.tasks[2]")
	}

	for _, pattern := range patterns {
		_, err := dyn.MapByPattern(v, pattern, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			locs[p.String()] = v.Locations()
			return v, nil
		})
		if err != nil {
			return nil, err
		}
	}

	return locs, nil
}

func (l *Locations) normalizeFilePath(file string) (string, error) {
	var err error

	// Compute the relative path. The base path may be empty.
	file, err = filepath.Rel(l.basePath, file)
	if err != nil {
		return "", err
	}

	// Convert the path separator to forward slashes.
	// This makes it possible to compare output across platforms.
	return filepath.ToSlash(file), nil
}

func (l *Locations) registerFileNames(locs []dyn.Location) error {
	cache := map[string]string{}
	for _, loc := range locs {
		// Never process the same file path twice.
		if _, ok := cache[loc.File]; ok {
			continue
		}

		// Normalize the file path.
		out, err := l.normalizeFilePath(loc.File)
		if err != nil {
			return err
		}

		// Cache the normalized path.
		cache[loc.File] = out
	}

	l.Files = maps.Values(cache)
	sort.Strings(l.Files)

	// Build the file-to-index map.
	for i, file := range l.Files {
		l.fileToIndex[file] = i
	}

	// Add entries for the original file path.
	// Doing this means we can perform the lookup with the verbatim file path.
	for k, v := range cache {
		l.fileToIndex[k] = l.fileToIndex[v]
	}

	return nil
}

func (l *Locations) addLocation(path, file string, line, col int) error {
	// Expect the file to be present in the lookup map.
	if _, ok := l.fileToIndex[file]; !ok {
		// This indicates a logic problem below, but we rather not panic.
		return fmt.Errorf("dynloc: unknown file %q", file)
	}

	// Add the location to the map.
	l.Locations[path] = append(
		l.Locations[path],
		[]int{l.fileToIndex[file], line, col},
	)

	return nil
}

// Build constructs a [Locations] object from a [dyn.Value].
func Build(v dyn.Value, basePath string) (Locations, error) {
	l := Locations{
		Version:   Version,
		Files:     make([]string, 0),
		Locations: make(map[string][][]int),

		// Internal state.
		fileToIndex: make(map[string]int),
		basePath:    basePath,
	}

	// Traverse the value and collect locations.
	pathToLocations, err := l.gatherLocations(v)
	if err != nil {
		return l, err
	}

	// Normalize file paths and add locations.
	// This step adds files to the [Files] array in alphabetical order.
	err = l.registerFileNames(slices.Concat(maps.Values(pathToLocations)...))
	if err != nil {
		return l, err
	}

	// Add locations to the map.
	for path, locs := range pathToLocations {
		for _, loc := range locs {
			err = l.addLocation(path, loc.File, loc.Line, loc.Column)
			if err != nil {
				return l, err
			}
		}
	}

	return l, err
}

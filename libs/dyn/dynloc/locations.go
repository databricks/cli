package dynloc

import (
	"path/filepath"

	"github.com/databricks/cli/libs/dyn"
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

	// maxDepth is the maximum depth of the [dyn.Path] keys in the [Locations] map.
	maxDepth int

	// basePath is the base path used to compute relative paths.
	basePath string
}

func (l *Locations) addLocation(p dyn.Path, file string, line, col int) error {
	var err error

	// Compute the relative path. The base path may be empty.
	file, err = filepath.Rel(l.basePath, file)
	if err != nil {
		return err
	}

	// Convert the path separator to forward slashes.
	// This makes it possible to compare output across platforms.
	file = filepath.ToSlash(file)

	// If the file is not yet in the list, add it.
	if _, ok := l.fileToIndex[file]; !ok {
		l.fileToIndex[file] = len(l.Files)
		l.Files = append(l.Files, file)
	}

	// Add the location to the map.
	l.Locations[p.String()] = append(
		l.Locations[p.String()],
		[]int{l.fileToIndex[file], line, col},
	)

	return nil
}

// Option is a functional option for the [Build] function.
type Option func(l *Locations)

// WithMaxDepth sets the maximum depth of the [dyn.Path] keys in the [Locations] map.
func WithMaxDepth(depth int) Option {
	return func(l *Locations) {
		l.maxDepth = depth
	}
}

// WithBasePath sets the base path used to compute relative paths.
func WithBasePath(basePath string) Option {
	return func(l *Locations) {
		l.basePath = basePath
	}
}

// Build constructs a [Locations] object from a [dyn.Value].
func Build(v dyn.Value, opts ...Option) (Locations, error) {
	l := Locations{
		Version:   Version,
		Files:     make([]string, 0),
		Locations: make(map[string][][]int),

		// Internal state.
		fileToIndex: make(map[string]int),
	}

	// Apply options.
	for _, opt := range opts {
		opt(&l)
	}

	// Traverse the value and collect locations.
	_, err := dyn.Walk(v, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
		// Skip the root value.
		if len(p) == 0 {
			return v, nil
		}

		// Skip if the path depth exceeds the maximum depth.
		if l.maxDepth > 0 && len(p) > l.maxDepth {
			return v, dyn.ErrSkip
		}

		for _, loc := range v.Locations() {
			err := l.addLocation(p, loc.File, loc.Line, loc.Column)
			if err != nil {
				return dyn.InvalidValue, err
			}
		}

		return v, nil
	})

	return l, err
}

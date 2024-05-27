package dyn

import (
	"fmt"
	"path/filepath"
)

type Location struct {
	File   string
	Line   int
	Column int
}

// nilLocation is a convenient constant for a nil location.
// TODO: Remove this constant and rely on the file path in the location?
var nilLocation = Location{}

func (l Location) String() string {
	return fmt.Sprintf("%s:%d:%d", l.File, l.Line, l.Column)
}

func (l Location) Directory() (string, error) {
	if l.File == "" {
		return "", fmt.Errorf("no file in location")
	}

	return filepath.Dir(l.File), nil
}

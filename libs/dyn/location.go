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

// Convenient constant for an empty location.
var EmptyLocation = Location{}

func (l Location) String() string {
	return fmt.Sprintf("%s:%d:%d", l.File, l.Line, l.Column)
}

func (l Location) Directory() (string, error) {
	if l.File == "" {
		return "", fmt.Errorf("no file in location")
	}

	return filepath.Dir(l.File), nil
}

package diag

import (
	"fmt"
	"path/filepath"
)

type Location struct {
	File   string
	Line   int
	Column int
}

func (l Location) String() string {
	return fmt.Sprintf("%s:%d:%d", filepath.ToSlash(l.File), l.Line, l.Column)
}

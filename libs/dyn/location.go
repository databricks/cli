package dyn

import (
	"fmt"
)

type Location struct {
	File   string
	Line   int
	Column int
}

func (l Location) String() string {
	return fmt.Sprintf("%s:%d:%d", l.File, l.Line, l.Column)
}

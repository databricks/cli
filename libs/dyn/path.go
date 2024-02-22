package dyn

import (
	"bytes"
	"fmt"
)

type pathComponent struct {
	key   string
	index int
}

func (c pathComponent) Key() string {
	return c.key
}

func (c pathComponent) Index() int {
	return c.index
}

func (c pathComponent) isKey() bool {
	return c.key != ""
}

func (c pathComponent) isIndex() bool {
	return c.key == ""
}

// Path represents a path to a value in a [Value] configuration tree.
type Path []pathComponent

// EmptyPath is the empty path.
// It is defined for convenience and clarity.
var EmptyPath = Path{}

// Key returns a path component for a key.
func Key(k string) pathComponent {
	return pathComponent{key: k}
}

// Index returns a path component for an index.
func Index(i int) pathComponent {
	return pathComponent{index: i}
}

// NewPath returns a new path from the given components.
// The individual components may be created with [Key] or [Index].
func NewPath(cs ...pathComponent) Path {
	return cs
}

// Join joins the given paths.
func (p Path) Join(qs ...Path) Path {
	for _, q := range qs {
		p = p.Append(q...)
	}
	return p
}

// Append appends the given components to the path.
func (p Path) Append(cs ...pathComponent) Path {
	return append(p, cs...)
}

// Equal returns true if the paths are equal.
func (p Path) Equal(q Path) bool {
	pl := len(p)
	ql := len(q)
	if pl != ql {
		return false
	}
	for i := 0; i < pl; i++ {
		if p[i] != q[i] {
			return false
		}
	}
	return true
}

// HasPrefix returns true if the path has the specified prefix.
// The empty path is a prefix of all paths.
func (p Path) HasPrefix(q Path) bool {
	pl := len(p)
	ql := len(q)
	if pl < ql {
		return false
	}
	for i := 0; i < ql; i++ {
		if p[i] != q[i] {
			return false
		}
	}
	return true
}

// String returns a string representation of the path.
func (p Path) String() string {
	var buf bytes.Buffer

	for i, c := range p {
		if i > 0 && c.key != "" {
			buf.WriteRune('.')
		}
		if c.key != "" {
			buf.WriteString(c.key)
		} else {
			buf.WriteString(fmt.Sprintf("[%d]", c.index))
		}
	}

	return buf.String()
}

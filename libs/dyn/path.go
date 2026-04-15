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

// Append appends the given components to the path.
// Mutations to the returned path do not affect the original path.
func (p Path) Append(cs ...pathComponent) Path {
	out := make(Path, len(p)+len(cs))
	copy(out, p)
	copy(out[len(p):], cs)
	return out
}

// Equal returns true if the paths are equal.
func (p Path) Equal(q Path) bool {
	pl := len(p)
	ql := len(q)
	if pl != ql {
		return false
	}
	for i := range pl {
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
	for i := range ql {
		if p[i] != q[i] {
			return false
		}
	}
	return true
}

// HasSuffix returns true if the path has the specified suffix.
// The empty path is a suffix of all paths.
func (p Path) HasSuffix(q Path) bool {
	pl := len(p)
	ql := len(q)
	if pl < ql {
		return false
	}
	for i := range ql {
		if p[pl-ql+i] != q[i] {
			return false
		}
	}
	return true
}

// CutPrefix returns the path with the specified prefix removed.
// If the path does not have the specified prefix, the original path is returned.
// The second return value is true if the prefix was removed.
// Logically equivalent to [strings.CutPrefix].
func (p Path) CutPrefix(q Path) (Path, bool) {
	if !p.HasPrefix(q) {
		return p, false
	}
	return p[len(q):], true
}

// CutSuffix returns the path with the specified suffix removed.
// If the path does not have the specified suffix, the original path is returned.
// The second return value is true if the suffix was removed.
// Logically equivalent to [strings.CutSuffix].
func (p Path) CutSuffix(q Path) (Path, bool) {
	if !p.HasSuffix(q) {
		return p, false
	}
	return p[:len(p)-len(q)], true
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

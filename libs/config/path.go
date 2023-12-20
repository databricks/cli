package config

import (
	"bytes"
	"fmt"
)

type pathComponent struct {
	key   string
	index int
}

type Path []pathComponent

var EmptyPath = Path{}

func Key(k string) pathComponent {
	return pathComponent{key: k}
}

func Index(i int) pathComponent {
	return pathComponent{index: i}
}

func NewPath(cs ...pathComponent) Path {
	return cs
}

func (p Path) Join(qs ...Path) Path {
	for _, q := range qs {
		p = p.Append(q...)
	}
	return p
}

func (p Path) Append(cs ...pathComponent) Path {
	return append(p, cs...)
}

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

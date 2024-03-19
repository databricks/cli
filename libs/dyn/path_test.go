package dyn_test

import (
	"testing"

	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
)

func TestPathAppend(t *testing.T) {
	p := dyn.NewPath(dyn.Key("foo"))

	// Single arg.
	p1 := p.Append(dyn.Key("bar"))
	assert.True(t, p1.Equal(dyn.NewPath(dyn.Key("foo"), dyn.Key("bar"))))

	// Multiple args.
	p2 := p.Append(dyn.Key("bar"), dyn.Index(1))
	assert.True(t, p2.Equal(dyn.NewPath(dyn.Key("foo"), dyn.Key("bar"), dyn.Index(1))))
}

func TestPathAppendAlwaysNew(t *testing.T) {
	p := make(dyn.Path, 0, 2)
	p = append(p, dyn.Key("foo"))

	// There is room for a second element in the slice.
	p1 := p.Append(dyn.Index(1))
	p2 := p.Append(dyn.Index(2))
	assert.NotEqual(t, p1, p2)
}

func TestPathEqualEmpty(t *testing.T) {
	assert.True(t, dyn.EmptyPath.Equal(dyn.EmptyPath))
}

func TestPathEqual(t *testing.T) {
	p1 := dyn.NewPath(dyn.Key("foo"), dyn.Index(1))
	p2 := dyn.NewPath(dyn.Key("bar"), dyn.Index(2))
	assert.False(t, p1.Equal(p2), "expected %q to not equal %q", p1, p2)

	p3 := dyn.NewPath(dyn.Key("foo"), dyn.Index(1))
	assert.True(t, p1.Equal(p3), "expected %q to equal %q", p1, p3)

	p4 := dyn.NewPath(dyn.Key("foo"), dyn.Index(1), dyn.Key("bar"), dyn.Index(2))
	assert.False(t, p1.Equal(p4), "expected %q to not equal %q", p1, p4)
}

func TestPathHasPrefixEmpty(t *testing.T) {
	empty := dyn.EmptyPath
	nonEmpty := dyn.NewPath(dyn.Key("foo"))
	assert.True(t, empty.HasPrefix(empty))
	assert.True(t, nonEmpty.HasPrefix(empty))
	assert.False(t, empty.HasPrefix(nonEmpty))
}

func TestPathHasPrefix(t *testing.T) {
	p1 := dyn.NewPath(dyn.Key("foo"), dyn.Index(1))
	p2 := dyn.NewPath(dyn.Key("bar"), dyn.Index(2))
	assert.False(t, p1.HasPrefix(p2), "expected %q to not have prefix %q", p1, p2)

	p3 := dyn.NewPath(dyn.Key("foo"))
	assert.True(t, p1.HasPrefix(p3), "expected %q to have prefix %q", p1, p3)
}

func TestPathString(t *testing.T) {
	p1 := dyn.NewPath(dyn.Key("foo"), dyn.Index(1))
	assert.Equal(t, "foo[1]", p1.String())

	p2 := dyn.NewPath(dyn.Key("bar"), dyn.Index(2), dyn.Key("baz"))
	assert.Equal(t, "bar[2].baz", p2.String())

	p3 := dyn.NewPath(dyn.Key("foo"), dyn.Index(1), dyn.Key("bar"), dyn.Index(2), dyn.Key("baz"))
	assert.Equal(t, "foo[1].bar[2].baz", p3.String())
}

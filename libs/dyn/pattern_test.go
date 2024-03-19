package dyn_test

import (
	"testing"

	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
)

func TestNewPattern(t *testing.T) {
	pat := dyn.NewPattern(
		dyn.Key("foo"),
		dyn.Index(1),
	)

	assert.Len(t, pat, 2)
}

func TestNewPatternFromPath(t *testing.T) {
	path := dyn.NewPath(
		dyn.Key("foo"),
		dyn.Index(1),
	)

	pat1 := dyn.NewPattern(dyn.Key("foo"), dyn.Index(1))
	pat2 := dyn.NewPatternFromPath(path)
	assert.Equal(t, pat1, pat2)
}

func TestPatternAppend(t *testing.T) {
	p := dyn.NewPattern(dyn.Key("foo"))

	// Single arg.
	p1 := p.Append(dyn.Key("bar"))
	assert.Equal(t, dyn.NewPattern(dyn.Key("foo"), dyn.Key("bar")), p1)

	// Multiple args.
	p2 := p.Append(dyn.Key("bar"), dyn.Index(1))
	assert.Equal(t, dyn.NewPattern(dyn.Key("foo"), dyn.Key("bar"), dyn.Index(1)), p2)
}

func TestPatternAppendAlwaysNew(t *testing.T) {
	p := make(dyn.Pattern, 0, 2)
	p = append(p, dyn.Key("foo"))

	// There is room for a second element in the slice.
	p1 := p.Append(dyn.Index(1))
	p2 := p.Append(dyn.Index(2))
	assert.NotEqual(t, p1, p2)
}

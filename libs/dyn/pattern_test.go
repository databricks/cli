package dyn_test

import (
	"fmt"
	"testing"

	"github.com/databricks/cli/libs/dyn"
	assert "github.com/databricks/cli/libs/dyn/dynassert"
)

func TestPatternString(t *testing.T) {
	patterns := []string{
		"foo.bar",
		"foo.bar.baz",
		"foo[1]",
		"foo[*]",
		"*[1].bar",
		"foo.*.bar",
		"foo[*].bar",
		"",
		"foo",
		"[1]",
		"[*]",
		"*",
	}

	for _, p := range patterns {
		pp := dyn.MustPatternFromString(p)
		assert.Equal(t, p, pp.String())
	}
}

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

func TestPatternSplitKey(t *testing.T) {
	p := dyn.NewPattern(
		dyn.Key("foo"),
		dyn.Key("bar"),
	)

	pat, key := p.SplitKey()
	assert.Equal(t, "bar", key)
	assert.Equal(t, dyn.NewPattern(dyn.Key("foo")), pat)
}

func TestPatternSplitKeyError(t *testing.T) {
	patterns := []dyn.Pattern{
		dyn.NewPattern(
			dyn.Key("foo"),
			dyn.AnyKey(),
		),
		dyn.NewPattern(
			dyn.Key("foo"),
			dyn.AnyIndex(),
		),
		dyn.NewPattern(
			dyn.Key("foo"),
			dyn.Index(1),
		),
		dyn.NewPattern(),
	}

	for ind, p := range patterns {
		t.Run(fmt.Sprintf("%d %#v", ind, p), func(t *testing.T) {
			pat, key := p.SplitKey()
			assert.Equal(t, "", key)
			assert.Empty(t, pat)
		})
	}
}

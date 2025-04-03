package dyn_test

import (
	"testing"

	"github.com/databricks/cli/libs/dyn"
	assert "github.com/databricks/cli/libs/dyn/dynassert"
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

	p3 := dyn.NewPath(dyn.Key("foo"), dyn.Index(1), dyn.Key("bar"))
	assert.False(t, p1.HasPrefix(p3), "expected %q to not have prefix %q", p1, p3)

	p4 := dyn.NewPath(dyn.Key("foo"), dyn.Index(1))
	assert.True(t, p1.HasPrefix(p4), "expected %q to have prefix %q", p1, p4)

	p5 := dyn.NewPath(dyn.Key("foo"))
	assert.True(t, p1.HasPrefix(p5), "expected %q to have prefix %q", p1, p5)
}

func TestPathHasSuffixEmpty(t *testing.T) {
	empty := dyn.EmptyPath
	nonEmpty := dyn.NewPath(dyn.Key("foo"))
	assert.True(t, empty.HasSuffix(empty))
	assert.True(t, nonEmpty.HasSuffix(empty))
	assert.False(t, empty.HasSuffix(nonEmpty))
}

func TestPathHasSuffix(t *testing.T) {
	p1 := dyn.NewPath(dyn.Key("foo"), dyn.Index(1))
	p2 := dyn.NewPath(dyn.Key("bar"), dyn.Index(2))
	assert.False(t, p1.HasSuffix(p2), "expected %q to not have suffix %q", p1, p2)

	p3 := dyn.NewPath(dyn.Index(1))
	assert.True(t, p1.HasSuffix(p3), "expected %q to have suffix %q", p1, p3)

	p4 := dyn.NewPath(dyn.Key("foo"), dyn.Index(1))
	assert.True(t, p1.HasSuffix(p4), "expected %q to have suffix %q", p1, p4)

	p5 := dyn.NewPath(dyn.Key("bar"), dyn.Index(2), dyn.Key("baz"))
	assert.False(t, p1.HasSuffix(p5), "expected %q to not have suffix %q", p1, p5)
}

func TestPathCutPrefix(t *testing.T) {
	p1 := dyn.NewPath(dyn.Key("foo"), dyn.Index(1), dyn.Key("bar"))
	prefix := dyn.NewPath(dyn.Key("foo"), dyn.Index(1))

	// Cut a valid prefix.
	rest, ok := p1.CutPrefix(prefix)
	assert.True(t, ok, "expected %q to have prefix %q", p1, prefix)
	assert.True(t, rest.Equal(dyn.NewPath(dyn.Key("bar"))), "expected rest to be %q, got %q", dyn.NewPath(dyn.Key("bar")), rest)

	// Try to cut an invalid prefix.
	invalidPrefix := dyn.NewPath(dyn.Key("bar"))
	rest, ok = p1.CutPrefix(invalidPrefix)
	assert.False(t, ok, "expected %q to not have prefix %q", p1, invalidPrefix)
	assert.True(t, rest.Equal(p1), "expected rest to be %q, got %q", p1, rest)

	// Cut an empty prefix.
	emptyPrefix := dyn.EmptyPath
	rest, ok = p1.CutPrefix(emptyPrefix)
	assert.True(t, ok, "expected %q to have prefix %q", p1, emptyPrefix)
	assert.True(t, rest.Equal(p1), "expected rest to be %q, got %q", p1, rest)

	// Cut a prefix equal to the path.
	rest, ok = p1.CutPrefix(p1)
	assert.True(t, ok, "expected %q to have prefix %q", p1, p1)
	assert.True(t, rest.Equal(dyn.EmptyPath), "expected rest to be %q, got %q", dyn.EmptyPath, rest)
}

func TestPathCutSuffix(t *testing.T) {
	p1 := dyn.NewPath(dyn.Key("foo"), dyn.Index(1), dyn.Key("bar"))
	suffix := dyn.NewPath(dyn.Index(1), dyn.Key("bar"))

	// Cut a valid suffix.
	rest, ok := p1.CutSuffix(suffix)
	assert.True(t, ok, "expected %q to have suffix %q", p1, suffix)
	assert.True(t, rest.Equal(dyn.NewPath(dyn.Key("foo"))), "expected rest to be %q, got %q", dyn.NewPath(dyn.Key("foo")), rest)

	// Try to cut an invalid suffix.
	invalidSuffix := dyn.NewPath(dyn.Key("foo"))
	rest, ok = p1.CutSuffix(invalidSuffix)
	assert.False(t, ok, "expected %q to not have suffix %q", p1, invalidSuffix)
	assert.True(t, rest.Equal(p1), "expected rest to be %q, got %q", p1, rest)

	// Cut an empty suffix.
	emptySuffix := dyn.EmptyPath
	rest, ok = p1.CutSuffix(emptySuffix)
	assert.True(t, ok, "expected %q to have suffix %q", p1, emptySuffix)
	assert.True(t, rest.Equal(p1), "expected rest to be %q, got %q", p1, rest)

	// Cut a suffix equal to the path.
	rest, ok = p1.CutSuffix(p1)
	assert.True(t, ok, "expected %q to have suffix %q", p1, p1)
	assert.True(t, rest.Equal(dyn.EmptyPath), "expected rest to be %q, got %q", dyn.EmptyPath, rest)
}

func TestPathString(t *testing.T) {
	p1 := dyn.NewPath(dyn.Key("foo"), dyn.Index(1))
	assert.Equal(t, "foo[1]", p1.String())

	p2 := dyn.NewPath(dyn.Key("bar"), dyn.Index(2), dyn.Key("baz"))
	assert.Equal(t, "bar[2].baz", p2.String())

	p3 := dyn.NewPath(dyn.Key("foo"), dyn.Index(1), dyn.Key("bar"), dyn.Index(2), dyn.Key("baz"))
	assert.Equal(t, "foo[1].bar[2].baz", p3.String())
}

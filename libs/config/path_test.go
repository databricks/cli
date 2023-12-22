package config_test

import (
	"testing"

	"github.com/databricks/cli/libs/config"
	"github.com/stretchr/testify/assert"
)

func TestPathAppend(t *testing.T) {
	p := config.NewPath(config.Key("foo"))

	// Single arg.
	p1 := p.Append(config.Key("bar"))
	assert.True(t, p1.Equal(config.NewPath(config.Key("foo"), config.Key("bar"))))

	// Multiple args.
	p2 := p.Append(config.Key("bar"), config.Index(1))
	assert.True(t, p2.Equal(config.NewPath(config.Key("foo"), config.Key("bar"), config.Index(1))))
}

func TestPathJoin(t *testing.T) {
	p := config.NewPath(config.Key("foo"))

	// Single arg.
	p1 := p.Join(config.NewPath(config.Key("bar")))
	assert.True(t, p1.Equal(config.NewPath(config.Key("foo"), config.Key("bar"))))

	// Multiple args.
	p2 := p.Join(config.NewPath(config.Key("bar")), config.NewPath(config.Index(1)))
	assert.True(t, p2.Equal(config.NewPath(config.Key("foo"), config.Key("bar"), config.Index(1))))
}

func TestPathEqualEmpty(t *testing.T) {
	assert.True(t, config.EmptyPath.Equal(config.EmptyPath))
}

func TestPathEqual(t *testing.T) {
	p1 := config.NewPath(config.Key("foo"), config.Index(1))
	p2 := config.NewPath(config.Key("bar"), config.Index(2))
	assert.False(t, p1.Equal(p2), "expected %q to not equal %q", p1, p2)

	p3 := config.NewPath(config.Key("foo"), config.Index(1))
	assert.True(t, p1.Equal(p3), "expected %q to equal %q", p1, p3)

	p4 := config.NewPath(config.Key("foo"), config.Index(1), config.Key("bar"), config.Index(2))
	assert.False(t, p1.Equal(p4), "expected %q to not equal %q", p1, p4)
}

func TestPathHasPrefixEmpty(t *testing.T) {
	empty := config.EmptyPath
	nonEmpty := config.NewPath(config.Key("foo"))
	assert.True(t, empty.HasPrefix(empty))
	assert.True(t, nonEmpty.HasPrefix(empty))
	assert.False(t, empty.HasPrefix(nonEmpty))
}

func TestPathHasPrefix(t *testing.T) {
	p1 := config.NewPath(config.Key("foo"), config.Index(1))
	p2 := config.NewPath(config.Key("bar"), config.Index(2))
	assert.False(t, p1.HasPrefix(p2), "expected %q to not have prefix %q", p1, p2)

	p3 := config.NewPath(config.Key("foo"))
	assert.True(t, p1.HasPrefix(p3), "expected %q to have prefix %q", p1, p3)
}

func TestPathString(t *testing.T) {
	p1 := config.NewPath(config.Key("foo"), config.Index(1))
	assert.Equal(t, "foo[1]", p1.String())

	p2 := config.NewPath(config.Key("bar"), config.Index(2), config.Key("baz"))
	assert.Equal(t, "bar[2].baz", p2.String())

	p3 := config.NewPath(config.Key("foo"), config.Index(1), config.Key("bar"), config.Index(2), config.Key("baz"))
	assert.Equal(t, "foo[1].bar[2].baz", p3.String())
}

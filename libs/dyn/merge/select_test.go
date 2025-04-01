package merge

import (
	"testing"

	assert "github.com/databricks/cli/libs/dyn/dynassert"

	"github.com/databricks/cli/libs/dyn"
)

func TestSelect(t *testing.T) {
	locations := []dyn.Location{{File: "foo.yml", Line: 1, Column: 1}}
	included := []string{"foo"}
	input := dyn.NewValue(
		map[string]dyn.Value{
			"foo": dyn.V("bar"),
			"baz": dyn.V("qux"),
		},
		locations,
	)
	expected := dyn.NewValue(
		map[string]dyn.Value{
			"foo": dyn.V("bar"),
		},
		locations,
	)

	actual, err := Select(input, included)

	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestAntiSelect(t *testing.T) {
	locations := []dyn.Location{{File: "foo.yml", Line: 1, Column: 1}}
	excluded := []string{"foo"}
	input := dyn.NewValue(
		map[string]dyn.Value{
			"foo": dyn.V("bar"),
			"baz": dyn.V("qux"),
		},
		locations,
	)
	expected := dyn.NewValue(
		map[string]dyn.Value{
			"baz": dyn.V("qux"),
		},
		locations,
	)

	actual, err := AntiSelect(input, excluded)

	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}

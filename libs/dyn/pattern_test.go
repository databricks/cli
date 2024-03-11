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

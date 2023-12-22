package dyn_test

import (
	"testing"

	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
)

func TestKindZero(t *testing.T) {
	var k dyn.Kind
	assert.Equal(t, dyn.KindInvalid, k)
	assert.Equal(t, "invalid", k.String())
}

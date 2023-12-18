package config_test

import (
	"testing"

	"github.com/databricks/cli/libs/config"
	"github.com/stretchr/testify/assert"
)

func TestKindZero(t *testing.T) {
	var k config.Kind
	assert.Equal(t, config.KindInvalid, k)
	assert.Equal(t, "invalid", k.String())
}

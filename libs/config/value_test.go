package config_test

import (
	"testing"

	"github.com/databricks/cli/libs/config"
	"github.com/stretchr/testify/assert"
)

func TestValueIsAnchor(t *testing.T) {
	var zero config.Value
	assert.False(t, zero.IsAnchor())
	mark := zero.MarkAnchor()
	assert.True(t, mark.IsAnchor())
}

func TestValueAsMap(t *testing.T) {
	var zeroValue config.Value
	m, ok := zeroValue.AsMap()
	assert.False(t, ok)
	assert.Nil(t, m)

	var intValue = config.NewValue(1, config.Location{})
	m, ok = intValue.AsMap()
	assert.False(t, ok)
	assert.Nil(t, m)

	var mapValue = config.NewValue(
		map[string]config.Value{
			"key": config.NewValue("value", config.Location{File: "file", Line: 1, Column: 2}),
		},
		config.Location{File: "file", Line: 1, Column: 2},
	)
	m, ok = mapValue.AsMap()
	assert.True(t, ok)
	assert.Len(t, m, 1)
}

func TestValueIsValid(t *testing.T) {
	var zeroValue config.Value
	assert.False(t, zeroValue.IsValid())
	var intValue = config.NewValue(1, config.Location{})
	assert.True(t, intValue.IsValid())
}

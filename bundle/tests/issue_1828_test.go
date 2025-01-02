package config_tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIssue1828(t *testing.T) {
	b := load(t, "./issue_1828")

	if assert.Contains(t, b.Config.Variables, "map") {
		assert.Equal(t, map[string]any{
			"foo": "bar",
		}, b.Config.Variables["map"].Default)
	}

	if assert.Contains(t, b.Config.Variables, "sequence") {
		assert.Equal(t, []any{
			"foo",
			"bar",
		}, b.Config.Variables["sequence"].Default)
	}

	if assert.Contains(t, b.Config.Variables, "string") {
		assert.Equal(t, "foo", b.Config.Variables["string"].Default)
	}

	if assert.Contains(t, b.Config.Variables, "bool") {
		assert.Equal(t, true, b.Config.Variables["bool"].Default)
	}

	if assert.Contains(t, b.Config.Variables, "int") {
		assert.Equal(t, 42, b.Config.Variables["int"].Default)
	}

	if assert.Contains(t, b.Config.Variables, "float") {
		assert.InDelta(t, 3.14, b.Config.Variables["float"].Default, 0.0001)
	}

	if assert.Contains(t, b.Config.Variables, "time") {
		assert.Equal(t, "2021-01-01", b.Config.Variables["time"].Default)
	}

	if assert.Contains(t, b.Config.Variables, "nil") {
		assert.Nil(t, b.Config.Variables["nil"].Default)
	}
}

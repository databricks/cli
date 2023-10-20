package config_test

import (
	"testing"

	"github.com/databricks/cli/libs/config"
	"github.com/stretchr/testify/assert"
)

func TestLocation(t *testing.T) {
	loc := config.Location{File: "file", Line: 1, Column: 2}
	assert.Equal(t, "file:1:2", loc.String())
}

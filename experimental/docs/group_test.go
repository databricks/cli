package main_test

import (
	"testing"

	main "github.com/databricks/cli/experimental/docs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGroupFind(t *testing.T) {
	g := main.Find("catalogs")
	require.NotNil(t, g)
	assert.Equal(t, "catalogs", g.Name)
	assert.Equal(t, "catalogs", g.Command.Use)
	assert.Len(t, g.Subcommands, 5)
	g.Command.CalledAs()
}

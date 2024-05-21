package main_test

import (
	"context"
	"testing"

	"github.com/databricks/cli/cmd"
	main "github.com/databricks/cli/experimental/docs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCaptureHelp(t *testing.T) {
	c := cmd.New(context.Background())
	h := main.CaptureHelp(c)
	require.NotNil(t, h)
}

func TestInvocation(t *testing.T) {
	g := main.Find("catalogs")
	require.NotNil(t, g)
	assert.Equal(t, "databricks catalogs", main.Invocation(g.Command))
}

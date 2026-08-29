package project

import (
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
)

func TestInstalled(t *testing.T) {
	ctx := t.Context()
	ctx = env.WithUserHomeDir(ctx, "testdata/installed-in-home")

	all, err := Installed(ctx)
	assert.NoError(t, err)
	assert.Len(t, all, 1)
	assert.Equal(t, "blueprint", all[0].Name)
}

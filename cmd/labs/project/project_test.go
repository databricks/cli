package project

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func assertEqualPaths(t *testing.T, expected, actual string) {
	expected = strings.ReplaceAll(expected, "/", string(os.PathSeparator))
	assert.Equal(t, expected, actual)
}

func TestLoad(t *testing.T) {
	ctx := context.Background()
	prj, err := Load(ctx, "testdata/installed-in-home/.databricks/labs/blueprint/lib/labs.yml")
	assert.NoError(t, err)
	assertEqualPaths(t, "testdata/installed-in-home/.databricks/labs/blueprint/lib", prj.folder)
}

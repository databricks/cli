package internal

import (
	"fmt"
	"testing"

	"github.com/databricks/cli/internal/build"
	"github.com/stretchr/testify/assert"
)

var expectedVersion = fmt.Sprintf("Databricks CLI v%s\n", build.GetInfo().Version)

func TestVersionFlagShort(t *testing.T) {
	stdout, stderr := RequireSuccessfulRun(t, "-v")
	assert.Equal(t, expectedVersion, stdout.String())
	assert.Equal(t, "", stderr.String())
}

func TestVersionFlagLong(t *testing.T) {
	stdout, stderr := RequireSuccessfulRun(t, "--version")
	assert.Equal(t, expectedVersion, stdout.String())
	assert.Equal(t, "", stderr.String())
}

func TestVersionCommand(t *testing.T) {
	stdout, stderr := RequireSuccessfulRun(t, "version")
	assert.Equal(t, expectedVersion, stdout.String())
	assert.Equal(t, "", stderr.String())
}

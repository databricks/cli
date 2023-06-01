package internal

import (
	"encoding/json"
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

func TestVersionCommandWithJSONOutput(t *testing.T) {
	stdout, stderr := RequireSuccessfulRun(t, "version", "--output", "json")
	assert.NotEmpty(t, stdout.String())
	assert.Equal(t, "", stderr.String())

	// Deserialize stdout and confirm we see the right fields.
	var output map[string]any
	err := json.Unmarshal(stdout.Bytes(), &output)
	assert.NoError(t, err)
	assert.Equal(t, build.GetInfo().Version, output["Version"])
}

package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAccWorkspaceList(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	stdout, stderr := RequireSuccessfulRun(t, "workspace", "list", "/")
	outStr := stdout.String()
	assert.Contains(t, outStr, "ID")
	assert.Contains(t, outStr, "Type")
	assert.Contains(t, outStr, "Language")
	assert.Contains(t, outStr, "Path")
	assert.Equal(t, "", stderr.String())
}

func TestWorkpaceListErrorWhenNoArguments(t *testing.T) {
	_, _, err := RequireErrorRun(t, "workspace", "list")
	assert.Equal(t, "accepts 1 arg(s), received 0", err.Error())
}

func TestWorkpaceGetStatusErrorWhenNoArguments(t *testing.T) {
	_, _, err := RequireErrorRun(t, "workspace", "get-status")
	assert.Equal(t, "accepts 1 arg(s), received 0", err.Error())
}

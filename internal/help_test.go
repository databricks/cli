package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAccVersionHelp(t *testing.T) {
	stdout, stderr := RequireSuccessfulRun(t, "version", "--help")
	assert.Empty(t, stderr.String())
	// We rely on the help message containing the string below in our driver local
	// tests that assert the CLI is installed in DBR versions newer than 15.0.
	// Please don't change this string. If you need to change the help message,
	// please update the driver local tests as well.
	assert.Contains(t, stdout.String(), "Retrieve information about the current version of this CLI")
}

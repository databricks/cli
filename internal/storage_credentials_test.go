package internal

import (
	"testing"

	"github.com/databricks/cli/internal/acc"
	"github.com/stretchr/testify/assert"
)

func TestStorageCredentialsListRendersResponse(t *testing.T) {
	_, _ = acc.WorkspaceTest(t)
	stdout, stderr := RequireSuccessfulRun(t, "storage-credentials", "list")
	assert.NotEmpty(t, stdout)
	assert.Empty(t, stderr)
}

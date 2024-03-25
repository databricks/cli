package internal

import (
	"testing"

	"github.com/databricks/cli/internal/acc"
	"github.com/stretchr/testify/assert"
)

func TestAccStorageCredentialsListRendersResponse(t *testing.T) {
	_, _ = acc.WorkspaceTest(t)

	// Check if metastore is assigned for the workspace, otherwise test will fail
	t.Log(GetEnvOrSkipTest(t, "TEST_METASTORE_ID"))

	stdout, stderr := RequireSuccessfulRun(t, "storage-credentials", "list")
	assert.NotEmpty(t, stdout)
	assert.Empty(t, stderr)
}

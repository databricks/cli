package auth

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckAuthentication_SkipWhenEnvVarSet(t *testing.T) {
	// Set the skip auth check environment variable
	os.Setenv("DATABRICKS_AITOOLS_SKIP_AUTH_CHECK", "1")
	defer os.Unsetenv("DATABRICKS_AITOOLS_SKIP_AUTH_CHECK")

	// Should not return an error when env var is set
	err := CheckAuthentication(context.Background())
	assert.NoError(t, err)
}

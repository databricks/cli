package labs_test

import (
	"context"
	"testing"

	"github.com/databricks/cli/cmd/internal"
	"github.com/stretchr/testify/assert"
)

func TestInstallDbx(t *testing.T) {
	ctx := context.Background()
	_, err := internal.RunGetOutput(ctx, "labs", "install", "dbx@metadata", "--profile", "bogdan")
	assert.NoError(t, err)
}

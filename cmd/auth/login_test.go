package auth

import (
	"context"
	"testing"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
)

func TestSetHostDoesNotFailWithNoDatabrickscfg(t *testing.T) {
	ctx := context.Background()
	ctx = env.Set(ctx, "DATABRICKS_CONFIG_FILE", "./imaginary-file/databrickscfg")
	err := setHost(ctx, "foo", &auth.PersistentAuth{Host: "test"}, []string{})
	assert.NoError(t, err)
}

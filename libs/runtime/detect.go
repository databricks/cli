package runtime

import (
	"context"

	"github.com/databricks/cli/libs/env"
)

const envDatabricksRuntimeVersion = "DATABRICKS_RUNTIME_VERSION"

func RunsOnDatabricks(ctx context.Context) bool {
	_, ok := env.Lookup(ctx, envDatabricksRuntimeVersion)
	return ok
}

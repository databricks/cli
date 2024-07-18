package env

import "context"

const HostVariable = "DATABRICKS_HOST"

func Host(ctx context.Context) (string, bool) {
	return get(ctx, []string{
		HostVariable,
	})
}

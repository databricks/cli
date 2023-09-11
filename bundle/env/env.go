package env

import (
	"context"

	envlib "github.com/databricks/cli/libs/env"
)

// Return the value of the first environment variable that is set.
func get(ctx context.Context, variables []string) (string, bool) {
	for _, v := range variables {
		value, ok := envlib.Lookup(ctx, v)
		if ok {
			return value, true
		}
	}
	return "", false
}

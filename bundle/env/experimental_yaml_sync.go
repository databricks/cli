package env

import "context"

// ExperimentalYamlSyncVariable names the environment variable that holds the flag whether
// experimental YAML sync is enabled.
const experimentalYamlSyncVariable = "DATABRICKS_BUNDLE_ENABLE_EXPERIMENTAL_YAML_SYNC"

// ExperimentalYamlSync returns the environment variable that holds the flag whether
// experimental YAML sync is enabled.
func ExperimentalYamlSync(ctx context.Context) (string, bool) {
	return get(ctx, []string{
		experimentalYamlSyncVariable,
	})
}

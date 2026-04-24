package terraform

import (
	"context"
	"errors"
	"io/fs"
	"os"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
)

// getEnvVarWithMatchingVersion returns envVarName's value only when the
// path it points to exists and, if versionVarName is set, its value
// matches currentVersion. Mirrors bundle/deploy/terraform/init.go. The
// VSCode extension sets DATABRICKS_TF_CLI_CONFIG_FILE + the corresponding
// DATABRICKS_TF_PROVIDER_VERSION so that a cached provider mirror is only
// honoured when it was built against the provider version we actually
// use; using a mismatched mirror would make terraform init fail.
func getEnvVarWithMatchingVersion(ctx context.Context, envVarName, versionVarName, currentVersion string) (string, error) {
	envValue := env.Get(ctx, envVarName)
	versionValue := env.Get(ctx, versionVarName)

	if envValue == "" {
		log.Debugf(ctx, "%s is not defined", envVarName)
		return "", nil
	}

	if _, err := os.Stat(envValue); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			log.Debugf(ctx, "%s at %s does not exist", envVarName, envValue)
			return "", nil
		}
		return "", err
	}

	if versionValue == "" {
		return envValue, nil
	}

	if versionValue != currentVersion {
		log.Debugf(ctx, "%s as %s does not match the current version %s, ignoring %s", versionVarName, versionValue, currentVersion, envVarName)
		return "", nil
	}
	return envValue, nil
}

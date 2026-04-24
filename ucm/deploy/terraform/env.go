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

// envCopy enumerates environment variables that are passed through to the
// terraform subprocess verbatim. Mirrors bundle/deploy/terraform/init.go.
var envCopy = []string{
	// $HOME — terraform and the databricks provider read ~/.databrickscfg,
	// ~/.databricks/token-cache, etc. from HOME.
	"HOME",

	// $USERPROFILE — Windows equivalent of HOME; used by Azure CLI to
	// locate stored credentials and metadata.
	"USERPROFILE",

	// $PATH — so the databricks provider can invoke auxiliary tools
	// (`az`, `gcloud`) that live on PATH.
	"PATH",

	// $AZURE_CONFIG_DIR — set by Azure DevOps' AzureCLI@2 task so
	// downstream az invocations share the same config dir.
	"AZURE_CONFIG_DIR",

	// $TF_CLI_CONFIG_FILE — override terraform provider source in
	// development. See
	// https://developer.hashicorp.com/terraform/cli/config/config-file
	"TF_CLI_CONFIG_FILE",

	// $USE_SDK_V2_RESOURCES / $USE_SDK_V2_DATA_SOURCES — escape hatch for
	// the databricks provider's plugin-framework ↔ SDKv2 migration.
	// See https://registry.terraform.io/providers/databricks/databricks/latest/docs/guides/troubleshooting#plugin-framework-migration-problems
	"USE_SDK_V2_RESOURCES",
	"USE_SDK_V2_DATA_SOURCES",
}

// azureDevOpsSystemVars enumerates Azure DevOps SYSTEM_* variables the
// databricks SDK reads during OIDC authentication on Azure DevOps
// pipelines. Passed through so terraform-spawned SDK calls can use the
// same OIDC token exchange the parent CLI would.
var azureDevOpsSystemVars = []string{
	"SYSTEM_ACCESSTOKEN",
	"SYSTEM_COLLECTIONID",
	"SYSTEM_COLLECTIONURI",
	"SYSTEM_DEFINITIONID",
	"SYSTEM_HOSTTYPE",
	"SYSTEM_JOBID",
	"SYSTEM_OIDCREQUESTURI",
	"SYSTEM_PLANID",
	"SYSTEM_TEAMFOUNDATIONCOLLECTIONURI",
	"SYSTEM_TEAMPROJECT",
	"SYSTEM_TEAMPROJECTID",
}

// inheritEnvVars populates environ with env vars that should cross into
// the terraform subprocess: the envCopy allow-list, OIDC token (direct or
// indirect via DATABRICKS_OIDC_TOKEN_ENV), Azure DevOps SYSTEM_* vars,
// and a version-gated DATABRICKS_TF_CLI_CONFIG_FILE → TF_CLI_CONFIG_FILE
// mapping. Mirrors bundle/deploy/terraform/init.go's inheritEnvVars.
func inheritEnvVars(ctx context.Context, environ map[string]string) error {
	for _, key := range envCopy {
		if v, ok := env.Lookup(ctx, key); ok {
			environ[key] = v
		}
	}

	// DATABRICKS_OIDC_TOKEN_ENV points at another env var that holds the
	// actual token. When unset we fall back to DATABRICKS_OIDC_TOKEN.
	oidcTokenEnv, ok := env.Lookup(ctx, "DATABRICKS_OIDC_TOKEN_ENV")
	if ok {
		environ["DATABRICKS_OIDC_TOKEN_ENV"] = oidcTokenEnv
	} else {
		oidcTokenEnv = "DATABRICKS_OIDC_TOKEN"
	}
	if token, ok := env.Lookup(ctx, oidcTokenEnv); ok {
		environ[oidcTokenEnv] = token
	}

	for _, k := range azureDevOpsSystemVars {
		if v, ok := env.Lookup(ctx, k); ok {
			environ[k] = v
		}
	}

	// Map DATABRICKS_TF_CLI_CONFIG_FILE → TF_CLI_CONFIG_FILE only when the
	// mirror matches the provider version we actually use; otherwise
	// terraform init would fail to download the right version.
	configFile, err := getEnvVarWithMatchingVersion(ctx, CliConfigPathEnv, ProviderVersionEnv, ProviderVersion)
	if err != nil {
		return err
	}
	if configFile != "" {
		log.Debugf(ctx, "Using Terraform CLI config from %s at %s", CliConfigPathEnv, configFile)
		environ["TF_CLI_CONFIG_FILE"] = configFile
	}

	return nil
}

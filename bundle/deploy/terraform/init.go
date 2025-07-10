package terraform

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
	"github.com/hashicorp/hc-install/product"
	"github.com/hashicorp/terraform-exec/tfexec"
	"golang.org/x/exp/maps"
)

type initialize struct{}

func (m *initialize) Name() string {
	return "terraform.Initialize"
}

func (m *initialize) findExecPath(ctx context.Context, b *bundle.Bundle, tf *config.Terraform, installer Installer) (string, error) {
	// If set, pass it through [exec.LookPath] to resolve its absolute path.
	if tf.ExecPath != "" {
		execPath, err := exec.LookPath(tf.ExecPath)
		if err != nil {
			return "", err
		}
		tf.ExecPath = execPath
		log.Debugf(ctx, "Using Terraform at %s", tf.ExecPath)
		return tf.ExecPath, nil
	}

	// Resolve the version of the Terraform CLI to use.
	tv, isDefault, err := GetTerraformVersion(ctx)
	if err != nil {
		return "", err
	}

	// Allow the user to specify the path to the Terraform CLI.
	// If this is set, verify that the version matches the version we expect.
	if execPathValue, ok := env.Lookup(ctx, TerraformExecPathEnv); ok {
		tfe, err := getTerraformExec(ctx, b, execPathValue)
		if err != nil {
			return "", err
		}

		expectedVersion := tv.Version
		actualVersion, _, err := tfe.Version(ctx, false)
		if err != nil {
			return "", fmt.Errorf("unable to execute %s: %w", TerraformExecPathEnv, err)
		}

		if !actualVersion.Equal(expectedVersion) {
			if isDefault {
				return "", fmt.Errorf(
					"Terraform binary at %s (from $%s) is %s but expected version is %s. Set %s to %s to continue.",
					execPathValue,
					TerraformExecPathEnv,
					actualVersion.String(),
					expectedVersion.String(),
					TerraformVersionEnv,
					actualVersion.String(),
				)
			} else {
				return "", fmt.Errorf(
					"Terraform binary at %s (from $%s) is %s but expected version is %s (from $%s). Update $%s and $%s so that versions match.",
					execPathValue,
					TerraformExecPathEnv,
					actualVersion.String(),
					expectedVersion.String(),
					TerraformVersionEnv,
					TerraformExecPathEnv,
					TerraformVersionEnv,
				)
			}
		}

		tf.ExecPath = execPathValue
		log.Debugf(ctx, "Using Terraform from %s at %s", TerraformExecPathEnv, tf.ExecPath)
		return tf.ExecPath, nil
	}

	binDir, err := b.LocalStateDir(ctx, "bin")
	if err != nil {
		return "", err
	}

	// If the execPath already exists, return it.
	execPath := filepath.Join(binDir, product.Terraform.BinaryName())
	_, err = os.Stat(execPath)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return "", err
	}
	if err == nil {
		tf.ExecPath = execPath
		log.Debugf(ctx, "Using Terraform at %s", tf.ExecPath)
		return tf.ExecPath, nil
	}

	// Download Terraform to private bin directory.
	execPath, err = installer.Install(ctx, binDir, tv.Version)
	if err != nil {
		return "", fmt.Errorf("error downloading Terraform: %w", err)
	}

	tf.ExecPath = execPath
	log.Debugf(ctx, "Using Terraform at %s", tf.ExecPath)
	return tf.ExecPath, nil
}

var envCopy = []string{
	// Include $HOME in set of environment variables to pass along.
	"HOME",

	// Include $USERPROFILE in set of environment variables to pass along.
	// This variable is used by Azure CLI on Windows to find stored credentials and metadata
	"USERPROFILE",

	// Include $PATH in set of environment variables to pass along.
	// This is necessary to ensure that our Terraform provider can use the
	// same auxiliary programs (e.g. `az`, or `gcloud`) as the CLI.
	"PATH",

	// Include $AZURE_CONFIG_DIR in set of environment variables to pass along.
	// This is set in Azure DevOps by the AzureCLI@2 task.
	"AZURE_CONFIG_DIR",

	// Include $TF_CLI_CONFIG_FILE to override terraform provider in development.
	// See: https://developer.hashicorp.com/terraform/cli/config/config-file#explicit-installation-method-configuration
	"TF_CLI_CONFIG_FILE",

	// Include $USE_SDK_V2_RESOURCES and $USE_SDK_V2_DATA_SOURCES, these are used to switch back from plugin framework to SDKv2.
	// This is used for mitigation issues with resource migrated to plugin framework, as recommended here:
	// https://registry.terraform.io/providers/databricks/databricks/latest/docs/guides/troubleshooting#plugin-framework-migration-problems
	// It is currently a workaround for deploying quality_monitors
	// https://github.com/databricks/terraform-provider-databricks/issues/4229#issuecomment-2520344690
	"USE_SDK_V2_RESOURCES",
	"USE_SDK_V2_DATA_SOURCES",
}

// This function inherits some environment variables for Terraform CLI.
func inheritEnvVars(ctx context.Context, environ map[string]string) error {
	for _, key := range envCopy {
		value, ok := env.Lookup(ctx, key)
		if ok {
			environ[key] = value
		}
	}

	// If there's a DATABRICKS_OIDC_TOKEN_ENV set, we need to pass the value of the environment variable defined in DATABRICKS_OIDC_TOKEN_ENV to Terraform.
	// This is necessary to ensure that Terraform can use the same OIDC token as the CLI.
	oidcTokenEnv, ok := env.Lookup(ctx, "DATABRICKS_OIDC_TOKEN_ENV")
	if ok {
		environ["DATABRICKS_OIDC_TOKEN_ENV"] = oidcTokenEnv
	} else {
		oidcTokenEnv = "DATABRICKS_OIDC_TOKEN"
	}

	oidcToken, ok := env.Lookup(ctx, oidcTokenEnv)
	if ok {
		environ[oidcTokenEnv] = oidcToken
	}

	// Map $DATABRICKS_TF_CLI_CONFIG_FILE to $TF_CLI_CONFIG_FILE
	// VSCode extension provides a file with the "provider_installation.filesystem_mirror" configuration.
	// We only use it if the provider version matches the currently used version,
	// otherwise terraform will fail to download the right version (even with unrestricted internet access).
	configFile, err := getEnvVarWithMatchingVersion(ctx, TerraformCliConfigPathEnv, TerraformProviderVersionEnv, schema.ProviderVersion)
	if err != nil {
		return err
	}
	if configFile != "" {
		log.Debugf(ctx, "Using Terraform CLI config from %s at %s", TerraformCliConfigPathEnv, configFile)
		environ["TF_CLI_CONFIG_FILE"] = configFile
	}

	return nil
}

// Example: this function will return a value of TF_EXEC_PATH only if the path exists and if TF_VERSION matches the TerraformVersion.
// This function is used for env vars set by the Databricks VSCode extension. The variables are intended to be used by the CLI
// bundled with the Databricks VSCode extension, but users can use different CLI versions in the VSCode terminals, in which case we want to ignore
// the variables if that CLI uses different versions of the dependencies.
func getEnvVarWithMatchingVersion(ctx context.Context, envVarName, versionVarName, currentVersion string) (string, error) {
	envValue := env.Get(ctx, envVarName)
	versionValue := env.Get(ctx, versionVarName)

	// return early if the environment variable is not set
	if envValue == "" {
		log.Debugf(ctx, "%s is not defined", envVarName)
		return "", nil
	}

	// If the path does not exist, we return early.
	_, err := os.Stat(envValue)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			log.Debugf(ctx, "%s at %s does not exist", envVarName, envValue)
			return "", nil
		} else {
			return "", err
		}
	}

	// If the version environment variable is not set, we directly return the value of the environment variable.
	if versionValue == "" {
		return envValue, nil
	}

	// When the version environment variable is set, we check if it matches the current version.
	// If it does not match, we return an empty string.
	if versionValue != currentVersion {
		log.Debugf(ctx, "%s as %s does not match the current version %s, ignoring %s", versionVarName, versionValue, currentVersion, envVarName)
		return "", nil
	}
	return envValue, nil
}

// This function sets temp dir location for terraform to use. If user does not
// specify anything here, we fall back to a `tmp` directory in the bundle's cache
// directory
//
// This is necessary to avoid trying to create temporary files in directories
// the CLI and its dependencies do not have access to.
//
// see: os.TempDir for more context
func setTempDirEnvVars(ctx context.Context, environ map[string]string, b *bundle.Bundle) error {
	switch runtime.GOOS {
	case "windows":
		if v, ok := env.Lookup(ctx, "TMP"); ok {
			environ["TMP"] = v
		} else if v, ok := env.Lookup(ctx, "TEMP"); ok {
			environ["TEMP"] = v
		} else {
			tmpDir, err := b.LocalStateDir(ctx, "tmp")
			if err != nil {
				return err
			}
			environ["TMP"] = tmpDir
		}
	default:
		// If TMPDIR is not set, we let the process fall back to its default value.
		if v, ok := env.Lookup(ctx, "TMPDIR"); ok {
			environ["TMPDIR"] = v
		}
	}
	return nil
}

// This function passes through all proxy related environment variables.
func setProxyEnvVars(ctx context.Context, environ map[string]string, b *bundle.Bundle) error {
	for _, v := range []string{"http_proxy", "https_proxy", "no_proxy"} {
		// The case (upper or lower) is notoriously inconsistent for tools on Unix systems.
		// We therefore try to read both the upper and lower case versions of the variable.
		for _, v := range []string{strings.ToUpper(v), strings.ToLower(v)} {
			if val, ok := env.Lookup(ctx, v); ok {
				// Only set uppercase version of the variable.
				environ[strings.ToUpper(v)] = val
			}
		}
	}
	return nil
}

func setUserAgentExtraEnvVar(environ map[string]string, b *bundle.Bundle) error {
	// Add "cli" to the user agent in set by the Databricks Terraform provider.
	// This will allow us to attribute downstream requests made by the Databricks
	// Terraform provider to the CLI.
	products := []string{"cli/" + build.GetInfo().Version}
	if experimental := b.Config.Experimental; experimental != nil {
		hasPython := experimental.Python.Resources != nil || experimental.Python.Mutators != nil

		if hasPython {
			products = append(products, "databricks-pydabs/0.7.0")
		}
	}

	userAgentExtra := strings.Join(products, " ")
	if userAgentExtra != "" {
		environ["DATABRICKS_USER_AGENT_EXTRA"] = userAgentExtra
	}

	return nil
}

func getTerraformExec(ctx context.Context, b *bundle.Bundle, execPath string) (*tfexec.Terraform, error) {
	workingDir, err := Dir(ctx, b)
	if err != nil {
		return nil, err
	}

	return tfexec.NewTerraform(workingDir, execPath)
}

func (m *initialize) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	tfConfig := b.Config.Bundle.Terraform
	if tfConfig == nil {
		tfConfig = &config.Terraform{}
		b.Config.Bundle.Terraform = tfConfig
	}

	execPath, err := m.findExecPath(ctx, b, tfConfig, tfInstaller{})
	if err != nil {
		return diag.FromErr(err)
	}

	tfe, err := getTerraformExec(ctx, b, execPath)
	if err != nil {
		return diag.FromErr(err)
	}

	environ, err := b.AuthEnv()
	if err != nil {
		return diag.FromErr(err)
	}

	err = inheritEnvVars(ctx, environ)
	if err != nil {
		return diag.FromErr(err)
	}

	// Set the temporary directory environment variables
	err = setTempDirEnvVars(ctx, environ, b)
	if err != nil {
		return diag.FromErr(err)
	}

	// Set the proxy related environment variables
	err = setProxyEnvVars(ctx, environ, b)
	if err != nil {
		return diag.FromErr(err)
	}

	err = setUserAgentExtraEnvVar(environ, b)
	if err != nil {
		return diag.FromErr(err)
	}

	// Configure environment variables for auth for Terraform to use.
	log.Debugf(ctx, "Environment variables for Terraform: %s", strings.Join(maps.Keys(environ), ", "))
	err = tfe.SetEnv(environ)
	if err != nil {
		return diag.FromErr(err)
	}

	b.Terraform = tfe
	return nil
}

func Initialize() bundle.Mutator {
	return &initialize{}
}

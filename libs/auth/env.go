package auth

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/databricks/databricks-sdk-go/config"
)

// Env generates the authentication environment variables we need to set for
// downstream applications from the CLI to work correctly.
func Env(cfg *config.Config) map[string]string {
	out := make(map[string]string)
	for _, attr := range config.ConfigAttributes {
		// Ignore profile so that downstream tools don't try and reload
		// the profile. We know the current configuration is already valid since
		// otherwise the CLI would have thrown an error when loading it.
		if attr.Name == "profile" {
			continue
		}
		if len(attr.EnvVars) == 0 {
			continue
		}
		if attr.IsZero(cfg) {
			continue
		}
		out[attr.EnvVars[0]] = attr.GetString(cfg)
	}

	return out
}

func GetEnvFor(name string) (string, bool) {
	for _, attr := range config.ConfigAttributes {
		if attr.Name != name {
			continue
		}
		if len(attr.EnvVars) == 0 {
			return "", false
		}
		return attr.EnvVars[0], true
	}

	return "", false
}

// EnvVars returns the list of environment variables that the SDK reads to configure
// authentication.
// This is useful for spawning subprocesses since you can unset all auth environment
// variables to clean up the environment before configuring authentication for the
// child process.
func envVars() []string {
	var out []string

	for _, attr := range config.ConfigAttributes {
		if len(attr.EnvVars) == 0 {
			continue
		}

		out = append(out, attr.EnvVars...)
	}

	return out
}

// ProcessEnv generates the environment variables that should be set to authenticate
// downstream processes to use the same auth credentials as in cfg.
func ProcessEnv(cfg *config.Config) []string {
	// We want child processes to inherit environment variables like $HOME or $HTTPS_PROXY
	// because they influence auth resolution.
	base := os.Environ()

	var out []string
	authEnvVars := envVars()

	// Remove any existing auth environment variables. This is done because
	// the CLI offers multiple modalities of configuring authentication like
	// `--profile` or `DATABRICKS_CONFIG_PROFILE` or `profile: <profile>` in the
	// bundle config file.
	//
	// Each of these modalities have different priorities and thus we don't want
	// any auth configuration to piggyback into the child process environment.
	//
	// This is a precaution to avoid conflicting auth configurations being passed
	// to the child telemetry process.
	//
	// Normally this should be unnecessary because the SDK should error if multiple
	// authentication methods have been configured. But there is no harm in doing this
	// as a precaution.
	for _, v := range base {
		k, _, found := strings.Cut(v, "=")
		if !found {
			continue
		}
		if slices.Contains(authEnvVars, k) {
			continue
		}
		out = append(out, v)
	}

	// Now add the necessary authentication environment variables.
	newEnv := Env(cfg)
	for k, v := range newEnv {
		out = append(out, fmt.Sprintf("%s=%s", k, v))
	}

	// Sort the environment variables so that the output is deterministic.
	// Keeping the output deterministic helps with reproducibility and keeping the
	// behavior consistent incase there are any issues.
	slices.Sort(out)

	return out
}

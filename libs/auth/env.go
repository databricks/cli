package auth

import "github.com/databricks/databricks-sdk-go/config"

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
func EnvVars() []string {
	out := []string{}

	for _, attr := range config.ConfigAttributes {
		if len(attr.EnvVars) == 0 {
			continue
		}

		out = append(out, attr.EnvVars...)
	}

	return out
}

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

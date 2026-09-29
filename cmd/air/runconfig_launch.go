package aircmd

// This file flattens the validated runConfig schema into the derived values the
// launch path consumes, replacing the Python CLI's _convert_to_run_config step.
// There is no separate internal config type: handle_run reads runConfig directly,
// using these accessors for the values that need computing rather than a plain
// field read.

const defaultMaxRetries = 3

// timeoutSeconds converts timeout_minutes to seconds. Zero means the user set no
// timeout and the backend default applies.
func (c *runConfig) timeoutSeconds() int {
	if c.TimeoutMinutes == nil {
		return 0
	}
	return *c.TimeoutMinutes * 60
}

// maxRetries returns the retry count, applying the schema default when unset.
func (c *runConfig) maxRetries() int {
	if c.MaxRetries == nil {
		return defaultMaxRetries
	}
	return *c.MaxRetries
}

// unityCatalogImagePath returns the configured Unity Catalog image reference,
// or "" when none is set.
func (c *runConfig) unityCatalogImagePath() string {
	if c.Environment == nil {
		return ""
	}
	return c.Environment.UnityCatalogImage
}

// inlineDependencies returns the inline package list from
// environment.dependencies, and whether it was set.
func (c *runConfig) inlineDependencies() ([]string, bool) {
	if c.Environment == nil || !c.Environment.Dependencies.set {
		return nil, false
	}
	return c.Environment.Dependencies.list, true
}

// runtimeVersion returns the client image version from environment.version when
// set.
func (c *runConfig) runtimeVersion() (string, bool) {
	if c.Environment == nil {
		return "", false
	}
	if !c.Environment.Version.set {
		return "", false
	}
	return c.Environment.Version.raw, true
}

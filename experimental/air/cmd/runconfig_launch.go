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

// dockerImageURL returns the custom docker image URL, or "" when none is set.
//
// TODO: not wired into submission yet — the native ai_runtime_task carries no
// docker field, and full support needs image registration (pending the DCS work).
func (c *runConfig) dockerImageURL() string {
	if c.Environment != nil && c.Environment.DockerImage != nil {
		return c.Environment.DockerImage.URL
	}
	return ""
}

// requirementsFile returns the path to a requirements file when
// environment.dependencies is a string, and whether it was set.
func (c *runConfig) requirementsFile() (string, bool) {
	if c.Environment == nil || !c.Environment.Dependencies.set || c.Environment.Dependencies.isList {
		return "", false
	}
	return c.Environment.Dependencies.path, true
}

// inlineDependencies returns the inline package list when
// environment.dependencies is a list, and whether it was set.
func (c *runConfig) inlineDependencies() ([]string, bool) {
	if c.Environment == nil || !c.Environment.Dependencies.set || !c.Environment.Dependencies.isList {
		return nil, false
	}
	return c.Environment.Dependencies.list, true
}

// runtimeVersion returns the client image version from environment.version when
// set. For a requirements-file dependency set, the version lives in that file and
// is resolved at launch, not here.
func (c *runConfig) runtimeVersion() (string, bool) {
	if c.Environment == nil || !c.Environment.Version.set {
		return "", false
	}
	return c.Environment.Version.raw, true
}

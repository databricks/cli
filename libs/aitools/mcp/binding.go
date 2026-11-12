package mcp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"

	"github.com/databricks/cli/libs/aitools/mcpconfig"
	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/shellquote"
	"github.com/databricks/databricks-sdk-go/config"
)

// bindExitCode is the exit status for binding failures, so callers can tell them apart
// from other errors.
const bindExitCode = 2

// Commands that binding messages tell the user to run.
const (
	authLoginCommand = "databricks auth login"
	mcpRemoveCommand = "databricks aitools mcp remove"
)

// BindCategory names the kind of a binding failure. The values are stable identifiers
// for the failure, not telemetry values.
type BindCategory string

const (
	BindProfileNotFound    BindCategory = "PROFILE_NOT_FOUND"
	BindProfileHostChanged BindCategory = "PROFILE_HOST_CHANGED"
	BindAuthFailed         BindCategory = "AUTH_FAILED"
	BindUncategorized      BindCategory = "UNCATEGORIZED"
)

// BindOptions selects what a proxy binds to. Service is the full name, for example system.ai.github.
type BindOptions struct {
	Profile string
	Service string
	State   *State
	// Stderr receives the notice printed when no install record exists. It must not be nil.
	Stderr io.Writer
}

// BindError is a failed binding check. Its message is the text printed to stderr.
type BindError struct {
	Category BindCategory
	message  string
	cause    error
}

func (e *BindError) Error() string {
	return e.message
}

func (e *BindError) Unwrap() error {
	return e.cause
}

// ExitCode returns the process exit status for this failure.
func (e *BindError) ExitCode() int {
	return bindExitCode
}

// Bind checks that the profile still names the host and workspace recorded for the service,
// then authenticates with the profile's own credentials. It never reads stdin, prompts, or
// loads a bundle. It returns the resolved config and the matching state entry, which is nil
// when none exists.
//
// The config file is the one that DATABRICKS_CONFIG_FILE names in the context, resolved once:
// the profile lookup, the entry match and cfg.ConfigFile all use that path. The profile is
// checked before the SDK resolves the config, so a re-pointed host is never contacted. The CLI's
// token store reads DATABRICKS_CONFIG_FILE from the process environment instead, so a caller
// that sets the variable in the context must also set it in the process environment.
func Bind(ctx context.Context, opts BindOptions) (*config.Config, *Entry, error) {
	configFile, err := ResolveConfigFile(ctx, env.Get(ctx, mcpconfig.ConfigFileEnv))
	if err != nil {
		return nil, nil, uncategorizedError(err)
	}
	ctx = env.Set(ctx, mcpconfig.ConfigFileEnv, configFile)
	entry, err := findEntry(ctx, opts.State, opts.Service, opts.Profile, configFile)
	if err != nil {
		return nil, nil, err
	}
	name := opts.Service
	if entry != nil {
		name = entry.Name
	}

	p, err := profileFor(name, opts.Profile, configFile, entry)
	if err != nil {
		return nil, nil, err
	}
	if entry != nil {
		if drift := driftOf(*entry, p); drift != nil {
			return nil, nil, driftError(name, opts.Profile, *entry, *drift)
		}
	}

	// The loaders take the profile's own settings and non-auth environment only. The
	// profile loader chain in databrickscfg also fills missing credentials from the
	// environment, but the proxy must not send a token meant for another workspace.
	cfg := &config.Config{
		Profile:    opts.Profile,
		ConfigFile: configFile,
		Loaders:    []config.Loader{databrickscfg.ResolveNonAuthFromEnv, config.ConfigFile},
	}
	if err := cfg.EnsureResolved(); err != nil {
		// Configuration that the SDK rejects is not a credential failure, so a login would not fix it.
		return nil, nil, uncategorizedError(err)
	}

	if entry == nil {
		fmt.Fprintf(opts.Stderr, "No install record for %s on profile \"%s\"; using its current host %s\n", opts.Service, opts.Profile, cfg.Host)
	}

	// Authenticate only writes headers onto req, which is never sent. The request takes
	// the command's context, as in cmd/root/auth.go.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "", nil)
	if err != nil {
		return nil, nil, uncategorizedError(err)
	}
	if err := cfg.Authenticate(req); err != nil {
		return nil, nil, AuthError(opts.Profile, err)
	}
	return cfg, entry, nil
}

// CheckBinding returns the error that Bind returns for the state entry of service and profileName
// when the profile no longer matches it: the entry is pinned to another config file, the profile is
// gone or has no host, or its host or workspace changed. It returns nil when there is no entry or the
// entry still matches. It reads the config files only and makes no request, so a caller can refuse
// before it sends a token to the profile's host.
func CheckBinding(ctx context.Context, state *State, service, profileName string) error {
	configFile, err := ResolveConfigFile(ctx, env.Get(ctx, mcpconfig.ConfigFileEnv))
	if err != nil {
		return uncategorizedError(err)
	}
	entry, err := findEntry(ctx, state, service, profileName, configFile)
	if err != nil || entry == nil {
		return err
	}
	p, err := profileFor(entry.Name, profileName, configFile, entry)
	if err != nil {
		return err
	}
	if drift := driftOf(*entry, p); drift != nil {
		return driftError(entry.Name, profileName, *entry, *drift)
	}
	return nil
}

// profileFor returns the profile profileName in configFile, or the PROFILE_NOT_FOUND error that says it
// is gone. name and entry describe the state entry the profile serves; entry is nil without one.
func profileFor(name, profileName, configFile string, entry *Entry) (profile.Profile, error) {
	p, err := configFiles{}.profile(configFile, profileName)
	switch {
	case errors.Is(err, errProfileMissing):
		return profile.Profile{}, profileGoneError(name, profileName, filepath.ToSlash(configFile), entry, "no longer exists")
	case errors.Is(err, errProfileNoHost):
		return profile.Profile{}, profileGoneError(name, profileName, filepath.ToSlash(configFile), entry, "has no host")
	case err != nil:
		return profile.Profile{}, uncategorizedError(err)
	}
	return p, nil
}

// uncategorizedError wraps a failure that has no category of its own. Its message is the cause's.
func uncategorizedError(err error) error {
	return &BindError{Category: BindUncategorized, message: err.Error(), cause: err}
}

// removeCommand returns the command that removes the server bound to profileName.
func removeCommand(profileName string) string {
	return mcpRemoveCommand + " --profile " + shellquote.BashArg(profileName)
}

// profileGoneError builds the PROFILE_NOT_FOUND error for a profile that the config file does not
// name with a host, either because it is absent or because its section has no host. problem completes
// the sentence "which ... in <path>". Without a state entry the recorded host is unknown, so the
// recreate hint omits --host. An entry pinned to a config file puts that file in the recreate hint.
func profileGoneError(name, profileName, path string, entry *Entry, problem string) error {
	recreate := authLoginCommand + " --profile " + shellquote.BashArg(profileName)
	if entry != nil {
		recreate = authLoginCommand + " --host " + shellquote.BashArg(entry.Host) + " --profile " + shellquote.BashArg(profileName)
		if entry.ConfigFile != "" {
			recreate = "DATABRICKS_CONFIG_FILE=" + shellquote.BashArg(path) + " " + recreate
		}
	}
	message := fmt.Sprintf("MCP server %s is bound to Databricks profile \"%s\", which %s in %s. Recreate it with: %s or remove the server with: %s",
		name, profileName, problem, path, recreate, removeCommand(profileName))
	return &BindError{Category: BindProfileNotFound, message: message}
}

// driftError is the PROFILE_HOST_CHANGED error for an entry whose profile now names another host,
// or another workspace on the same host.
func driftError(name, profileName string, entry Entry, drift Drift) error {
	recordedHost, recordedWorkspaceID := recordedTarget(entry)
	var message string
	if NormalizeHost(recordedHost) != NormalizeHost(drift.CurrentHost) {
		message = fmt.Sprintf("MCP server %s was set up for %s, but profile \"%s\" now points at %s. Run: %s and add it again",
			name, recordedHost, profileName, drift.CurrentHost, removeCommand(profileName))
	} else {
		message = fmt.Sprintf("MCP server %s was set up for workspace %s, but profile \"%s\" now points at workspace %s. Run: %s and add it again",
			name, recordedWorkspaceID, profileName, drift.CurrentWorkspaceID, removeCommand(profileName))
	}
	return &BindError{Category: BindProfileHostChanged, message: message}
}

// configFileError is the PROFILE_HOST_CHANGED error for an entry whose config file is not the one in
// use. The profile may name the same host there, but the server was set up against the recorded file.
func configFileError(name, profileName, recordedPath, currentPath string) error {
	message := fmt.Sprintf("MCP server %s was set up with config file %s, but profile \"%s\" is read from %s now. Run: %s and add it again, or set DATABRICKS_CONFIG_FILE to %s",
		name, recordedPath, profileName, currentPath, removeCommand(profileName), shellquote.BashArg(recordedPath))
	return &BindError{Category: BindProfileHostChanged, message: message}
}

// AuthError is the AUTH_FAILED error for a credential that the profile's own settings could not
// provide. The message points at the login command for the profile.
func AuthError(profileName string, cause error) error {
	message := fmt.Sprintf("%s\nRun: %s --profile %s", cause, authLoginCommand, shellquote.BashArg(profileName))
	return &BindError{Category: BindAuthFailed, message: message, cause: cause}
}

// findEntry returns the entry for the service and profile whose config file is configFile, or nil when
// the service and profile have no entry at all. Both sides are compared as resolved paths, so an entry
// with no pinned file matches the default file.
//
// An entry for the service and profile in another file is an error, not a missing entry: the server was
// set up against that file, and binding through this one would switch it without a record. An entry
// whose file cannot be resolved, such as a "~" path with no home directory, cannot be compared. The
// bind fails on it, since it might be the match.
func findEntry(ctx context.Context, state *State, service, profileName, configFile string) (*Entry, error) {
	var unresolved error
	var other *Entry
	var otherPath string
	for i := range state.Servers {
		entry := &state.Servers[i]
		if entry.Service != service || entry.Profile != profileName {
			continue
		}
		path, err := ResolveConfigFile(ctx, entry.ConfigFile)
		if err != nil {
			unresolved = errors.Join(unresolved, fmt.Errorf("MCP server %s: %w", entry.Name, err))
			continue
		}
		if samePath(path, configFile) {
			return entry, nil
		}
		if other == nil {
			other, otherPath = entry, path
		}
	}
	if unresolved != nil {
		return nil, uncategorizedError(unresolved)
	}
	if other != nil {
		return nil, configFileError(other.Name, profileName, filepath.ToSlash(otherPath), filepath.ToSlash(configFile))
	}
	return nil, nil
}

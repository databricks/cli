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
	// Stderr receives the notice printed when no install record exists.
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
// The profile is checked before any credential is resolved, so a re-pointed host is never
// contacted. The config file is the one in the context, the same file the profile lookup reads.
func Bind(ctx context.Context, opts BindOptions) (*config.Config, *Entry, error) {
	configFile, err := resolveConfigFile(ctx, env.Get(ctx, mcpconfig.ConfigFileEnv))
	if err != nil {
		return nil, nil, &BindError{Category: BindUncategorized, message: err.Error(), cause: err}
	}
	entry, err := findEntry(ctx, opts.State, opts.Service, opts.Profile, configFile)
	if err != nil {
		return nil, nil, &BindError{Category: BindUncategorized, message: err.Error(), cause: err}
	}
	name := opts.Service
	if entry != nil {
		name = entry.Name
	}

	profiler := profile.GetProfiler(ctx)
	profiles, err := profiler.LoadProfiles(ctx, profile.WithName(opts.Profile))
	if err != nil && !errors.Is(err, profile.ErrNoConfiguration) {
		return nil, nil, &BindError{Category: BindUncategorized, message: err.Error(), cause: err}
	}
	if len(profiles) == 0 {
		path, err := profiler.GetPath(ctx)
		if err != nil {
			return nil, nil, &BindError{Category: BindUncategorized, message: err.Error(), cause: err}
		}
		return nil, nil, profileGoneError(name, opts.Profile, filepath.ToSlash(path), entry)
	}
	p := profiles[0]

	if entry != nil && hostChanged(*entry, p) {
		message := fmt.Sprintf("MCP server %s was set up for %s, but profile \"%s\" now points at %s. Run: %s --profile %s and add it again",
			name, entry.Host, opts.Profile, p.Host, mcpRemoveCommand, opts.Profile)
		return nil, nil, &BindError{Category: BindProfileHostChanged, message: message}
	}
	if entry != nil && workspaceChanged(*entry, p) {
		message := fmt.Sprintf("MCP server %s was set up for workspace %s, but profile \"%s\" now points at workspace %s. Run: %s --profile %s and add it again",
			name, entry.WorkspaceID, opts.Profile, profileWorkspaceID(p), mcpRemoveCommand, opts.Profile)
		return nil, nil, &BindError{Category: BindProfileHostChanged, message: message}
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
		return nil, nil, authError(opts.Profile, err)
	}

	if entry == nil {
		fmt.Fprintf(opts.Stderr, "No install record for %s on profile \"%s\"; using its current host %s\n", opts.Service, opts.Profile, cfg.Host)
	}

	// Authenticate only writes headers onto req, which is never sent. The request takes
	// the command's context, as in cmd/root/auth.go.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "", nil)
	if err != nil {
		return nil, nil, &BindError{Category: BindUncategorized, message: err.Error(), cause: err}
	}
	if err := cfg.Authenticate(req); err != nil {
		return nil, nil, authError(opts.Profile, err)
	}
	return cfg, entry, nil
}

// profileGoneError builds the error for a profile that is missing from the config file.
// Without a state entry the recorded host is unknown, so the recreate hint omits --host.
func profileGoneError(name, profileName, path string, entry *Entry) error {
	recreate := authLoginCommand + " --profile " + profileName
	if entry != nil {
		recreate = authLoginCommand + " --host " + entry.Host + " --profile " + profileName
	}
	message := fmt.Sprintf("MCP server %s is bound to Databricks profile \"%s\", which no longer exists in %s. Recreate it with: %s   or remove the server with: %s --profile %s",
		name, profileName, path, recreate, mcpRemoveCommand, profileName)
	return &BindError{Category: BindProfileNotFound, message: message}
}

func authError(profileName string, cause error) error {
	message := fmt.Sprintf("%s\nRun: %s --profile %s", cause, authLoginCommand, profileName)
	return &BindError{Category: BindAuthFailed, message: message, cause: cause}
}

// findEntry returns the entry for the service and profile whose config file is configFile, or nil.
// Both are compared as resolved paths, so an entry with no pinned file matches the default file.
func findEntry(ctx context.Context, state *State, service, profileName, configFile string) (*Entry, error) {
	for i := range state.Servers {
		entry := &state.Servers[i]
		if entry.Service != service || entry.Profile != profileName {
			continue
		}
		path, err := resolveConfigFile(ctx, entry.ConfigFile)
		if err != nil {
			return nil, err
		}
		if path == configFile {
			return entry, nil
		}
	}
	return nil, nil
}

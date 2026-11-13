package mcp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/libs/aitools/mcpconfig"
	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/shellquote"
	"github.com/databricks/databricks-sdk-go/config"
)

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
	entry, err := CheckBinding(ctx, opts.State, opts.Service, opts.Profile, configFile)
	if err != nil {
		return nil, nil, err
	}

	cfg, err := BoundConfig(opts.Profile, configFile)
	if err != nil {
		return nil, nil, err
	}
	if entry == nil {
		fmt.Fprintf(opts.Stderr, "No install record for %s on profile \"%s\"; using its current host %s\n", opts.Service, opts.Profile, cfg.Host)
	}
	if err := AuthenticateBound(ctx, cfg, opts.Profile); err != nil {
		return nil, nil, err
	}
	return cfg, entry, nil
}

// BoundConfig returns the config of profileName in configFile, which is resolved, with the loaders that Bind uses: the
// profile's own settings and the non-auth environment. The profile's loader chain would also fill missing credentials
// from the environment, and a token meant for another workspace must not reach this one. It does not authenticate; see
// AuthenticateBound.
func BoundConfig(profileName, configFile string) (*config.Config, error) {
	cfg := &config.Config{
		Profile:    profileName,
		ConfigFile: configFile,
		Loaders:    []config.Loader{databrickscfg.ResolveNonAuthFromEnv, config.ConfigFile},
	}
	if err := cfg.EnsureResolved(); err != nil {
		// Configuration that the SDK rejects is not a credential failure, so a login would not fix it.
		return nil, uncategorizedError(err)
	}
	return cfg, nil
}

// AuthenticateBound checks that cfg, from BoundConfig for profileName, has a credential. A credential the profile cannot
// provide is AUTH_FAILED, with the login command for the profile.
func AuthenticateBound(ctx context.Context, cfg *config.Config, profileName string) error {
	// Authenticate only writes headers onto req, which is never sent. The request takes
	// the command's context, as in cmd/root/auth.go.
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "", nil)
	if err != nil {
		return uncategorizedError(err)
	}
	if err := cfg.Authenticate(req); err != nil {
		return AuthError(profileName, err)
	}
	return nil
}

// CheckBinding returns the state entry for service and profileName bound to configFile, the config file the caller
// resolved with ResolveConfigFile, or nil when there is none. It returns the error Bind returns when the profile does not
// match: the profile is gone or has no host, or with an entry, its host or workspace changed. The profile is checked
// without an entry too, under the service's name. It reads the config files only and makes no request, so a caller can
// refuse before it sends a token to the profile's host.
func CheckBinding(ctx context.Context, state *State, service, profileName, configFile string) (*Entry, error) {
	entry, err := FindEntry(ctx, state, service, profileName, configFile)
	if err != nil {
		return nil, err
	}
	if entry == nil {
		_, err := profileFor(service, profileName, configFile, nil)
		return nil, err
	}
	p, err := profileFor(entry.Name, profileName, configFile, entry)
	if err != nil {
		return nil, err
	}
	if drift := driftOf(*entry, p); drift != nil {
		return nil, driftError(entry.Name, profileName, *entry, *drift)
	}
	return entry, nil
}

// profileFor returns the profile profileName in configFile, or the PROFILE_NOT_FOUND error that says it
// is gone. name and entry describe the state entry the profile serves; entry is nil without one.
func profileFor(name, profileName, configFile string, entry *Entry) (profile.Profile, error) {
	p, err := configFiles{}.profile(configFile, profileName)
	switch {
	case errors.Is(err, errProfileMissing):
		problem := "no longer exists"
		if entry == nil {
			// Nothing records that the profile existed: the server is not installed yet, or its entry was written by hand.
			problem = "does not exist"
		}
		return profile.Profile{}, profileGoneError(name, profileName, filepath.ToSlash(configFile), entry, problem)
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

// RemoveNamesCommand returns the command that removes the server named name. Server names are unique in the state.
func RemoveNamesCommand(name string) string {
	return mcpRemoveCommand + " --names " + shellquote.BashArg(name)
}

// RecreateCommand returns the command that logs in to host for profileName again. configFile is the file the login must
// write, passed as DATABRICKS_CONFIG_FILE; pass "" when the login's default file is that file.
func RecreateCommand(host, profileName, configFile string) string {
	command := authLoginCommand + " --host " + shellquote.BashArg(host) + " --profile " + shellquote.BashArg(profileName)
	if configFile != "" {
		command = "DATABRICKS_CONFIG_FILE=" + shellquote.BashArg(configFile) + " " + command
	}
	return command
}

// profileGoneError builds the PROFILE_NOT_FOUND error for a profile that the config file does not
// name with a host, either because it is absent or because its section has no host. problem completes
// the sentence "which ... in <path>". Without a state entry nothing was set up for the profile, so the
// message only says how to set the profile up, without a host. An entry pinned to a config file puts that
// file in the recreate hint.
func profileGoneError(name, profileName, path string, entry *Entry, problem string) error {
	if entry == nil {
		login := authLoginCommand + " --profile " + shellquote.BashArg(profileName)
		message := fmt.Sprintf("MCP server %s needs Databricks profile \"%s\", which %s in %s. Set it up with: %s", name, profileName, problem, path, login)
		return &BindError{Category: BindProfileNotFound, message: message}
	}
	pinnedPath := ""
	if entry.ConfigFile != "" {
		pinnedPath = path
	}
	recreate := RecreateCommand(entry.Host, profileName, pinnedPath)
	message := fmt.Sprintf("MCP server %s is bound to Databricks profile \"%s\", which %s in %s. Recreate it with: %s or remove the server with: %s",
		name, profileName, problem, path, recreate, RemoveNamesCommand(name))
	return &BindError{Category: BindProfileNotFound, message: message}
}

// DriftMessage tells how the profile of entry, which is named profileName, now differs from what the entry recorded: its
// host, or its workspace when the host is the same. subject starts the sentence, for example "MCP server github-logfood"
// or "it". The serve errors and `mcp list` both word a drift with it.
func DriftMessage(subject string, entry Entry, profileName string, drift Drift) string {
	recordedHost, recordedWorkspaceID := recordedTarget(entry)
	if NormalizeHost(recordedHost) != NormalizeHost(drift.CurrentHost) {
		return fmt.Sprintf("%s was set up for %s, but profile \"%s\" now points at %s", subject, recordedHost, profileName, drift.CurrentHost)
	}
	return fmt.Sprintf("%s was set up for workspace %s, but profile \"%s\" now points at workspace %s", subject, recordedWorkspaceID, profileName, drift.CurrentWorkspaceID)
}

// driftError is the PROFILE_HOST_CHANGED error for an entry whose profile now names another host,
// or another workspace on the same host.
func driftError(name, profileName string, entry Entry, drift Drift) error {
	message := fmt.Sprintf("%s. Run: %s and add it again", DriftMessage("MCP server "+name, entry, profileName, drift), RemoveNamesCommand(name))
	return &BindError{Category: BindProfileHostChanged, message: message}
}

// configFileError is the PROFILE_HOST_CHANGED error for an entry whose config file is not the one in
// use. The profile may name the same host there, but the server was set up against the recorded file.
func configFileError(name, profileName, recordedPath, currentPath string) error {
	message := fmt.Sprintf("MCP server %s was set up with config file %s, but profile \"%s\" is read from %s now. Run: %s and add it again, or set DATABRICKS_CONFIG_FILE to %s",
		name, recordedPath, profileName, currentPath, RemoveNamesCommand(name), shellquote.BashArg(recordedPath))
	return &BindError{Category: BindProfileHostChanged, message: message}
}

// AuthError is the AUTH_FAILED error for a credential that the profile's own settings could not
// provide. The message points at the login command for the profile.
func AuthError(profileName string, cause error) error {
	message := fmt.Sprintf("%s\nRun: %s --profile %s", cause, authLoginCommand, shellquote.BashArg(profileName))
	return &BindError{Category: BindAuthFailed, message: message, cause: cause}
}

// FindEntry returns the entry for the service and profile whose config file is configFile, or nil when
// the service and profile have no entry at all. Service names are compared without case, as Unity Catalog
// names are. Both sides are compared as resolved paths, so an entry with no pinned file matches the default file.
//
// An entry for the service and profile in another file is an error, not a missing entry: the server was
// set up against that file, and binding through this one would switch it without a record. An entry
// whose file cannot be resolved, such as a "~" path with no home directory, cannot be compared. The
// bind fails on it, since it might be the match.
func FindEntry(ctx context.Context, state *State, service, profileName, configFile string) (*Entry, error) {
	var unresolved error
	var other *Entry
	var otherPath string
	for i := range state.Servers {
		entry := &state.Servers[i]
		if !strings.EqualFold(entry.Service, service) || entry.Profile != profileName {
			continue
		}
		path, err := ResolveConfigFile(ctx, entry.ConfigFile)
		if err != nil {
			unresolved = errors.Join(unresolved, fmt.Errorf("MCP server %s: %w", entry.Name, err))
			continue
		}
		if SamePath(path, configFile) {
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

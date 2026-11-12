package mcp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"

	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/databricks-sdk-go/config"
)

// bindExitCode is the exit status for binding failures. Agents and `list --check`
// use it to tell binding failures apart from transport errors (SPEC 11.2).
const bindExitCode = 2

// BindCategory is the error category for a binding failure. Values match the
// AitoolsErrorCategory names so telemetry can map them directly.
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

// BindError is a failed binding check. Its message is the stderr text, byte-exact with SPEC 11.2.
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

// Bind runs the binding checks of SPEC 11.2 for one profile and service. It never reads
// stdin, prompts, or loads a bundle. It returns the resolved config and the matching state
// entry, which is nil when none exists.
func Bind(ctx context.Context, opts BindOptions) (*config.Config, *Entry, error) {
	entry := findEntry(opts.State, opts.Service, opts.Profile)
	name := opts.Service
	if entry != nil {
		name = entry.Name
	}

	profiler := profile.GetProfiler(ctx)
	path, err := profiler.GetPath(ctx)
	if err != nil {
		return nil, nil, &BindError{Category: BindUncategorized, message: err.Error(), cause: err}
	}
	profiles, err := profiler.LoadProfiles(ctx, profile.WithName(opts.Profile))
	if err != nil && !errors.Is(err, profile.ErrNoConfiguration) {
		return nil, nil, &BindError{Category: BindUncategorized, message: err.Error(), cause: err}
	}
	if len(profiles) == 0 {
		return nil, nil, profileGoneError(name, opts.Profile, path, entry)
	}

	cfg := &config.Config{Profile: opts.Profile, Loaders: databrickscfg.ProfileAuthLoaders}
	if err := cfg.EnsureResolved(); err != nil {
		return nil, nil, authError(opts.Profile, err)
	}

	if entry != nil && NormalizeHost(entry.Host) != NormalizeHost(cfg.Host) {
		message := fmt.Sprintf("MCP server %s was set up for %s, but profile \"%s\" now points at %s. Run: databricks aitools mcp remove --profile %s and add it again.",
			name, entry.Host, opts.Profile, cfg.Host, opts.Profile)
		return nil, nil, &BindError{Category: BindProfileHostChanged, message: message}
	}

	if entry == nil {
		fmt.Fprintf(opts.Stderr, "No install record for %s on profile \"%s\"; using its current host %s.\n", opts.Service, opts.Profile, cfg.Host)
	}

	if err := cfg.Authenticate(&http.Request{Header: make(http.Header)}); err != nil {
		return nil, nil, authError(opts.Profile, err)
	}
	return cfg, entry, nil
}

// profileGoneError builds the SPEC 11.2 step 2 error. Without a state entry the
// recorded host is unknown, so the recreate hint omits --host.
func profileGoneError(name, profileName, path string, entry *Entry) error {
	recreate := "databricks auth login --profile " + profileName
	if entry != nil {
		recreate = "databricks auth login --host " + entry.Host + " --profile " + profileName
	}
	message := fmt.Sprintf("MCP server %s is bound to Databricks profile \"%s\", which no longer exists in %s. Recreate it with: %s   or remove the server with: databricks aitools mcp remove --profile %s",
		name, profileName, path, recreate, profileName)
	return &BindError{Category: BindProfileNotFound, message: message}
}

func authError(profileName string, cause error) error {
	message := fmt.Sprintf("%s\nRun: databricks auth login --profile %s", cause, profileName)
	return &BindError{Category: BindAuthFailed, message: message, cause: cause}
}

func findEntry(state *State, service, profileName string) *Entry {
	i := slices.IndexFunc(state.Servers, func(e Entry) bool {
		return e.Service == service && e.Profile == profileName
	})
	if i < 0 {
		return nil
	}
	return &state.Servers[i]
}

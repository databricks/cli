// Package managedstate exposes the configuration surface that gates the
// bundle's use of the Deployment Metadata Service (DMS) for server-managed
// deployment state and locking.
//
// The setting can be controlled via two routes, in priority order:
//  1. The bundle.managed_state field in databricks.yml.
//  2. The DATABRICKS_BUNDLE_MANAGED_STATE environment variable.
//
// When neither route opts in, the bundle falls back to the historical
// workspace-filesystem-based state and lock implementation. Use
// cmd/bundle/utils.ResolveManagedStateSetting to combine the two sources
// with location-aware source attribution.
package managedstate

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/env"
)

// EnvVar is the environment variable that opts the bundle into managed
// (server-side) deployment state.
const EnvVar = "DATABRICKS_BUNDLE_MANAGED_STATE"

// Default is used when the user has not opted in via config or env.
const Default = false

// Setting represents a resolved managed-state setting along with where it
// came from. Source is a human-readable string used in log lines and error
// messages (e.g. "bundle.managed_state setting at databricks.yml:3:5" or
// "DATABRICKS_BUNDLE_MANAGED_STATE environment variable").
type Setting struct {
	Enabled bool
	Source  string
}

// FromEnv reads the DATABRICKS_BUNDLE_MANAGED_STATE environment variable.
//
// Accepts the standard strconv.ParseBool spellings (1/0, t/T, true/True/TRUE,
// f/F, false/False/FALSE) and additionally accepts "yes"/"no"/"y"/"n"
// case-insensitively. An unset or empty value returns (false, false, nil) --
// isSet=false signals that the env var was not set, not that it parsed to
// false. Invalid values return an error.
func FromEnv(ctx context.Context) (value, isSet bool, err error) {
	raw := env.Get(ctx, EnvVar)
	if raw == "" {
		return false, false, nil
	}

	switch strings.ToLower(raw) {
	case "yes", "y":
		return true, true, nil
	case "no", "n":
		return false, true, nil
	}

	parsed, parseErr := strconv.ParseBool(raw)
	if parseErr != nil {
		return false, true, fmt.Errorf("unexpected setting for %s=%q (expected a boolean value)", EnvVar, raw)
	}
	return parsed, true, nil
}

// Resolve combines the bundle.managed_state config field and the
// DATABRICKS_BUNDLE_MANAGED_STATE environment variable into a single
// Setting with source attribution.
//
// Priority: configEnabled (i.e. bundle.managed_state=true in databricks.yml)
// > env var > Default. configValue is the dyn.Value for the bundle root and
// is used only for file/line/column source attribution; pass dyn.InvalidValue
// if a location isn't available.
func Resolve(ctx context.Context, configEnabled bool, configValue dyn.Value) (Setting, error) {
	if configEnabled {
		source := "bundle.managed_state setting"
		v := dyn.GetValue(configValue, "bundle.managed_state")
		if locs := v.Locations(); len(locs) > 0 {
			loc := locs[0]
			source = fmt.Sprintf("bundle.managed_state setting at %s:%d:%d", filepath.ToSlash(loc.File), loc.Line, loc.Column)
		}
		return Setting{Enabled: true, Source: source}, nil
	}

	envValue, isSet, err := FromEnv(ctx)
	if err != nil {
		return Setting{}, err
	}
	if isSet {
		return Setting{Enabled: envValue, Source: EnvVar + " environment variable"}, nil
	}

	return Setting{Enabled: Default}, nil
}

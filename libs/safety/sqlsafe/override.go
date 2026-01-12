package sqlsafe

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/databricks/databricks-sdk-go/config"

	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/env"
)

// OverrideHelper resolves layered overrides (flag, env var, profile) for the SQL
// safety gate and standardizes the user-facing messaging.
type OverrideHelper struct {
	// FlagName is the CLI flag users can pass (for messaging only).
	FlagName string

	// EnvVar is the environment variable name to inspect.
	EnvVar string

	// ProfileKey is the configuration key inside ~/.databrickscfg.
	ProfileKey string
}

// Resolve returns true when the profile/env/flag chain enables destructive SQL.
// Precedence (highest to lowest): profile, environment, CLI flag.
func (h OverrideHelper) Resolve(ctx context.Context, cfg *config.Config, flagSet, flagValue bool) (bool, error) {
	if val, ok, err := h.resolveFromProfile(ctx, cfg); err != nil {
		return false, err
	} else if ok {
		return val, nil
	}

	if val, ok, err := h.resolveFromEnv(ctx); err != nil {
		return false, err
	} else if ok {
		return val, nil
	}

	if flagSet {
		return flagValue, nil
	}

	return false, nil
}

func (h OverrideHelper) resolveFromEnv(ctx context.Context) (bool, bool, error) {
	raw, ok := env.Lookup(ctx, h.EnvVar)
	if !ok {
		return false, false, nil
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false, false, nil
	}
	val, err := strconv.ParseBool(raw)
	if err != nil {
		return false, false, fmt.Errorf("invalid value for %s: %w", h.EnvVar, err)
	}
	return val, true, nil
}

func (h OverrideHelper) resolveFromProfile(ctx context.Context, cfg *config.Config) (bool, bool, error) {
	if cfg == nil {
		return false, false, nil
	}

	profileName := cfg.Profile
	if profileName == "" {
		profileName = "DEFAULT"
	}

	profiler := profile.FileProfilerImpl{}
	file, err := profiler.Get(ctx)
	if err != nil {
		if errors.Is(err, profile.ErrNoConfiguration) {
			return false, false, nil
		}
		return false, false, err
	}

	section := file.Section(profileName)
	if section == nil || !section.HasKey(h.ProfileKey) {
		return false, false, nil
	}

	raw := strings.TrimSpace(section.Key(h.ProfileKey).String())
	if raw == "" {
		return false, false, nil
	}

	val, err := strconv.ParseBool(raw)
	if err != nil {
		return false, false, fmt.Errorf("invalid boolean for %s in profile %s: %s", h.ProfileKey, profileName, raw)
	}
	return val, true, nil
}

// BlockedError decorates the classifier error with override instructions.
func (h OverrideHelper) BlockedError(err error) error {
	return fmt.Errorf("%w. Pass %s, set %s=true, or add %s=true to your profile to override", err, h.FlagName, h.EnvVar, h.ProfileKey)
}

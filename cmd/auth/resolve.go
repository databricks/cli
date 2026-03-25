package auth

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/libs/databrickscfg/profile"
)

// looksLikeHost returns true if the argument looks like a host URL rather than
// a profile name. Profile names are short identifiers (e.g., "logfood",
// "DEFAULT"), while host URLs contain dots or start with "http".
func looksLikeHost(arg string) bool {
	return strings.Contains(arg, ".") || strings.HasPrefix(arg, "http://") || strings.HasPrefix(arg, "https://")
}

// resolvePositionalArg resolves a positional argument to either a profile name
// or a host. It tries the argument as a profile name first. If no profile
// matches and the argument looks like a host URL, it returns it as a host. If
// no profile matches and the argument does not look like a host, it returns an
// error.
func resolvePositionalArg(ctx context.Context, arg string, profiler profile.Profiler) (profileName string, host string, err error) {
	candidateProfile, err := loadProfileByName(ctx, arg, profiler)
	if err != nil {
		return "", "", err
	}
	if candidateProfile != nil {
		return arg, "", nil
	}

	if looksLikeHost(arg) {
		return "", arg, nil
	}

	return "", "", fmt.Errorf("no profile named %q found", arg)
}

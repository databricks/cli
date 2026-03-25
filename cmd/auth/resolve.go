package auth

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/databricks-sdk-go/config"
)

// looksLikeHost returns true if the argument looks like a host URL rather than
// a profile name. Profile names are short identifiers (e.g., "logfood",
// "DEFAULT"), while host URLs contain dots or start with "http".
func looksLikeHost(arg string) bool {
	if strings.Contains(arg, ".") || strings.HasPrefix(arg, "http://") || strings.HasPrefix(arg, "https://") {
		return true
	}
	// Match host:port pattern without dots or scheme (e.g., localhost:8080).
	if i := strings.LastIndex(arg, ":"); i > 0 {
		if _, err := strconv.Atoi(arg[i+1:]); err == nil {
			return true
		}
	}
	return false
}

// resolvePositionalArg resolves a positional argument to either a profile name
// or a host. It tries the argument as a profile name first. If no profile
// matches and the argument looks like a host URL, it returns it as a host. If
// no profile matches and the argument does not look like a host, it returns an
// error.
func resolvePositionalArg(ctx context.Context, arg string, profiler profile.Profiler) (profileName, host string, err error) {
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

// resolveHostToProfile resolves a host URL to a profile name. If multiple
// profiles match the host, it prompts the user to select one (or errors in
// non-interactive mode). If no profiles match, it returns an error.
func resolveHostToProfile(ctx context.Context, host string, profiler profile.Profiler) (string, error) {
	canonicalHost := (&config.Config{Host: host}).CanonicalHostName()
	hostProfiles, err := profiler.LoadProfiles(ctx, profile.WithHost(canonicalHost))
	if err != nil {
		return "", err
	}

	switch len(hostProfiles) {
	case 1:
		return hostProfiles[0].Name, nil
	case 0:
		allProfiles, err := profiler.LoadProfiles(ctx, profile.MatchAllProfiles)
		if err != nil {
			return "", fmt.Errorf("no profile found matching host %q", host)
		}
		names := strings.Join(allProfiles.Names(), ", ")
		return "", fmt.Errorf("no profile found matching host %q. Available profiles: %s", host, names)
	default:
		if cmdio.IsPromptSupported(ctx) {
			selected, err := profile.SelectProfile(ctx, profile.SelectConfig{
				Label:             fmt.Sprintf("Multiple profiles found for %q. Select one to use", host),
				Profiles:          hostProfiles,
				StartInSearchMode: len(hostProfiles) > 5,
				ActiveTemplate:    "▸ {{.PaddedName | bold}}{{if .AccountID}} (account: {{.AccountID}}){{else}} ({{.Host}}){{end}}",
				InactiveTemplate:  "  {{.PaddedName}}{{if .AccountID}} (account: {{.AccountID | faint}}){{else}} ({{.Host | faint}}){{end}}",
				SelectedTemplate:  `{{ "Selected profile" | faint }}: {{ .Name | bold }}`,
			})
			if err != nil {
				return "", err
			}
			return selected, nil
		}
		names := strings.Join(hostProfiles.Names(), ", ")
		return "", fmt.Errorf("multiple profiles found matching host %q: %s. Please specify the profile name directly", host, names)
	}
}

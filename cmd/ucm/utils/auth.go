package utils

import (
	"errors"
	"fmt"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/spf13/cobra"
)

// MustWorkspaceClient is the UCM analogue of root.MustWorkspaceClient. It
// loads ucm.yml, selects the target, resolves a workspace client from
// workspace.host (matching against ~/.databrickscfg) and installs it on the
// command context. Ambiguous matches are resolved interactively, mirroring
// DAB's configureBundle + resolveProfileAmbiguity flow in cmd/root/bundle.go.
func MustWorkspaceClient(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	if !logdiag.IsSetup(ctx) {
		ctx = logdiag.InitContext(ctx)
		cmd.SetContext(ctx)
	}

	u := ProcessUcm(cmd, ProcessOptions{})
	ctx = cmd.Context()
	if u == nil || logdiag.HasError(ctx) {
		return root.ErrAlreadyPrinted
	}

	client, err := u.WorkspaceClientE()
	if err != nil {
		names, isMulti := databrickscfg.AsMultipleProfiles(err)
		if !isMulti {
			logdiag.LogError(ctx, err)
			return root.ErrAlreadyPrinted
		}

		selected, resolveErr := resolveProfileAmbiguity(cmd, u.Config.Workspace.Host, err, names)
		if resolveErr != nil {
			logdiag.LogError(ctx, resolveErr)
			return root.ErrAlreadyPrinted
		}

		u.Config.Workspace.Profile = selected
		u.ClearWorkspaceClient()
		client, err = u.WorkspaceClientE()
		if err != nil {
			logdiag.LogError(ctx, err)
			return root.ErrAlreadyPrinted
		}
	}

	ctx = cmdctx.SetConfigUsed(ctx, client.Config)
	ctx = cmdctx.SetWorkspaceClient(ctx, client)
	cmd.SetContext(ctx)
	return nil
}

// resolveProfileAmbiguity is the UCM copy of cmd/root/bundle.go:resolveProfileAmbiguity.
// The DAB version takes a *bundle.Bundle which we can't import here, so we
// pass the host URL directly — it's the only Bundle field the upstream
// function actually reads.
func resolveProfileAmbiguity(cmd *cobra.Command, host string, originalErr error, names []string) (string, error) {
	ctx := cmd.Context()

	namesMatcher := profile.MatchProfileNames(names...)
	profiler := profile.GetProfiler(ctx)
	profiles, err := profiler.LoadProfiles(ctx, func(p profile.Profile) bool {
		return namesMatcher(p) && profile.MatchWorkspaceProfiles(p)
	})
	if err != nil {
		if errors.Is(err, profile.ErrNoConfiguration) {
			return "", originalErr
		}
		return "", err
	}

	switch len(profiles) {
	case 0:
		return "", originalErr
	case 1:
		return profiles[0].Name, nil
	}

	_, hasProfileFlag := profileFlagValue(cmd)
	if hasProfileFlag || !cmdio.IsPromptSupported(ctx) {
		return "", fmt.Errorf(
			"%w\n\nMatching workspace profiles: %s\n\n"+
				"Fix (pick one):\n"+
				"  1. Set profile in ucm.yml:\n"+
				"       workspace:\n"+
				"         profile: %s\n"+
				"  2. Pass a flag:\n"+
				"       %s --profile %s\n"+
				"  3. Set env var:\n"+
				"       DATABRICKS_CONFIG_PROFILE=%s",
			originalErr,
			strings.Join(profiles.Names(), ", "),
			profiles[0].Name,
			cmd.CommandPath(),
			profiles[0].Name,
			profiles[0].Name,
		)
	}

	return profile.SelectProfile(ctx, profile.SelectConfig{
		Label:             "Multiple profiles match host " + host,
		Profiles:          profiles,
		StartInSearchMode: true,
		ActiveTemplate:    `{{.Name | bold}}{{if .AccountID}} (account: {{.AccountID|faint}}){{end}}{{if .WorkspaceID}} (workspace: {{.WorkspaceID|faint}}){{end}}`,
		InactiveTemplate:  `{{.Name}}{{if .AccountID}} (account: {{.AccountID}}){{end}}{{if .WorkspaceID}} (workspace: {{.WorkspaceID}}){{end}}`,
		SelectedTemplate:  `{{ "Using profile" | faint }}: {{ .Name | bold }}`,
	})
}

func profileFlagValue(cmd *cobra.Command) (string, bool) {
	flag := cmd.Flag("profile")
	if flag == nil {
		return "", false
	}
	value := flag.Value.String()
	return value, value != ""
}

package docker

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	authlib "github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/dockercredentials"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/spf13/cobra"
)

func newDockerConfigureCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "configure [PROFILE]",
		Short: "(Experimental) Configure Docker authentication for Databricks Artifact Registry",
		Long: `(Experimental) Configure Docker authentication for Databricks Artifact Registry.

This command installs docker-credential-databricks and configures Docker to use
it for the selected workspace's Artifact Registry host. If the selected profile
does not already include a workspace_id, the command resolves and saves it so
the Docker helper can map the registry host back to the profile. The registry
region is inferred from the workspace's metastore. Select the workspace with
[PROFILE] or --profile; --host, --account-id, and --workspace-id are not
supported. The deprecated --region flag is retained for compatibility; omit it
because it will be fully removed in the next release.`,
		Args: cobra.MaximumNArgs(1),
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			return errorOnUnsupportedDockerFlags(cmd, "[PROFILE] or --profile")
		},
	}
	var region string
	cmd.Flags().StringVar(&region, "region", "", "Artifact Registry region; we recommend omitting this flag because the region is inferred automatically")
	cmd.Flags().Lookup("region").Deprecated = "--region will be fully removed in the next release"
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		target, err := configureDockerTarget(cmd, args, region)
		if err != nil {
			return err
		}
		result, err := target.configure(cmd.Context())
		if err != nil {
			return err
		}
		target.logConfigured(cmd.Context(), result)
		return nil
	}
	return cmd
}

func configureDockerTarget(cmd *cobra.Command, args []string, region string) (*dockerTarget, error) {
	ctx := cmd.Context()
	name, err := configureDockerProfileName(ctx, cmd, args, profile.DefaultProfiler)
	if err != nil {
		return nil, err
	}
	var explicitRegion *string
	if cmd.Flags().Changed("region") {
		explicitRegion = &region
	}
	w, err := loadDockerWorkspace(ctx, name, explicitRegion)
	if err != nil {
		return nil, err
	}
	if err := dockercredentials.EnsureUniqueProfile(ctx, profile.DefaultProfiler, w.profile, w.id); err != nil {
		return nil, err
	}
	return w.target(ctx, explicitRegion)
}

func (t *dockerTarget) configure(ctx context.Context) (dockercredentials.Configuration, error) {
	if t.profile.WorkspaceID == "" || t.profile.WorkspaceID == authlib.WorkspaceIDNone {
		err := databrickscfg.SaveToProfile(ctx, &config.Config{
			ConfigFile:  env.Get(ctx, "DATABRICKS_CONFIG_FILE"),
			Profile:     t.profile.Name,
			WorkspaceID: t.workspaceID,
		})
		if err != nil {
			return dockercredentials.Configuration{}, fmt.Errorf("save workspace ID to profile %q: %w", t.profile.Name, err)
		}
	}
	return dockercredentials.Configure(ctx, t.executable, t.registryHost)
}

func (t *dockerTarget) logConfigured(ctx context.Context, result dockercredentials.Configuration) {
	cmdio.LogString(ctx, "Configured Docker credential helper for "+t.registryHost)
	cmdio.LogString(ctx, "Updated Docker config: "+filepath.ToSlash(result.ConfigPath))
	cmdio.LogString(ctx, "Installed Docker credential helper: "+filepath.ToSlash(result.Shim.Path))
	if !result.Shim.OnPath {
		installDir := filepath.ToSlash(filepath.Dir(result.Shim.Path))
		cmdio.LogString(ctx, fmt.Sprintf("Warning: ensure %s is on PATH before any other docker-credential-databricks helper, and that .CMD is in PATHEXT on Windows", installDir))
	}
}

func errorOnUnsupportedDockerFlags(cmd *cobra.Command, selection string) error {
	for _, name := range []string{"host", "account-id", "workspace-id"} {
		flag := cmd.Flag(name)
		if flag != nil && flag.Changed {
			return fmt.Errorf("--%s is not supported for auth docker %s. Select the workspace with %s instead", name, cmd.Name(), selection)
		}
	}
	return nil
}

// configureDockerProfileName resolves profile selectors from highest to lowest precedence, prompting only as a last resort.
func configureDockerProfileName(ctx context.Context, cmd *cobra.Command, args []string, profiler profile.Profiler) (string, error) {
	profileFlag := cmd.Flag("profile")
	profileName := ""
	if profileFlag != nil {
		profileName = profileFlag.Value.String()
	}
	if len(args) == 1 {
		if profileName != "" {
			return "", fmt.Errorf("argument %q cannot be combined with --profile. Use --profile instead", args[0])
		}
		return args[0], nil
	}
	if profileName != "" {
		return profileName, nil
	}
	if profileName = env.Get(ctx, "DATABRICKS_CONFIG_PROFILE"); profileName != "" {
		return profileName, nil
	}
	if profileName = databrickscfg.ResolveDefaultProfile(ctx); profileName != "" {
		return profileName, nil
	}
	if !cmdio.IsPromptSupported(ctx) {
		return "", errors.New("no profile specified. Use --profile <name> to specify which profile to use")
	}

	profiles, err := profiler.LoadProfiles(ctx, profile.MatchWorkspaceProfiles)
	if err != nil {
		return "", err
	}
	currentDefault, _ := databrickscfg.GetDefaultProfile(ctx, env.Get(ctx, "DATABRICKS_CONFIG_FILE"))
	return profile.SelectProfile(ctx, profile.SelectConfig{
		Label:             "Select a workspace profile",
		Profiles:          profiles,
		StartInSearchMode: true,
		Default:           currentDefault,
		ActiveTemplate:    `▸ {{.Name | bold}}{{if .IsDefault}} {{ "[default]" | green }}{{end}}{{if .AccountID}} (account: {{.AccountID|faint}}){{else if .Host}} ({{.Host|faint}}){{end}}`,
		InactiveTemplate:  `  {{.Name}}{{if .IsDefault}} [default]{{end}}{{if .AccountID}} (account: {{.AccountID|faint}}){{else if .Host}} ({{.Host|faint}}){{end}}`,
		SelectedTemplate:  `{{ "Using profile" | faint }}: {{ .Name | bold }}`,
	})
}

package docker

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	authlib "github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/dockercredentials"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/spf13/cobra"
)

type configureDockerDeps struct {
	dockerProfileDeps
	installShim         func(string) (dockercredentials.ShimInstallResult, error)
	setCredentialHelper func(string, string) error
}

func defaultConfigureDockerDeps() configureDockerDeps {
	return configureDockerDeps{
		dockerProfileDeps:   defaultDockerProfileDeps(),
		installShim:         dockercredentials.InstallShim,
		setCredentialHelper: dockercredentials.SetCredentialHelper,
	}
}

func newDockerConfigureCommand() *cobra.Command {
	return newDockerConfigureCommandWithDeps(defaultConfigureDockerDeps())
}

func newDockerConfigureCommandWithDeps(deps configureDockerDeps) *cobra.Command {
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
	}
	var regionFlag string
	cmd.Flags().StringVar(&regionFlag, "region", "", "Artifact Registry region; we recommend omitting this flag because the region is inferred automatically")
	cmd.Flags().Lookup("region").Deprecated = "--region will be fully removed in the next release"
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		if err := errorOnUnsupportedConfigureDockerFlags(cmd); err != nil {
			return err
		}

		profileName, err := configureDockerProfileName(ctx, cmd, args, deps.profiler)
		if err != nil {
			return err
		}

		p, err := loadAndValidateDockerProfile(ctx, profileName, deps.profiler)
		if err != nil {
			return err
		}
		if err := deps.validateWorkspaceHost(p.Host); err != nil {
			return err
		}
		regionProvided := cmd.Flags().Changed("region")
		region := strings.TrimSpace(regionFlag)
		if regionProvided {
			if err := dockercredentials.ValidateRegion(region); err != nil {
				return err
			}
		}

		executable, err := deps.executable()
		if err != nil {
			return fmt.Errorf("locate databricks executable: %w", err)
		}
		needsWorkspaceClient := !regionProvided || p.WorkspaceID == "" || p.WorkspaceID == authlib.WorkspaceIDNone
		var w *databricks.WorkspaceClient
		if needsWorkspaceClient {
			w, err = newDockerWorkspaceClient(ctx, p, executable, deps.dockerProfileDeps)
			if err != nil {
				return err
			}
		}
		workspaceID, err := resolveDockerWorkspaceID(ctx, p, w, deps.dockerProfileDeps)
		if err != nil {
			return err
		}
		if err := ensureConfigureDockerUniqueProfile(ctx, deps.profiler, p, workspaceID); err != nil {
			return err
		}
		if !regionProvided {
			w.Config.WorkspaceID = workspaceID
			region, err = deps.resolveWorkspaceRegion(ctx, w)
			if err != nil {
				return fmt.Errorf("resolve workspace region for profile %q: %w", p.Name, err)
			}
			region = strings.TrimSpace(region)
			if region == "" {
				return fmt.Errorf("resolve workspace region for profile %q: metastore summary did not include a region", p.Name)
			}
		}
		registryHost, err := deps.registryHost(workspaceID, region, p.Host)
		if err != nil {
			return err
		}
		if p.WorkspaceID == "" || p.WorkspaceID == authlib.WorkspaceIDNone {
			if err := persistConfigureDockerWorkspaceID(ctx, p, workspaceID); err != nil {
				return fmt.Errorf("save workspace ID to profile %q: %w", p.Name, err)
			}
		}

		shim, err := deps.installShim(executable)
		if err != nil {
			return fmt.Errorf("install Docker credential helper: %w", err)
		}
		dockerConfigPath, err := dockerConfigPath(ctx)
		if err != nil {
			return err
		}
		if err := deps.setCredentialHelper(dockerConfigPath, registryHost); err != nil {
			return fmt.Errorf("update Docker config %s: %w", filepath.ToSlash(dockerConfigPath), err)
		}

		cmdio.LogString(ctx, "Configured Docker credential helper for "+registryHost)
		cmdio.LogString(ctx, "Updated Docker config: "+filepath.ToSlash(dockerConfigPath))
		cmdio.LogString(ctx, "Installed Docker credential helper: "+filepath.ToSlash(shim.Path))
		if !shim.OnPath {
			installDir := filepath.ToSlash(filepath.Dir(shim.Path))
			cmdio.LogString(ctx, fmt.Sprintf("Warning: ensure %s is on PATH before any other docker-credential-databricks helper, and that .CMD is in PATHEXT on Windows", installDir))
		}
		return nil
	}

	return cmd
}

func errorOnUnsupportedConfigureDockerFlags(cmd *cobra.Command) error {
	for _, name := range []string{"host", "account-id", "workspace-id"} {
		flag := cmd.Flag(name)
		if flag != nil && flag.Changed {
			return fmt.Errorf("--%s is not supported for auth docker configure. Select the workspace with [PROFILE] or --profile instead", name)
		}
	}
	return nil
}

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

func loadAndValidateDockerProfile(ctx context.Context, profileName string, profiler profile.Profiler) (profile.Profile, error) {
	profiles, err := profiler.LoadProfiles(ctx, profile.WithName(profileName))
	if err != nil {
		return profile.Profile{}, err
	}
	if len(profiles) == 0 {
		return profile.Profile{}, fmt.Errorf("profile %q not found", profileName)
	}
	if err := validateDockerCredentialProfile(profiles[0]); err != nil {
		return profile.Profile{}, err
	}
	return profiles[0], nil
}

func newDockerWorkspaceClient(ctx context.Context, p profile.Profile, executable string, deps dockerProfileDeps) (*databricks.WorkspaceClient, error) {
	cfg := &databricks.Config{
		Profile:           p.Name,
		Host:              p.Host,
		AccountID:         p.AccountID,
		AuthType:          p.AuthType,
		ConfigFile:        env.Get(ctx, "DATABRICKS_CONFIG_FILE"),
		Loaders:           databrickscfg.ProfileAuthLoaders,
		DatabricksCliPath: executable,
	}
	w, err := deps.newWorkspaceClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("load workspace profile %q: %w. Run databricks auth login --host <workspace-url> and retry with that profile", p.Name, err)
	}
	return w, nil
}

func resolveDockerWorkspaceID(ctx context.Context, p profile.Profile, w *databricks.WorkspaceClient, deps dockerProfileDeps) (string, error) {
	if p.WorkspaceID != "" && p.WorkspaceID != authlib.WorkspaceIDNone {
		return p.WorkspaceID, nil
	}

	// The selected profile may contain the CLI-only "none" sentinel, which the SDK would send as a routing header.
	w.Config.WorkspaceID = ""
	workspaceID, err := deps.resolveWorkspaceID(ctx, w)
	if err != nil {
		return "", fmt.Errorf("resolve workspace ID for profile %q: %w. Run databricks auth login --host <workspace-url> and retry with that profile", p.Name, err)
	}
	return workspaceID, nil
}

// ensureConfigureDockerUniqueProfile rejects registry-to-profile mappings that would be ambiguous at credential lookup time.
func ensureConfigureDockerUniqueProfile(ctx context.Context, profiler profile.Profiler, p profile.Profile, workspaceID string) error {
	matches, err := profiler.LoadProfiles(ctx, func(candidate profile.Profile) bool {
		return candidate.WorkspaceID == workspaceID
	})
	if err != nil {
		return err
	}

	if p.WorkspaceID == "" || p.WorkspaceID == authlib.WorkspaceIDNone {
		matches = append(matches, p)
	}
	var validProfiles profile.Profiles
	for _, candidate := range matches {
		if validateDockerCredentialProfile(candidate) == nil {
			validProfiles = append(validProfiles, candidate)
		}
	}
	names := validProfiles.Names()
	if len(names) <= 1 {
		return nil
	}

	return fmt.Errorf("multiple Databricks profiles match workspace ID %s: %s. Remove duplicate workspace_id entries before using Docker credential helper", workspaceID, strings.Join(names, " and "))
}

func persistConfigureDockerWorkspaceID(ctx context.Context, p profile.Profile, workspaceID string) error {
	return databrickscfg.SaveToProfile(ctx, &config.Config{
		ConfigFile:  env.Get(ctx, "DATABRICKS_CONFIG_FILE"),
		Profile:     p.Name,
		WorkspaceID: workspaceID,
	})
}

func dockerConfigPath(ctx context.Context) (string, error) {
	if dockerConfig := env.Get(ctx, "DOCKER_CONFIG"); dockerConfig != "" {
		return filepath.Join(dockerConfig, "config.json"), nil
	}
	home, err := env.UserHomeDir(ctx)
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".docker", "config.json"), nil
}

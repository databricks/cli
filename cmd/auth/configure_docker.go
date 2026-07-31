package auth

import (
	"context"
	"errors"
	"fmt"
	"os"
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
	profiler            profile.Profiler
	newWorkspaceClient  func(*databricks.Config) (*databricks.WorkspaceClient, error)
	resolveWorkspaceID  func(context.Context, *databricks.WorkspaceClient) (string, error)
	executable          func() (string, error)
	registryHost        func(string, string, string) (string, error)
	installShim         func(context.Context, string, string) (dockercredentials.ShimInstallResult, error)
	setCredentialHelper func(string, string) error
}

func defaultConfigureDockerDeps() configureDockerDeps {
	return configureDockerDeps{
		profiler: profile.DefaultProfiler,
		newWorkspaceClient: func(cfg *databricks.Config) (*databricks.WorkspaceClient, error) {
			return databricks.NewWorkspaceClient(cfg)
		},
		resolveWorkspaceID:  authlib.ResolveWorkspaceID,
		executable:          os.Executable,
		registryHost:        dockercredentials.RegistryHost,
		installShim:         dockercredentials.InstallShim,
		setCredentialHelper: dockercredentials.SetCredentialHelper,
	}
}

func (d configureDockerDeps) withDefaults() configureDockerDeps {
	defaults := defaultConfigureDockerDeps()
	if d.profiler == nil {
		d.profiler = defaults.profiler
	}
	if d.newWorkspaceClient == nil {
		d.newWorkspaceClient = defaults.newWorkspaceClient
	}
	if d.resolveWorkspaceID == nil {
		d.resolveWorkspaceID = defaults.resolveWorkspaceID
	}
	if d.executable == nil {
		d.executable = defaults.executable
	}
	if d.registryHost == nil {
		d.registryHost = defaults.registryHost
	}
	if d.installShim == nil {
		d.installShim = defaults.installShim
	}
	if d.setCredentialHelper == nil {
		d.setCredentialHelper = defaults.setCredentialHelper
	}
	return d
}

func newConfigureDockerCommand() *cobra.Command {
	return newConfigureDockerCommandWithDeps(defaultConfigureDockerDeps())
}

func newConfigureDockerCommandWithDeps(deps configureDockerDeps) *cobra.Command {
	deps = deps.withDefaults()
	var region string

	cmd := &cobra.Command{
		Use:   "configure-docker [PROFILE]",
		Short: "Configure Docker authentication for Databricks Artifact Registry",
		Long: `Configure Docker authentication for Databricks Artifact Registry.

This command installs docker-credential-databricks and configures Docker to use
it for the selected workspace's Artifact Registry host. If the selected profile
does not already include a workspace_id, the command resolves and saves it so
the Docker helper can map the registry host back to the profile.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if err := rejectConfigureDockerAuthFlags(cmd); err != nil {
				return err
			}
			if region == "" {
				return errors.New("--region is required because workspace region cannot be inferred from this profile")
			}

			profileName, err := configureDockerProfileName(ctx, cmd, args, deps.profiler)
			if err != nil {
				return err
			}

			p, err := loadConfigureDockerProfile(ctx, profileName, deps.profiler)
			if err != nil {
				return err
			}
			if err := validateConfigureDockerProfile(p); err != nil {
				return err
			}
			if isConfigureDockerAccountOnlyProfile(p) {
				return fmt.Errorf("profile %q does not target a workspace. Run databricks auth login --host <workspace-url> and retry with that profile", p.Name)
			}

			workspaceID, err := resolveConfigureDockerWorkspaceID(ctx, p, deps)
			if err != nil {
				return err
			}
			if err := ensureConfigureDockerUniqueProfile(ctx, deps.profiler, p, workspaceID); err != nil {
				return err
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

			executable, err := deps.executable()
			if err != nil {
				return fmt.Errorf("locate databricks executable: %w", err)
			}
			installDir, err := configureDockerShimInstallDir(ctx)
			if err != nil {
				return err
			}
			shim, err := deps.installShim(ctx, executable, installDir)
			if err != nil {
				return fmt.Errorf("install Docker credential helper: %w", err)
			}
			dockerConfigPath, err := configureDockerConfigPath(ctx)
			if err != nil {
				return err
			}
			if err := deps.setCredentialHelper(dockerConfigPath, registryHost); err != nil {
				return fmt.Errorf("update Docker config %s: %w", dockerConfigPath, err)
			}

			cmdio.LogString(ctx, fmt.Sprintf("Configured Docker credential helper for %s", registryHost))
			cmdio.LogString(ctx, fmt.Sprintf("Updated Docker config: %s", dockerConfigPath))
			cmdio.LogString(ctx, fmt.Sprintf("Installed Docker credential helper: %s", shim.Path))
			if !shim.OnPath {
				cmdio.LogString(ctx, fmt.Sprintf("Warning: ensure %s is on PATH before any other docker-credential-databricks helper so Docker can find it", installDir))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&region, "region", "", "Cloud region for the Databricks Artifact Registry host")

	return cmd
}

func rejectConfigureDockerAuthFlags(cmd *cobra.Command) error {
	for _, name := range []string{"host", "account-id", "workspace-id"} {
		flag := cmd.Flag(name)
		if flag != nil && flag.Changed {
			return fmt.Errorf("--%s is not supported for configure-docker. Select the workspace with [PROFILE] or --profile instead", name)
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
	result, selected, err := pickAuthProfile(ctx, profiles, profilePickerOptions{
		Label:   "Select a workspace profile",
		Default: currentDefault,
	})
	if err != nil {
		return "", err
	}
	if result != profilePickerProfile {
		return "", errors.New("no profile selected")
	}
	return selected, nil
}

func loadConfigureDockerProfile(ctx context.Context, profileName string, profiler profile.Profiler) (profile.Profile, error) {
	profiles, err := profiler.LoadProfiles(ctx, profile.WithName(profileName))
	if err != nil {
		return profile.Profile{}, err
	}
	if len(profiles) == 0 {
		return profile.Profile{}, fmt.Errorf("profile %q not found", profileName)
	}
	return profiles[0], nil
}

func validateConfigureDockerProfile(p profile.Profile) error {
	if p.HasClientCredentials {
		return fmt.Errorf("profile %q uses client credentials. databricks auth configure-docker requires a profile created by databricks auth login", p.Name)
	}
	if p.AuthType != authTypeDatabricksCLI {
		return fmt.Errorf("profile %q uses auth_type %q. databricks auth configure-docker requires a profile created by databricks auth login", p.Name, p.AuthType)
	}
	return nil
}

func isConfigureDockerAccountOnlyProfile(p profile.Profile) bool {
	if p.Host == "" {
		return true
	}
	cfg := &config.Config{Host: p.Host, AccountID: p.AccountID, WorkspaceID: p.WorkspaceID}
	if authlib.IsClassicAccountHost(cfg.CanonicalHostName()) {
		return true
	}
	return p.AccountID != "" && (p.WorkspaceID == "" || p.WorkspaceID == authlib.WorkspaceIDNone)
}

func resolveConfigureDockerWorkspaceID(ctx context.Context, p profile.Profile, deps configureDockerDeps) (string, error) {
	if p.WorkspaceID != "" && p.WorkspaceID != authlib.WorkspaceIDNone {
		return p.WorkspaceID, nil
	}

	cfg := &databricks.Config{
		Profile:     p.Name,
		Host:        p.Host,
		AccountID:   p.AccountID,
		WorkspaceID: p.WorkspaceID,
		ConfigFile:  env.Get(ctx, "DATABRICKS_CONFIG_FILE"),
	}
	w, err := deps.newWorkspaceClient(cfg)
	if err != nil {
		return "", fmt.Errorf("load workspace profile %q: %w. Run databricks auth login --host <workspace-url> and retry with that profile", p.Name, err)
	}
	workspaceID, err := deps.resolveWorkspaceID(ctx, w)
	if err != nil {
		return "", fmt.Errorf("resolve workspace ID for profile %q: %w. Run databricks auth login --host <workspace-url> and retry with that profile", p.Name, err)
	}
	return workspaceID, nil
}

func ensureConfigureDockerUniqueProfile(ctx context.Context, profiler profile.Profiler, p profile.Profile, workspaceID string) error {
	matches, err := profiler.LoadProfiles(ctx, func(candidate profile.Profile) bool {
		return candidate.WorkspaceID == workspaceID
	})
	if err != nil {
		return err
	}

	names := matches.Names()
	if p.WorkspaceID == "" || p.WorkspaceID == authlib.WorkspaceIDNone {
		names = append(names, p.Name)
	}
	if len(names) <= 1 {
		return nil
	}

	return fmt.Errorf("multiple Databricks profiles match workspace ID %s: %s. Make the profile selection unambiguous before using Docker credential helper", workspaceID, strings.Join(names, " and "))
}

func persistConfigureDockerWorkspaceID(ctx context.Context, p profile.Profile, workspaceID string) error {
	return databrickscfg.SaveToProfile(ctx, &config.Config{
		ConfigFile:    env.Get(ctx, "DATABRICKS_CONFIG_FILE"),
		Profile:       p.Name,
		Host:          p.Host,
		AccountID:     p.AccountID,
		WorkspaceID:   workspaceID,
		AuthType:      p.AuthType,
		ClusterID:     p.ClusterID,
		Scopes:        splitScopes(p.Scopes),
		AzureTenantID: "",
	})
}

func configureDockerConfigPath(ctx context.Context) (string, error) {
	if dockerConfig := env.Get(ctx, "DOCKER_CONFIG"); dockerConfig != "" {
		return filepath.Join(dockerConfig, "config.json"), nil
	}
	home, err := env.UserHomeDir(ctx)
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".docker", "config.json"), nil
}

func configureDockerShimInstallDir(ctx context.Context) (string, error) {
	home, err := env.UserHomeDir(ctx)
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".databricks", "bin"), nil
}

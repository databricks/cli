package docker

import (
	"errors"
	"fmt"
	"strings"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/dockercredentials"
	"github.com/spf13/cobra"
)

type dockerHostDeps struct {
	dockerProfileDeps
	credentialHelperConfigured func(string, string) (bool, error)
}

type dockerHostOutput struct {
	Host       string `json:"host"`
	Configured bool   `json:"configured"`
}

func defaultDockerHostDeps() dockerHostDeps {
	return dockerHostDeps{
		dockerProfileDeps:          defaultDockerProfileDeps(),
		credentialHelperConfigured: dockercredentials.CredentialHelperConfigured,
	}
}

func newDockerHostCommand() *cobra.Command {
	return newDockerHostCommandWithDeps(defaultDockerHostDeps())
}

func newDockerHostCommandWithDeps(deps dockerHostDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "host",
		Short: "(Experimental) Show the Databricks Artifact Registry host for a profile",
		Long: `(Experimental) Show the Databricks Artifact Registry host for a profile.

The --profile flag is required.`,
		Example: "  databricks auth docker host --profile DEFAULT",
		Args:    cobra.NoArgs,
		Annotations: map[string]string{
			"template": "Registry host: {{.Host}}\nCredential helper configured: {{bool .Configured}}\n",
		},
	}
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		ctx := cmd.Context()
		if err := errorOnUnsupportedDockerHostFlags(cmd); err != nil {
			return err
		}

		profileName := strings.TrimSpace(cmd.Flag("profile").Value.String())
		if profileName == "" {
			return errors.New("--profile is required for auth docker host")
		}

		p, err := loadAndValidateDockerProfile(ctx, profileName, deps.profiler)
		if err != nil {
			return err
		}
		if err := deps.validateWorkspaceHost(p.Host); err != nil {
			return err
		}

		executable, err := deps.executable()
		if err != nil {
			return fmt.Errorf("locate databricks executable: %w", err)
		}
		w, err := newDockerWorkspaceClient(ctx, p, executable, deps.dockerProfileDeps)
		if err != nil {
			return err
		}
		workspaceID, err := resolveDockerWorkspaceID(ctx, p, w, deps.dockerProfileDeps)
		if err != nil {
			return err
		}

		w.Config.WorkspaceID = workspaceID
		region, err := deps.resolveWorkspaceRegion(ctx, w)
		if err != nil {
			return fmt.Errorf("resolve workspace region for profile %q: %w", p.Name, err)
		}
		region = strings.TrimSpace(region)
		if region == "" {
			return fmt.Errorf("resolve workspace region for profile %q: metastore summary did not include a region", p.Name)
		}

		registryHost, err := deps.registryHost(workspaceID, region, p.Host)
		if err != nil {
			return err
		}
		dockerConfigPath, err := dockerConfigPath(ctx)
		if err != nil {
			return err
		}
		configured, err := deps.credentialHelperConfigured(dockerConfigPath, registryHost)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, dockerHostOutput{
			Host:       registryHost,
			Configured: configured,
		})
	}
	return cmd
}

func errorOnUnsupportedDockerHostFlags(cmd *cobra.Command) error {
	for _, name := range []string{"host", "account-id", "workspace-id"} {
		if cmd.Flag(name).Changed {
			return fmt.Errorf("--%s is not supported for auth docker host. Select the workspace with --profile instead", name)
		}
	}
	return nil
}

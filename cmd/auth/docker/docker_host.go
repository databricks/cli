package docker

import (
	"errors"
	"strings"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/dockercredentials"
	"github.com/databricks/cli/libs/env"
	"github.com/spf13/cobra"
)

func newDockerHostCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "host",
		Short: "(Experimental) Show the registry host and credential helper status for a profile",
		Long: `(Experimental) Show the Databricks Artifact Registry host and Docker credential helper status for a profile.

The output names the profile the credential helper uses for the registry. The
helper counts as configured only if Docker's config.json selects it for the
registry and it uses this profile. The --profile flag is required.`,
		Example: "  databricks auth docker host --profile DEFAULT",
		Args:    cobra.NoArgs,
		Annotations: map[string]string{
			"template": "Registry host: {{.Host}}\nCredential helper configured: {{bool .Configured}}\n{{if .Profile}}Credential helper profile: {{.Profile}}\n{{end}}",
		},
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			return errorOnUnsupportedDockerFlags(cmd, "--profile")
		},
	}
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		ctx := cmd.Context()
		target, err := dockerHostTarget(cmd)
		if err != nil {
			return err
		}
		recorded, err := databrickscfg.GetDockerProfile(ctx, target.registryHost, env.Get(ctx, "DATABRICKS_CONFIG_FILE"))
		if err != nil {
			return err
		}
		registry := dockercredentials.Registry{WorkspaceID: target.workspaceID, Host: target.registryHost}
		status, err := dockercredentials.Inspect(ctx, profile.DefaultProfiler, registry, recorded, target.profile.Name)
		if err != nil {
			return err
		}
		return cmdio.Render(ctx, status)
	}
	return cmd
}

func dockerHostTarget(cmd *cobra.Command) (*dockerTarget, error) {
	name := strings.TrimSpace(cmd.Flag("profile").Value.String())
	if name == "" {
		return nil, errors.New("--profile is required for auth docker host")
	}
	w, err := loadDockerWorkspace(cmd.Context(), name, nil)
	if err != nil {
		return nil, err
	}
	return w.target(cmd.Context(), nil)
}

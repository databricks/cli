package docker

import (
	"errors"
	"strings"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/dockercredentials"
	"github.com/spf13/cobra"
)

func newDockerHostCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "host",
		Short: "(Experimental) Show the registry host and credential helper status for a profile",
		Long: `(Experimental) Show the Databricks Artifact Registry host and Docker credential helper status for a profile.

The --profile flag is required.`,
		Example: "  databricks auth docker host --profile DEFAULT",
		Args:    cobra.NoArgs,
		Annotations: map[string]string{
			"template": "Registry host: {{.Host}}\nCredential helper configured: {{bool .Configured}}\n",
		},
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			return errorOnUnsupportedDockerFlags(cmd, "--profile")
		},
	}
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		target, err := dockerHostTarget(cmd)
		if err != nil {
			return err
		}
		status, err := dockercredentials.Inspect(cmd.Context(), target.registryHost)
		if err != nil {
			return err
		}
		return cmdio.Render(cmd.Context(), status)
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

package agent

import (
	"fmt"

	libagent "github.com/databricks/cli/libs/agent"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

func newConsentCommand() *cobra.Command {
	var operation string
	var reason string

	cmd := &cobra.Command{
		Use:   "consent",
		Short: "Capture user consent for a gated operation",
		Long: `Capture explicit user consent before performing a potentially destructive operation.

AI agents are required to obtain user consent before using flags like --force-lock,
--auto-approve, or --force on deploy. This command creates a consent token that
must be passed via the DATABRICKS_CLI_AGENT_CONSENT environment variable.

The consent token expires after 10 minutes.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			tokenPath, err := libagent.CreateConsentToken(operation, reason)
			if err != nil {
				return err
			}

			cmdio.LogString(cmd.Context(), fmt.Sprintf("%s=%s", libagent.ConsentEnvVar, tokenPath))
			return nil
		},
	}

	cmd.Flags().StringVar(&operation, "operation", "", fmt.Sprintf("The operation to consent to: %v", libagent.ValidOperations))
	cmd.Flags().StringVar(&reason, "reason", "", "Why the user approved this operation (minimum 20 characters)")
	_ = cmd.MarkFlagRequired("operation")
	_ = cmd.MarkFlagRequired("reason")

	return cmd
}

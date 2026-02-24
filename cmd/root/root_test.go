package root

import (
	"bytes"
	"context"
	"testing"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecuteEnrichesAuthErrors(t *testing.T) {
	ctx := context.Background()
	stderr := &bytes.Buffer{}

	cmd := &cobra.Command{
		Use:           "test",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return &apierr.APIError{
				StatusCode: 403,
				ErrorCode:  "PERMISSION_DENIED",
				Message:    "no access",
			}
		},
	}
	cmd.SetErr(stderr)

	// Simulate MustWorkspaceClient setting config in context via PersistentPreRunE.
	cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		cfg := &config.Config{
			Host:     "https://test.cloud.databricks.com",
			Profile:  "test-profile",
			AuthType: "pat",
		}
		ctx := cmdctx.SetConfigUsed(cmd.Context(), cfg)
		cmd.SetContext(ctx)
		return nil
	}

	err := Execute(ctx, cmd)
	require.Error(t, err)

	output := stderr.String()
	assert.Contains(t, output, "no access")
	assert.Contains(t, output, "Next steps:")
}

func TestExecuteNoEnrichmentWithoutConfigUsed(t *testing.T) {
	ctx := context.Background()
	stderr := &bytes.Buffer{}

	cmd := &cobra.Command{
		Use:           "test",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return &apierr.APIError{
				StatusCode: 403,
				ErrorCode:  "PERMISSION_DENIED",
				Message:    "no access",
			}
		},
	}
	cmd.SetErr(stderr)

	err := Execute(ctx, cmd)
	require.Error(t, err)

	output := stderr.String()
	assert.Contains(t, output, "no access")
	assert.NotContains(t, output, "Profile:")
	assert.NotContains(t, output, "Next steps:")
}

func TestExecuteErrAlreadyPrintedNotEnriched(t *testing.T) {
	ctx := context.Background()
	stderr := &bytes.Buffer{}

	cmd := &cobra.Command{
		Use:           "test",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return ErrAlreadyPrinted
		},
	}
	cmd.SetErr(stderr)

	err := Execute(ctx, cmd)
	require.Error(t, err)
	assert.Empty(t, stderr.String())
}

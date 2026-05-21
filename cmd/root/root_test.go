package root

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"github.com/charmbracelet/huh"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecuteEnrichesAuthErrors(t *testing.T) {
	ctx := t.Context()
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
	ctx := t.Context()
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

func TestIsInterrupted(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"random error", errors.New("boom"), false},
		{"cmdio interrupt", cmdio.ErrInterrupted, true},
		{"huh aborted", huh.ErrUserAborted, true},
		{"wrapped cmdio interrupt", fmt.Errorf("prompt: %w", cmdio.ErrInterrupted), true},
		{"wrapped huh aborted", fmt.Errorf("form: %w", huh.ErrUserAborted), true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, IsInterrupted(tc.err))
		})
	}
}

func TestExecuteInterruptPrintsCancelled(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{"cmdio interrupt", cmdio.ErrInterrupted},
		{"huh aborted", huh.ErrUserAborted},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stderr := &bytes.Buffer{}
			cmd := &cobra.Command{
				Use:           "test",
				SilenceUsage:  true,
				SilenceErrors: true,
				RunE:          func(cmd *cobra.Command, args []string) error { return tc.err },
			}
			cmd.SetErr(stderr)

			err := Execute(t.Context(), cmd)
			require.Error(t, err)
			assert.True(t, IsInterrupted(err))

			output := stderr.String()
			assert.Equal(t, "cancelled\n", output)
			assert.NotContains(t, output, "Error:")
		})
	}
}

func TestExecuteErrAlreadyPrintedNotEnriched(t *testing.T) {
	ctx := t.Context()
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

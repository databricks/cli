package root

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/databricks-sdk-go"
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

func TestExecuteAppendsAccountHostHint(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(ctx context.Context) context.Context
		wantHint bool
	}{
		{
			name: "workspace client on account console host",
			setup: func(ctx context.Context) context.Context {
				cfg := &config.Config{Host: "https://accounts.test", Profile: "acc"}
				ctx = cmdctx.SetConfigUsed(ctx, cfg)
				return cmdctx.SetWorkspaceClient(ctx, &databricks.WorkspaceClient{Config: cfg})
			},
			wantHint: true,
		},
		{
			name: "workspace client on workspace host",
			setup: func(ctx context.Context) context.Context {
				cfg := &config.Config{Host: "https://adb-123.test", Profile: "ws"}
				ctx = cmdctx.SetConfigUsed(ctx, cfg)
				return cmdctx.SetWorkspaceClient(ctx, &databricks.WorkspaceClient{Config: cfg})
			},
			wantHint: false,
		},
		{
			// `databricks account ...` commands configure an account client,
			// not a workspace client, so the hint must stay silent.
			name: "account client on account console host",
			setup: func(ctx context.Context) context.Context {
				cfg := &config.Config{Host: "https://accounts.test", Profile: "acc"}
				ctx = cmdctx.SetConfigUsed(ctx, cfg)
				return cmdctx.SetAccountClient(ctx, &databricks.AccountClient{Config: cfg})
			},
			wantHint: false,
		},
		{
			name: "no client on account console host",
			setup: func(ctx context.Context) context.Context {
				cfg := &config.Config{Host: "https://accounts.test", Profile: "acc"}
				return cmdctx.SetConfigUsed(ctx, cfg)
			},
			wantHint: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stderr := &bytes.Buffer{}
			cmd := &cobra.Command{
				Use:           "test",
				SilenceUsage:  true,
				SilenceErrors: true,
				RunE: func(cmd *cobra.Command, args []string) error {
					// The account console returns unstructured junk for
					// workspace API paths; this is one real example.
					return errors.New("received HTML response instead of JSON")
				},
			}
			cmd.SetErr(stderr)
			cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
				cmd.SetContext(tt.setup(cmd.Context()))
				return nil
			}

			err := Execute(t.Context(), cmd)
			require.Error(t, err)

			output := stderr.String()
			assert.Contains(t, output, "received HTML response instead of JSON")
			if tt.wantHint {
				assert.Contains(t, output, "account console host")
				assert.Contains(t, output, "databricks auth login --host")
			} else {
				assert.NotContains(t, output, "account console host")
			}
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

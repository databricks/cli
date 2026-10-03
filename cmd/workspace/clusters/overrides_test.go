package clusters

import (
	"errors"
	"testing"

	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStartIdempotentOverride(t *testing.T) {
	t.Run("swallows INVALID_STATE from a non terminated cluster", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.RunE = func(cmd *cobra.Command, args []string) error {
			return apierr.ErrInvalidState
		}

		startIdempotentOverride(cmd, &compute.StartCluster{})
		require.NoError(t, cmd.RunE(cmd, nil))
	})

	t.Run("swallows APIError with INVALID_STATE code", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.RunE = func(cmd *cobra.Command, args []string) error {
			return &apierr.APIError{
				ErrorCode: "INVALID_STATE",
				Message:   "Cluster abc is in unexpected state Running.",
			}
		}

		startIdempotentOverride(cmd, &compute.StartCluster{})
		require.NoError(t, cmd.RunE(cmd, nil))
	})

	t.Run("preserves other errors", func(t *testing.T) {
		cmd := &cobra.Command{}
		want := errors.New("boom")
		cmd.RunE = func(cmd *cobra.Command, args []string) error {
			return want
		}

		startIdempotentOverride(cmd, &compute.StartCluster{})
		err := cmd.RunE(cmd, nil)
		require.Error(t, err)
		assert.ErrorIs(t, err, want)
	})

	t.Run("passes success through", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.RunE = func(cmd *cobra.Command, args []string) error {
			return nil
		}

		startIdempotentOverride(cmd, &compute.StartCluster{})
		require.NoError(t, cmd.RunE(cmd, nil))
	})
}

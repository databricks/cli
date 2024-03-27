package bundleflag

import (
	"context"
	"testing"

	benv "github.com/databricks/cli/bundle/env"
	"github.com/databricks/cli/libs/env"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func prepareFlagTest() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	Init(cmd)
	return cmd
}

func TestTargetValue(t *testing.T) {
	cmd := prepareFlagTest()
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		target := Target(cmd)
		if target != "test" {
			t.Errorf("expected 'test', got %q", target)
		}
		return nil
	}

	t.Run("target flag", func(t *testing.T) {
		cmd.SetArgs([]string{"--target", "test"})
		err := cmd.Execute()
		require.NoError(t, err)
	})

	t.Run("environment flag", func(t *testing.T) {
		cmd.SetArgs([]string{"--environment", "test"})
		err := cmd.Execute()
		require.NoError(t, err)
	})

	t.Run("environment variable", func(t *testing.T) {
		ctx := env.Set(cmd.Context(), benv.TargetVariable, "test")
		cmd.SetContext(ctx)
		cmd.SetArgs([]string{})
		err := cmd.Execute()
		require.NoError(t, err)
	})
}

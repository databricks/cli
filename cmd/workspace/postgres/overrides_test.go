package postgres

import (
	"testing"

	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func cmdWithJSON(t *testing.T, raw string) *cobra.Command {
	t.Helper()
	cmd := &cobra.Command{}
	var jf flags.JsonFlag
	cmd.Flags().Var(&jf, "json", "JSON body")
	if raw != "" {
		require.NoError(t, jf.Set(raw))
	}
	return cmd
}

func TestRejectWrappedRoleJSON(t *testing.T) {
	t.Run("rejects wrapped {role: ...}", func(t *testing.T) {
		cmd := cmdWithJSON(t, `{"role":{"spec":{"identity_type":"SERVICE_PRINCIPAL"}}}`)
		err := rejectWrappedRoleJSON(cmd)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "should NOT be wrapped")
		assert.Contains(t, err.Error(), `databricks postgres create-role`)
	})

	t.Run("passes when body has spec at top level", func(t *testing.T) {
		cmd := cmdWithJSON(t, `{"spec":{"identity_type":"SERVICE_PRINCIPAL"}}`)
		assert.NoError(t, rejectWrappedRoleJSON(cmd))
	})

	t.Run("passes when --json was not provided", func(t *testing.T) {
		cmd := cmdWithJSON(t, "")
		assert.NoError(t, rejectWrappedRoleJSON(cmd))
	})

	t.Run("passes through non-object JSON to the generated diagnostics path", func(t *testing.T) {
		cmd := cmdWithJSON(t, `"not-an-object"`)
		assert.NoError(t, rejectWrappedRoleJSON(cmd))
	})

	t.Run("passes when --json flag is absent on the command", func(t *testing.T) {
		assert.NoError(t, rejectWrappedRoleJSON(&cobra.Command{}))
	})
}

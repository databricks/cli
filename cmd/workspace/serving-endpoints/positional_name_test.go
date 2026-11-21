package serving_endpoints

import (
	"testing"

	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func cmdWithJSON(t *testing.T, raw string) (*cobra.Command, *flags.JsonFlag) {
	t.Helper()
	cmd := &cobra.Command{Use: "update-config NAME"}
	var jf flags.JsonFlag
	cmd.Flags().Var(&jf, "json", "")
	require.NoError(t, cmd.Flags().Set("json", raw))
	return cmd, &jf
}

func TestTakeJSONName(t *testing.T) {
	cmd, jf := cmdWithJSON(t, `{"name": "my-endpoint", "served_entities": [{"entity_version": 12345678901234567890}]}`)
	name, err := takeJSONName(cmd)
	require.NoError(t, err)
	assert.Equal(t, "my-endpoint", name)
	assert.JSONEq(t, `{"served_entities": [{"entity_version": 12345678901234567890}]}`, string(jf.Raw()))
}

func TestTakeJSONNameMissing(t *testing.T) {
	for _, raw := range []string{`{"served_entities": []}`, `{"name": ""}`, `{"name": 1}`, `[1]`} {
		t.Run(raw, func(t *testing.T) {
			cmd, jf := cmdWithJSON(t, raw)
			_, err := takeJSONName(cmd)
			assert.ErrorContains(t, err, "accepts 1 arg(s), received 0")
			assert.Equal(t, raw, string(jf.Raw()))
		})
	}
}

func TestJSONName(t *testing.T) {
	cmd, _ := cmdWithJSON(t, `{"name": "a"}`)
	assert.Equal(t, "a", jsonName(cmd))
	cmd, _ = cmdWithJSON(t, `{"config": {}}`)
	assert.Empty(t, jsonName(cmd))
}

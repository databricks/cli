package aircmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderEnvelope(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	require.NoError(t, renderEnvelope(ctx, getData{RunID: "1", Status: "RUNNING"}))
}

// withOutput registers the --output flag on cmd and sets it, mirroring how the
// root command wires output mode in production. Subcommand unit tests need it
// because they invoke RunE without going through the root command.
func withOutput(cmd *cobra.Command, output flags.Output) *cobra.Command {
	cmd.Flags().Var(&output, "output", "")
	return cmd
}

func TestRenderErrorJSON(t *testing.T) {
	var buf bytes.Buffer
	ctx := cmdio.InContext(t.Context(), cmdio.NewIO(t.Context(), flags.OutputJSON, nil, &buf, &buf, "", ""))
	cmd := withOutput(&cobra.Command{}, flags.OutputJSON)

	err := renderError(ctx, cmd, "NOT_FOUND", "NOT_FOUND", false, errors.New("run 1 not found"))
	// JSON mode prints the envelope, so Cobra must stay silent but still exit non-zero.
	require.ErrorIs(t, err, root.ErrAlreadyPrinted)

	// The envelope must match the Python air CLI's print_json_error shape exactly.
	var got errorEnvelope
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	assert.Equal(t, 1, got.V)
	assert.NotEmpty(t, got.TS)
	assert.Equal(t, jsonError{Code: "NOT_FOUND", Kind: "NOT_FOUND", Message: "run 1 not found"}, got.Error)
}

func TestRenderErrorText(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	cmd := withOutput(&cobra.Command{}, flags.OutputText)
	want := errors.New("run 1 not found")
	require.Equal(t, want, renderError(ctx, cmd, "NOT_FOUND", "NOT_FOUND", false, want))
}

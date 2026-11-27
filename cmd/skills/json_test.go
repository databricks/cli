package skills

import (
	"bytes"
	"testing"
	"time"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	libskills "github.com/databricks/cli/libs/skills"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// outputTestCmd returns a command whose output is in the given format, and the
// buffers that capture stdout and stderr.
func outputTestCmd(t *testing.T, output flags.Output) (*cobra.Command, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	ctx := cmdio.InContext(t.Context(), cmdio.NewIO(t.Context(), output, nil, &stdout, &stderr, "", ""))
	cmd := &cobra.Command{}
	cmd.SetContext(ctx)
	out := output
	cmd.Flags().Var(&out, "output", "")
	return cmd, &stdout, &stderr
}

func TestRunListJSONStaleCatalogSaysStale(t *testing.T) {
	catalog := &libskills.Catalog{
		Label:     "cached",
		Version:   "1.2.3",
		FetchedAt: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC),
		Stale:     true,
		Skills:    []libskills.Skill{{Name: "alpha"}},
	}

	cmd, stdout, _ := outputTestCmd(t, flags.OutputJSON)
	require.NoError(t, runList(cmd, listFakeSource{catalog}, false))
	assert.Contains(t, stdout.String(), `"stale": true`)
	assert.Contains(t, stdout.String(), `"fetched_at": "2026-11-01T00:00:00Z"`)
}

// A stale catalog gets the provenance line in both output modes, as skills list
// prints it. The JSON says "stale": true and leaves out fetched_at, because a
// bundled copy has no fetch time.
func TestRunSearchStaleCatalogSaysStale(t *testing.T) {
	catalog := &libskills.Catalog{Label: "bundled", Version: "1.2.3", Stale: true, Skills: searchCatalog}

	cmd, stdout, stderr := outputTestCmd(t, flags.OutputJSON)
	require.NoError(t, runSearch(cmd, listFakeSource{catalog}, []string{"jobs"}, false))
	assert.Contains(t, stdout.String(), `"stale": true`)
	assert.NotContains(t, stdout.String(), "fetched_at")
	assert.Contains(t, stderr.String(), "could not be refreshed")

	cmd, _, stderr = outputTestCmd(t, flags.OutputText)
	require.NoError(t, runSearch(cmd, listFakeSource{catalog}, []string{"jobs"}, false))
	assert.Contains(t, stderr.String(), "could not be refreshed")
}

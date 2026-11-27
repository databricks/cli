package skills

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	libskills "github.com/databricks/cli/libs/skills"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// jsonTestCmd returns a command whose output is JSON, and the buffer that
// captures stdout.
func jsonTestCmd(t *testing.T) (*cobra.Command, *bytes.Buffer) {
	t.Helper()
	var stdout bytes.Buffer
	ctx := cmdio.InContext(t.Context(), cmdio.NewIO(t.Context(), flags.OutputJSON, nil, &stdout, io.Discard, "", ""))
	cmd := &cobra.Command{}
	cmd.SetContext(ctx)
	out := flags.OutputJSON
	cmd.Flags().Var(&out, "output", "")
	return cmd, &stdout
}

func TestRunListJSONMarksExperimentalAndCarriesRules(t *testing.T) {
	catalog := &libskills.Catalog{
		Label:   "bundled",
		Version: "1.2.3",
		Rules:   "Pick the skill.\n",
		Skills: []libskills.Skill{
			{Name: "alpha", Description: "Alpha."},
			{Name: "preview", Description: "Preview.", Experimental: true},
		},
	}

	cmd, stdout := jsonTestCmd(t)
	require.NoError(t, runList(cmd, listFakeSource{catalog}, true))

	var got struct {
		Rules  string `json:"rules"`
		Stale  bool   `json:"stale"`
		Skills []struct {
			Name         string `json:"name"`
			Experimental bool   `json:"experimental"`
		} `json:"skills"`
	}
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &got))
	require.Len(t, got.Skills, 2)
	assert.Equal(t, "alpha", got.Skills[0].Name)
	assert.False(t, got.Skills[0].Experimental)
	assert.Equal(t, "preview", got.Skills[1].Name)
	assert.True(t, got.Skills[1].Experimental)
	assert.False(t, got.Stale)

	// The rules are the text the text output prints above the list.
	assert.Equal(t, "Pick the skill.", got.Rules)
	assert.True(t, strings.HasPrefix(renderText(catalog.Rules, catalog.Skills), got.Rules+"\n\n"))
}

func TestRunListJSONStaleCatalogSaysStale(t *testing.T) {
	catalog := &libskills.Catalog{
		Label:     "cached",
		Version:   "1.2.3",
		FetchedAt: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC),
		Stale:     true,
		Skills:    []libskills.Skill{{Name: "alpha"}},
	}

	cmd, stdout := jsonTestCmd(t)
	require.NoError(t, runList(cmd, listFakeSource{catalog}, false))
	assert.Contains(t, stdout.String(), `"stale": true`)
	assert.Contains(t, stdout.String(), `"fetched_at": "2026-11-01T00:00:00Z"`)
}

// skills search builds its JSON with the same constructor, so a stale catalog
// says so there too.
func TestSearchJSONStaleCatalogSaysStale(t *testing.T) {
	catalog := &libskills.Catalog{Label: "bundled", Version: "1.2.3", Stale: true, Skills: searchCatalog}

	out, err := json.Marshal(newCatalogJSON(catalog, matchSkills(searchCatalog, []string{"jobs"})))
	require.NoError(t, err)
	assert.Contains(t, string(out), `"stale":true`)
	assert.NotContains(t, string(out), "fetched_at")
}

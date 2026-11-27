package skills

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	libskills "github.com/databricks/cli/libs/skills"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type listFakeSource struct {
	catalog *libskills.Catalog
}

func (f listFakeSource) Catalog(context.Context) (*libskills.Catalog, error) {
	return f.catalog, nil
}

func (f listFakeSource) ReadFile(context.Context, string, string) ([]byte, error) {
	return nil, libskills.ErrUnknownFile
}

// listTestCmd returns a command with the root's output flag, whose context
// captures stderr, and the buffer that captures stdout.
func listTestCmd(t *testing.T, output flags.Output) (*cobra.Command, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	cmd := &cobra.Command{}
	cmd.SetContext(ctx)
	out := output
	cmd.Flags().Var(&out, "output", "")
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	return cmd, &stdout, stderr
}

// longSkills returns n skills whose descriptions are far longer than the budget
// allows for each one.
func longSkills(n int) []libskills.Skill {
	var list []libskills.Skill
	for i := range n {
		desc := strings.TrimSpace(strings.Repeat(fmt.Sprintf("word%d describes the skill in detail. ", i), 12))
		list = append(list, libskills.Skill{Name: fmt.Sprintf("skill-%02d", i), Description: desc})
	}
	return list
}

func TestRenderTextFortySkillsFitsBudget(t *testing.T) {
	list := visibleSkills(longSkills(40), false)
	require.Len(t, list, 40)

	out := renderText("Pick the skill whose description matches the task.", list)

	assert.LessOrEqual(t, len(out), textBudget)
	for _, s := range list {
		assert.Contains(t, out, s.Name+": "+strings.Fields(s.Description)[0])
	}
	assert.True(t, strings.HasPrefix(out, "Pick the skill whose description matches the task.\n\n"))
	assert.True(t, strings.HasSuffix(out, readHint+"\n"))
	assert.NotContains(t, out, "databricks skills search")
}

func TestRenderTextKeepsFullDescriptionsWhenTheyFit(t *testing.T) {
	list := []libskills.Skill{
		{Name: "alpha", Description: "Alpha does one thing well."},
		{Name: "beta", Description: "Beta does another."},
	}
	out := renderText("", list)
	assert.Equal(t, "alpha: Alpha does one thing well.\nbeta: Beta does another.\n"+readHint+"\n", out)
}

func TestRenderTextKeepsOneLinePerSkill(t *testing.T) {
	list := []libskills.Skill{{Name: "alpha", Description: "Line one.\nLine two.\tTabbed  text."}}
	assert.Equal(t, "alpha: Line one. Line two. Tabbed text.\n"+readHint+"\n", renderText("", list))
}

func TestRenderTextNamesOnlyWhenNamesExceedBudget(t *testing.T) {
	var list []libskills.Skill
	for i := range 400 {
		list = append(list, libskills.Skill{
			Name:        fmt.Sprintf("a-rather-long-skill-name-%03d", i),
			Description: "Short.",
		})
	}
	out := renderText("", visibleSkills(list, false))

	assert.Contains(t, out, "a-rather-long-skill-name-000\n")
	assert.Contains(t, out, "a-rather-long-skill-name-399\n")
	assert.NotContains(t, out, "Short.")
	assert.Contains(t, out, "databricks skills search <words>")
	assert.Contains(t, out, readHint)
}

func TestRenderTextNeverCutsDescriptionToNothing(t *testing.T) {
	var list []libskills.Skill
	for i := range 130 {
		list = append(list, libskills.Skill{
			Name:        fmt.Sprintf("skill-%014d", i),
			Description: "Deploys the workspace resources, then reports every step taken and any error found.",
		})
	}
	out := renderText("", list)

	assert.LessOrEqual(t, len(out), textBudget)
	assert.NotContains(t, out, ": ...\n")
	for _, s := range list {
		assert.Contains(t, out, s.Name+"\n")
	}
	assert.Contains(t, out, "databricks skills search <words>")
}

func TestRenderTextOneUnbreakableDescriptionDropsOnlyItself(t *testing.T) {
	list := visibleSkills(longSkills(40), false)
	list = append(list, libskills.Skill{Name: "blob", Description: strings.Repeat("x", 300)})

	out := renderText("", list)

	assert.LessOrEqual(t, len(out), textBudget)
	assert.Contains(t, out, "\nblob\n")
	assert.Contains(t, out, "skill-39: word39 ")
	assert.Contains(t, out, "databricks skills search <words>")
}

// The search hint goes away once the last dropped description gets its first
// word. Widths 35 to 37 are over the budget here, 39 is the widest that fits, and
// a binary search settles on 34, which fits only with the hint.
func TestRenderTextKeepsWidestFittingWidth(t *testing.T) {
	var list []libskills.Skill
	for i := range 84 {
		list = append(list, libskills.Skill{
			Name:        fmt.Sprintf("skill-%02d", i),
			Description: "Deploys the workspace resources, then reports every step taken and any error found.",
		})
	}
	list = append(list, libskills.Skill{
		Name:        "odd",
		Description: strings.Repeat("x", 35) + " more.",
	})

	out := renderText("", list)

	assert.LessOrEqual(t, len(out), textBudget)
	assert.Contains(t, out, "odd: "+strings.Repeat("x", 35)+"...\n")
	assert.NotContains(t, out, "databricks skills search")
}

func TestRenderTextNoSearchHintWithoutDescriptions(t *testing.T) {
	list := []libskills.Skill{{Name: "alpha"}, {Name: "beta"}}
	assert.Equal(t, "alpha\nbeta\n"+readHint+"\n", renderText("", list))
}

func TestRenderTextMarksSkillThatNeedsNewerCLI(t *testing.T) {
	list := []libskills.Skill{
		{Name: "alpha", Description: "Alpha does one thing well."},
		{Name: "future", Description: "Needs a newer CLI.", MinCLIVer: "99.0.0"},
	}
	assert.Equal(t, "alpha: Alpha does one thing well.\nfuture: Needs a newer CLI. (needs CLI 99.0.0 or newer)\n"+readHint+"\n",
		renderText("", list))
}

func TestRenderTextMarksNamesOnlyLineThatNeedsNewerCLI(t *testing.T) {
	list := []libskills.Skill{{Name: "alpha"}, {Name: "future", MinCLIVer: "99.0.0"}}
	assert.Equal(t, "alpha\nfuture (needs CLI 99.0.0 or newer)\n"+readHint+"\n", renderText("", list))
}

func TestRenderTextNoMarkWhenCLIIsNewEnough(t *testing.T) {
	list := []libskills.Skill{{Name: "current", Description: "Current.", MinCLIVer: build.GetInfo().Version}}
	assert.Equal(t, "current: Current.\n"+readHint+"\n", renderText("", list))
}

func TestRunListJSONCarriesMinCLIVersion(t *testing.T) {
	catalog := &libskills.Catalog{
		Label:   "cached",
		Version: "1.2.3",
		Skills: []libskills.Skill{
			{Name: "alpha", Description: "First."},
			{Name: "future", Description: "Second.", MinCLIVer: "99.0.0"},
		},
	}

	cmd := &cobra.Command{}
	output := flags.OutputJSON
	cmd.Flags().Var(&output, "output", "")
	ctx, stdout := cmdio.NewTestContextWithStdout(t.Context())
	cmd.SetContext(ctx)
	require.NoError(t, runList(cmd, listFakeSource{catalog}, false))

	var got struct {
		Skills []map[string]any `json:"skills"`
	}
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &got))
	require.Len(t, got.Skills, 2)
	assert.NotContains(t, got.Skills[0], "min_cli_version")
	assert.Equal(t, "99.0.0", got.Skills[1]["min_cli_version"])
}

func TestRenderTextNoSkills(t *testing.T) {
	assert.Equal(t, "No skills found.\n", renderText("", nil))
}

func TestCutWords(t *testing.T) {
	tests := []struct {
		desc  string
		width int
		want  string
	}{
		{"short text", 20, "short text"},
		{"Hello world again", 12, "Hello..."},
		{"Hello world again", 11, "Hello..."},
		{"Hello world again", 9, "Hello..."},
		{"Hello world again", 8, "Hello..."},
		{"Hello world again", 7, ""},
		{"Unbreakable", 6, ""},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, cutWords(tt.desc, tt.width), "cutWords(%q, %d)", tt.desc, tt.width)
	}
}

func TestVisibleSkillsHidesExperimentalKeepsOrder(t *testing.T) {
	all := []libskills.Skill{
		{Name: "alpha"},
		{Name: "preview", Experimental: true},
		{Name: "zeta"},
	}

	stable := visibleSkills(all, false)
	assert.Equal(t, []string{"alpha", "zeta"}, names(stable))

	withExperimental := visibleSkills(all, true)
	assert.Equal(t, []string{"alpha", "preview", "zeta"}, names(withExperimental))
}

func TestRunListHintsAtExperimentalWhenAllHidden(t *testing.T) {
	catalog := &libskills.Catalog{Skills: []libskills.Skill{{Name: "preview", Experimental: true}}}

	cmd, stdout, _ := listTestCmd(t, flags.OutputText)
	require.NoError(t, runList(cmd, listFakeSource{catalog}, false))
	assert.Equal(t, "No skills found.\n"+experimentalHint+"\n", stdout.String())
}

func TestRunListStaleCatalogPrintsProvenanceOnStderr(t *testing.T) {
	catalog := &libskills.Catalog{
		Label:     "cached",
		Version:   "1.2.3",
		FetchedAt: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC),
		Stale:     true,
		Skills:    []libskills.Skill{{Name: "alpha", Description: "First."}},
	}

	cmd, stdout, stderr := listTestCmd(t, flags.OutputText)
	require.NoError(t, runList(cmd, listFakeSource{catalog}, false))
	assert.Contains(t, stderr.String(), "Skills come from cached (release 1.2.3, fetched 2026-11-01T00:00:00Z)")
	assert.NotContains(t, stdout.String(), "Skills come from")
}

func TestProvenanceLineOmitsUnknownFetchTime(t *testing.T) {
	catalog := &libskills.Catalog{Label: "bundled", Version: "1.2.3", Stale: true}
	assert.Equal(t, "Skills come from bundled (release 1.2.3). The source could not be refreshed, so the list may be out of date.",
		provenanceLine(catalog))
}

func names(list []libskills.Skill) []string {
	var out []string
	for _, s := range list {
		out = append(out, s.Name)
	}
	return out
}

package skills

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/databricks/cli/libs/flags"
	libskills "github.com/databricks/cli/libs/skills"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeSource struct {
	catalog *libskills.Catalog
}

func (f fakeSource) Catalog(context.Context) (*libskills.Catalog, error) {
	return f.catalog, nil
}

func (f fakeSource) ReadFile(context.Context, string, string) ([]byte, error) {
	return nil, libskills.ErrUnknownFile
}

func listTestCmd(t *testing.T, output flags.Output) (*cobra.Command, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	cmd := &cobra.Command{}
	cmd.SetContext(t.Context())
	out := output
	cmd.Flags().Var(&out, "output", "")
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	return cmd, &stdout, &stderr
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
		assert.Contains(t, out, s.Name+": ")
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
		{"Hello world again", 7, "..."},
		{"Unbreakable", 6, "..."},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, cutWords(tt.desc, tt.width), "cutWords(%q, %d)", tt.desc, tt.width)
	}
}

func TestVisibleSkillsSortsAndHidesExperimental(t *testing.T) {
	all := []libskills.Skill{
		{Name: "zeta"},
		{Name: "preview", Experimental: true},
		{Name: "alpha"},
	}

	stable := visibleSkills(all, false)
	assert.Equal(t, []string{"alpha", "zeta"}, names(stable))

	withExperimental := visibleSkills(all, true)
	assert.Equal(t, []string{"alpha", "preview", "zeta"}, names(withExperimental))
}

func TestRunListTextAndJSON(t *testing.T) {
	catalog := &libskills.Catalog{
		Label:     "test",
		Version:   "1.2.3",
		FetchedAt: time.Now(),
		Skills: []libskills.Skill{
			{Name: "beta", Description: "Second.", Version: "0.1", Files: []string{"SKILL.md"}, Commands: []string{"jobs"}},
			{Name: "alpha", Description: "First.", Version: "0.2", Files: []string{"SKILL.md"}},
		},
	}

	cmd, stdout, stderr := listTestCmd(t, flags.OutputText)
	require.NoError(t, runList(cmd, fakeSource{catalog}, false))
	assert.Equal(t, "alpha: First.\nbeta: Second.\n"+readHint+"\n", stdout.String())
	assert.Empty(t, stderr.String())

	cmd, stdout, _ = listTestCmd(t, flags.OutputJSON)
	require.NoError(t, runList(cmd, fakeSource{catalog}, false))
	var got struct {
		Source         string `json:"source"`
		ReleaseVersion string `json:"release_version"`
		Skills         []struct {
			Name     string   `json:"name"`
			Version  string   `json:"version"`
			Files    []string `json:"files"`
			Commands []string `json:"commands"`
		} `json:"skills"`
	}
	require.NoError(t, json.Unmarshal(stdout.Bytes(), &got))
	assert.Equal(t, "test", got.Source)
	assert.Equal(t, "1.2.3", got.ReleaseVersion)
	require.Len(t, got.Skills, 2)
	assert.Equal(t, "alpha", got.Skills[0].Name)
	assert.Equal(t, "beta", got.Skills[1].Name)
	assert.Equal(t, []string{"jobs"}, got.Skills[1].Commands)
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
	require.NoError(t, runList(cmd, fakeSource{catalog}, false))
	assert.Contains(t, stderr.String(), "Skills come from cached (release 1.2.3, fetched 2026-11-01T00:00:00Z)")
	assert.NotContains(t, stdout.String(), "Skills come from")
}

func names(list []libskills.Skill) []string {
	var out []string
	for _, s := range list {
		out = append(out, s.Name)
	}
	return out
}

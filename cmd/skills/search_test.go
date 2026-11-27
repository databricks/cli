package skills

import (
	"testing"

	"github.com/databricks/cli/libs/skills"
	"github.com/stretchr/testify/assert"
)

func skillNames(list []skills.Skill) []string {
	out := make([]string, 0, len(list))
	for _, s := range list {
		out = append(out, s.Name)
	}
	return out
}

var searchCatalog = []skills.Skill{
	{Name: "jobs-helper", Description: "Run and monitor jobs.", Commands: []string{"jobs"}},
	{Name: "pipeline-guide", Description: "Build and debug data flows.", Commands: []string{"pipelines"}},
	{Name: "warehouse-sql", Description: "Query SQL endpoints.", Commands: []string{"warehouses", "sql"}},
	// Its command group appears in neither the name nor the description, so only
	// a match on Commands finds it.
	{Name: "cluster-ops", Description: "Manage compute.", Commands: []string{"clusters"}},
}

func TestMatchSkills(t *testing.T) {
	tests := []struct {
		name  string
		words []string
		want  []string
	}{
		{name: "one word matches the name", words: []string{"guide"}, want: []string{"pipeline-guide"}},
		{name: "one word matches the description", words: []string{"monitor"}, want: []string{"jobs-helper"}},
		{name: "one word matches a command group", words: []string{"warehouses"}, want: []string{"warehouse-sql"}},
		{name: "several words match several skills", words: []string{"jobs", "sql"}, want: []string{"jobs-helper", "warehouse-sql"}},
		{name: "case is ignored", words: []string{"JOBS", "Pipelines"}, want: []string{"jobs-helper", "pipeline-guide"}},
		{name: "a part of a command group matches", words: []string{"lines"}, want: []string{"pipeline-guide"}},
		{name: "a command group matches when the name and description do not", words: []string{"clusters"}, want: []string{"cluster-ops"}},
		{name: "no match", words: []string{"nothing-here"}, want: []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, skillNames(matchSkills(searchCatalog, tt.words)))
		})
	}
}

func TestMatchSkillsOrdersByMatchedWordsThenName(t *testing.T) {
	catalog := []skills.Skill{
		{Name: "c-skill", Description: "two"},
		{Name: "b-skill", Description: "one"},
		{Name: "a-skill", Description: "one two"},
		{Name: "d-skill", Description: "three"},
	}
	got := skillNames(matchSkills(catalog, []string{"one", "two"}))
	assert.Equal(t, []string{"a-skill", "b-skill", "c-skill"}, got)
}

func TestMatchSkillsCountsAWordOnceForEachSkill(t *testing.T) {
	catalog := []skills.Skill{
		{Name: "pipeline-guide", Description: "pipeline tips", Commands: []string{"pipelines"}},
		{Name: "other", Description: "pipeline"},
	}
	// "pipeline" appears in the name, the description and the command group of
	// pipeline-guide, but it counts once, so both skills have one hit and the name
	// decides: "other" sorts first.
	got := skillNames(matchSkills(catalog, []string{"pipeline"}))
	assert.Equal(t, []string{"other", "pipeline-guide"}, got)
}

func TestMatchSkillsCountsARepeatedWordOnce(t *testing.T) {
	catalog := []skills.Skill{
		{Name: "a-skill", Description: "two"},
		{Name: "b-skill", Description: "one"},
	}
	// "one" repeated (in another case too) is one word, so b-skill has one hit,
	// the same as a-skill, and the name decides.
	got := skillNames(matchSkills(catalog, []string{"one", " ONE ", "two"}))
	assert.Equal(t, []string{"a-skill", "b-skill"}, got)
}

func TestVisibleSkillsHidesExperimentalUnlessAsked(t *testing.T) {
	catalog := []skills.Skill{
		{Name: "stable"},
		{Name: "beta", Experimental: true},
	}
	assert.Equal(t, []string{"stable"}, skillNames(searchableSkills(catalog, false)))
	assert.Equal(t, []string{"stable", "beta"}, skillNames(searchableSkills(catalog, true)))
	assert.Empty(t, skillNames(matchSkills(searchableSkills(catalog, false), []string{"beta"})))
}

func TestCheckWordsRejectsBlankWord(t *testing.T) {
	assert.NoError(t, checkWords([]string{"jobs", "sql"}))
	assert.Error(t, checkWords([]string{"jobs", ""}))
	assert.Error(t, checkWords([]string{"  \t"}))
}

func TestMatchSkillsTrimsWords(t *testing.T) {
	got := skillNames(matchSkills(searchCatalog, []string{" jobs "}))
	assert.Equal(t, []string{"jobs-helper"}, got)
}

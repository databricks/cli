package skills

import (
	"testing"

	"github.com/databricks/cli/libs/skills"
	"github.com/stretchr/testify/assert"
)

func names(list []skills.Skill) []string {
	out := make([]string, 0, len(list))
	for _, s := range list {
		out = append(out, s.Name)
	}
	return out
}

var searchCatalog = []skills.Skill{
	{Name: "jobs-helper", Description: "Run and monitor jobs.", Commands: []string{"jobs"}},
	{Name: "pipeline-guide", Description: "Build and debug data pipelines.", Commands: []string{"pipelines"}},
	{Name: "warehouse-sql", Description: "Query SQL warehouses.", Commands: []string{"warehouses", "sql"}},
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
		{name: "a word matches inside a command group", words: []string{"pipe"}, want: []string{"pipeline-guide"}},
		{name: "no match", words: []string{"nothing-here"}, want: []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, names(matchSkills(searchCatalog, tt.words)))
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
	got := names(matchSkills(catalog, []string{"one", "two"}))
	assert.Equal(t, []string{"a-skill", "b-skill", "c-skill"}, got)
}

func TestMatchSkillsCountsAWordOnceForEachSkill(t *testing.T) {
	catalog := []skills.Skill{
		{Name: "pipeline-guide", Description: "pipeline tips", Commands: []string{"pipelines"}},
		{Name: "other", Description: "pipeline"},
	}
	// "pipeline" appears in both fields of the first skill, but it counts once,
	// so the second skill, which matches the same single word, sorts after it by name.
	got := names(matchSkills(catalog, []string{"pipeline"}))
	assert.Equal(t, []string{"other", "pipeline-guide"}, got)
}

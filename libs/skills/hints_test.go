package skills

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteGroupHint(t *testing.T) {
	const dir = "testdata/release"
	const footer = "\nCoding agents: read the skills for this command group before you use it.\n" +
		"  databricks skills read alpha\n"
	cases := []struct {
		name    string
		preview string
		hints   string
		dir     string
		group   string
		want    string
	}{
		{name: "hints off", preview: "1", hints: "0", dir: dir, group: "jobs"},
		{name: "no source", preview: "1", hints: "1", dir: "", group: "jobs"},
		{name: "unmapped group", preview: "1", hints: "1", dir: dir, group: "fs"},
		{name: "mapped group", preview: "1", hints: "1", dir: dir, group: "jobs", want: footer},
		// Without DATABRICKS_SKILLS_PREVIEW the feature is off, so the footer
		// stays off even with hints on and a source that maps the group.
		{name: "preview off", preview: "", hints: "1", dir: dir, group: "jobs"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := env.Set(t.Context(), "DATABRICKS_SKILLS_DIR", c.dir)
			ctx = env.Set(ctx, "DATABRICKS_SKILLS_PREVIEW", c.preview)
			ctx = env.Set(ctx, "DATABRICKS_SKILLS_HINTS", c.hints)

			var buf bytes.Buffer
			WriteGroupHint(ctx, &buf, c.group)
			assert.Equal(t, c.want, buf.String())
		})
	}
}

func TestWriteGroupHintLeavesOutSkillNameTheInstallerRejects(t *testing.T) {
	dir := t.TempDir()
	const manifest = `{"version": "1.0.0", "skills": {
		"-foo": {"version": "1.0.0", "description": "A skill whose name starts with a hyphen.", "files": ["SKILL.md"], "commands": ["jobs"]},
		"alpha": {"version": "1.0.0", "description": "A valid skill.", "files": ["SKILL.md"], "commands": ["jobs"]}}}`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0o644))

	ctx := env.Set(t.Context(), "DATABRICKS_SKILLS_DIR", dir)
	ctx = env.Set(ctx, "DATABRICKS_SKILLS_PREVIEW", "1")
	ctx = env.Set(ctx, "DATABRICKS_SKILLS_HINTS", "1")

	var buf bytes.Buffer
	WriteGroupHint(ctx, &buf, "jobs")
	require.Equal(t, "\nCoding agents: read the skills for this command group before you use it.\n"+
		"  databricks skills read alpha\n", buf.String())
}

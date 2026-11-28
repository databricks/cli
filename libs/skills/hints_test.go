package skills

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteGroupHint(t *testing.T) {
	const dir = "testdata/release"
	const footer = "\nCoding agents: read the skills for this command group before you use it.\n" +
		"  databricks skills read alpha\n"
	cases := []struct {
		name  string
		hints string
		dir   string
		group string
		want  string
	}{
		{name: "hints off", hints: "0", dir: dir, group: "jobs"},
		{name: "no source", hints: "1", dir: "", group: "jobs"},
		{name: "unmapped group", hints: "1", dir: dir, group: "fs"},
		{name: "mapped group", hints: "1", dir: dir, group: "jobs", want: footer},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := env.Set(t.Context(), "DATABRICKS_SKILLS_DIR", c.dir)
			ctx = env.Set(ctx, "DATABRICKS_AGENT_FEATURES_DISABLED", "")
			ctx = env.Set(ctx, "DATABRICKS_SKILLS_PREVIEW", "1")
			ctx = env.Set(ctx, "DATABRICKS_SKILLS_HINTS", c.hints)

			var buf bytes.Buffer
			WriteGroupHint(ctx, &buf, c.group)
			assert.Equal(t, c.want, buf.String())
		})
	}
}

func TestWriteGroupHintRunsLeadingHyphenSkillAsPrinted(t *testing.T) {
	dir := t.TempDir()
	const manifest = `{"version": "1.0.0", "skills": {"-foo": {"version": "1.0.0", "description": "A skill whose name starts with a hyphen.", "files": ["SKILL.md"], "commands": ["jobs"]}}}`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0o644))

	ctx := env.Set(t.Context(), "DATABRICKS_SKILLS_DIR", dir)
	ctx = env.Set(ctx, "DATABRICKS_SKILLS_PREVIEW", "1")
	ctx = env.Set(ctx, "DATABRICKS_SKILLS_HINTS", "1")

	var buf bytes.Buffer
	WriteGroupHint(ctx, &buf, "jobs")
	assert.Equal(t, "\nCoding agents: read the skills for this command group before you use it.\n"+
		"  databricks skills read -- -foo\n", buf.String())

	// The printed command, minus "databricks skills read", must parse to the name, not to flags.
	printed := strings.Fields(buf.String())
	args := printed[len(printed)-2:]
	fs := pflag.NewFlagSet("read", pflag.ContinueOnError)
	require.NoError(t, fs.Parse(args))
	assert.Equal(t, []string{"-foo"}, fs.Args())
}

package skills

import (
	"bytes"
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
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

package root

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeSkillsCatalog writes a local release that maps the skill alpha to the
// "jobs" group, and returns its folder.
func writeSkillsCatalog(t *testing.T) string {
	dir := t.TempDir()
	manifest := `{"version": "1.0.0", "skills": {"alpha": {"version": "1.0.0", "description": "A skill.", "files": ["SKILL.md"], "commands": ["jobs"], "repo_dir": "skills"}}}`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0o644))
	return dir
}

// TestWriteSkillsHintLogsEvent checks that help logs one hint-shown event of
// the skills help kind for each footer or root line it prints, and nothing
// when the feature or the hints are off.
func TestWriteSkillsHintLogsEvent(t *testing.T) {
	const footer = "\nCoding agents: read the skills for this command group before you use it.\n" +
		"  databricks skills read alpha\n"
	catalog := writeSkillsCatalog(t)
	cases := []struct {
		name    string
		preview string
		hints   string
		dir     string
		path    string
		want    string
		events  int
	}{
		{name: "guard off", preview: "", hints: "1", dir: catalog, path: "", events: 0},
		{name: "hints off", preview: "1", hints: "0", dir: catalog, path: "", events: 0},
		{name: "root line", preview: "1", hints: "1", dir: catalog, path: "", want: "\n" + rootSkillsHint + "\n", events: 1},
		{name: "group footer", preview: "1", hints: "1", dir: catalog, path: "jobs", want: footer, events: 1},
		{name: "unmapped group", preview: "1", hints: "1", dir: catalog, path: "fs", events: 0},
		{name: "no catalog", preview: "1", hints: "1", dir: "", path: "jobs", events: 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := telemetry.WithNewLogger(t.Context())
			ctx = env.Set(ctx, "DATABRICKS_SKILLS_PREVIEW", c.preview)
			ctx = env.Set(ctx, "DATABRICKS_SKILLS_HINTS", c.hints)
			ctx = env.Set(ctx, "DATABRICKS_SKILLS_DIR", c.dir)

			root, out := newHelpTree(ctx)
			target, _, err := root.Find(strings.Fields(c.path))
			require.NoError(t, err)
			WriteSkillsHint(target)

			assert.Equal(t, c.want, out.String())
			assert.Equal(t, c.events, countSkillsHelpHintEvents(telemetry.BufferedEvents(ctx)))
		})
	}
}

// newHelpTree returns a databricks root with the groups "jobs" and "fs", each
// with one subcommand, and the buffer its help is written to. Cobra does not
// pass a context on to subcommands, so each command gets ctx.
func newHelpTree(ctx context.Context) (*cobra.Command, *bytes.Buffer) {
	out := &bytes.Buffer{}
	root := &cobra.Command{Use: "databricks"}
	root.SetContext(ctx)
	root.SetOut(out)
	for _, name := range []string{"jobs", "fs"} {
		group := &cobra.Command{Use: name}
		group.SetContext(ctx)
		group.AddCommand(&cobra.Command{Use: "list"})
		root.AddCommand(group)
	}
	return root, out
}

func countSkillsHelpHintEvents(events []protos.DatabricksCliLog) int {
	n := 0
	for _, e := range events {
		if e.AitoolsHintEvent != nil && e.AitoolsHintEvent.HintKind == protos.AitoolsHintKindSkillsHelpHint {
			n++
		}
	}
	return n
}

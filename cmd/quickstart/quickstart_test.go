package quickstart

import (
	"bytes"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQuickstartForInteractiveReturnsHumanGuide(t *testing.T) {
	out := quickstartFor(true, false)
	assert.Contains(t, out, "# Welcome to Databricks")
	assert.NotContains(t, out, "## Golden rules")
	assert.False(t, strings.HasSuffix(out, "\n"), "trailing newline should be trimmed; Fprintln re-adds one")
}

func TestQuickstartForNonInteractiveReturnsAgentGuide(t *testing.T) {
	out := quickstartFor(false, false)
	assert.Contains(t, out, "# Databricks Quickstart")
	assert.Contains(t, out, "## Golden rules")
	// Frontmatter must be stripped so the output starts at the heading.
	assert.True(t, strings.HasPrefix(out, "# Databricks Quickstart"))
	assert.NotContains(t, out, "name: databricks-quickstart")
}

func TestQuickstartForInteractiveIgnoresSkillsVariant(t *testing.T) {
	assert.Equal(t, quickstartFor(true, false), quickstartFor(true, true))
}

func TestQuickstartForNonInteractiveAitoolsVariant(t *testing.T) {
	out := quickstartFor(false, false)
	assert.Contains(t, out, "databricks aitools install")
	assert.NotContains(t, out, "databricks skills")
	assert.NotContains(t, out, "<!--")
}

func TestQuickstartForNonInteractiveSkillsVariant(t *testing.T) {
	out := quickstartFor(false, true)
	assert.True(t, strings.HasPrefix(out, "# Databricks Quickstart"))
	assert.Contains(t, out, "databricks skills list")
	assert.Contains(t, out, "databricks skills read <name>")
	assert.Contains(t, out, "databricks skills read-file <name> <path>")
	assert.NotContains(t, out, "aitools install")
	assert.NotContains(t, out, "<!--")
	for _, name := range []string{"databricks-core", "databricks-jobs", "databricks-pipelines", "databricks-dabs", "databricks-apps"} {
		assert.NotContains(t, out, name)
	}
}

func TestSkillForHasFrontmatterAndAgentBody(t *testing.T) {
	out := skillFor()
	assert.True(t, strings.HasPrefix(out, "---\nname: databricks-quickstart\ndescription: "), out)
	assert.NotContains(t, out, "<!--")
	assert.Equal(t, quickstartFor(false, true), stripFrontmatter(out))
}

func TestVariantText(t *testing.T) {
	const in = "shared\n<!--skills-->\nnew\n<!--aitools-->\nold\n<!--end-->\nafter"
	assert.Equal(t, "shared\nnew\nafter", variantText(in, true))
	assert.Equal(t, "shared\nold\nafter", variantText(in, false))
}

func TestVariantTextAcceptsCRLFMarkers(t *testing.T) {
	// A CRLF checkout (Git for Windows with core.autocrlf=true) leaves a \r on
	// each marker line. The markers must still match and be dropped.
	const in = "shared\r\n<!--skills-->\r\nnew\r\n<!--aitools-->\r\nold\r\n<!--end-->\r\nafter\r\n"
	assert.Equal(t, "shared\r\nnew\r\nafter\r\n", variantText(in, true))
	assert.Equal(t, "shared\r\nold\r\nafter\r\n", variantText(in, false))
}

func TestVariantTextCRLFMatchesLF(t *testing.T) {
	crlf := strings.ReplaceAll(agentQuickstart, "\n", "\r\n")
	for _, skillsVariant := range []bool{false, true} {
		assert.Equal(t, strings.ReplaceAll(variantText(agentQuickstart, skillsVariant), "\n", "\r\n"), variantText(crlf, skillsVariant))
	}
}

func TestSkillFlagRegisteredOnlyWhenSkillsEnabled(t *testing.T) {
	cases := []struct {
		name  string
		value string
		set   bool
		want  bool
	}{
		{name: "unset", want: false},
		{name: "false", set: true, value: "false", want: false},
		{name: "true", set: true, value: "true", want: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := t.Context()
			if c.set {
				ctx = env.Set(ctx, "DATABRICKS_SKILLS_PREVIEW", c.value)
			}
			cmd := &cobra.Command{}
			buildCommand(ctx, cmd)
			flag := cmd.Flags().Lookup("skill")
			if !c.want {
				assert.Nil(t, flag)
				return
			}
			require.NotNil(t, flag)
			assert.True(t, flag.Hidden)
		})
	}
}

func TestQuickstartVariantSelectedByEnv(t *testing.T) {
	cases := []struct {
		name       string
		value      string
		set        bool
		wantSkills bool
	}{
		{name: "unset", wantSkills: false},
		{name: "false", set: true, value: "false", wantSkills: false},
		{name: "true", set: true, value: "true", wantSkills: true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := cmdio.MockDiscard(t.Context())
			if c.set {
				ctx = env.Set(ctx, "DATABRICKS_SKILLS_PREVIEW", c.value)
			}
			cmd := &cobra.Command{}
			buildCommand(ctx, cmd)
			var out bytes.Buffer
			cmd.SetContext(ctx)
			cmd.SetOut(&out)
			// Run without --skill and with stdout not a terminal, so the agent text is printed.
			require.NoError(t, cmd.RunE(cmd, nil))
			if c.wantSkills {
				assert.Contains(t, out.String(), "databricks skills list")
				assert.NotContains(t, out.String(), "databricks aitools install")
			} else {
				assert.Contains(t, out.String(), "databricks aitools install")
				assert.NotContains(t, out.String(), "databricks skills list")
			}
		})
	}
}

func TestStripFrontmatter(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "with frontmatter",
			in:   "---\nname: x\ndescription: y\n---\n\n# Title\n\nbody\n",
			want: "# Title\n\nbody",
		},
		{
			name: "without frontmatter",
			in:   "# Title\n\nbody\n",
			want: "# Title\n\nbody",
		},
		{
			name: "unterminated frontmatter is left intact",
			in:   "---\nname: x\n# Title",
			want: "---\nname: x\n# Title",
		},
		{
			name: "empty",
			in:   "",
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, stripFrontmatter(tt.in))
		})
	}
}

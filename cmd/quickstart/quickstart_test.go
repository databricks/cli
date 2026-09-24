package quickstart

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQuickstartForInteractiveReturnsHumanGuide(t *testing.T) {
	out := quickstartFor(true)
	assert.Contains(t, out, "# Welcome to Databricks")
	assert.NotContains(t, out, "## Golden rules")
	assert.False(t, strings.HasSuffix(out, "\n"), "trailing newline should be trimmed; Fprintln re-adds one")
}

func TestQuickstartForNonInteractiveReturnsAgentGuide(t *testing.T) {
	out := quickstartFor(false)
	assert.Contains(t, out, "# Databricks Quickstart")
	assert.Contains(t, out, "## Golden rules")
	// Frontmatter must be stripped so the output starts at the heading.
	assert.True(t, strings.HasPrefix(out, "# Databricks Quickstart"))
	assert.NotContains(t, out, "name: databricks-quickstart")
}

func TestQuickstartUsesStdoutTTYCapability(t *testing.T) {
	ctx, testIO := cmdio.SetupTest(t.Context(), cmdio.TestOptions{PromptSupported: true})
	t.Cleanup(testIO.Done)

	t.Run("TTY stdout", func(t *testing.T) {
		var stdout bytes.Buffer
		cmd := New()
		cmd.SetArgs([]string{})
		cmd.SetContext(ctx)
		cmd.SetOut(cmdio.FakeTTY(&stdout))
		require.NoError(t, cmd.Execute())
		assert.Contains(t, stdout.String(), "# Welcome to Databricks")
		assert.NotContains(t, stdout.String(), "## Golden rules")
	})

	t.Run("piped stdout", func(t *testing.T) {
		var stdout bytes.Buffer
		cmd := New()
		cmd.SetArgs([]string{})
		cmd.SetContext(ctx)
		cmd.SetIn(io.NopCloser(strings.NewReader("")))
		cmd.SetOut(&stdout)
		cmd.SetErr(io.Discard)
		require.NoError(t, cmd.Execute())
		assert.Contains(t, stdout.String(), "# Databricks Quickstart")
		assert.Contains(t, stdout.String(), "## Golden rules")
	})
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

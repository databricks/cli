package skills

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	libskills "github.com/databricks/cli/libs/skills"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteReadTextQuotesFileCommands(t *testing.T) {
	var out bytes.Buffer
	err := writeReadText(&out, "linux", "guide-skill", []byte("# Guide\n"), []string{"references/my guide.md"})
	require.NoError(t, err)
	assert.Contains(t, out.String(), "  databricks skills read-file guide-skill 'references/my guide.md'\n")

	out.Reset()
	err = writeReadText(&out, "windows", "guide-skill", []byte("# Guide\n"), []string{"references/my guide.md"})
	require.NoError(t, err)
	assert.Contains(t, out.String(), "  databricks skills read-file guide-skill \"references/my guide.md\"\n")
}

func TestWriteReadTextEndsFlagsBeforeLeadingHyphenFile(t *testing.T) {
	var out bytes.Buffer
	err := writeReadText(&out, "linux", "guide-skill", []byte("# Guide\n"), []string{"-notes.md", "references/ok.md"})
	require.NoError(t, err)
	assert.Contains(t, out.String(), "  databricks skills read-file -- guide-skill -notes.md\n  databricks skills read-file guide-skill references/ok.md\n")
}

func TestWriteReadTextPrintsPathWithoutCommandWhenItCannotBeQuoted(t *testing.T) {
	var out bytes.Buffer
	err := writeReadText(&out, "linux", "guide-skill", []byte("# Guide\n"), []string{"references/a\nb.md", "references/ok.md"})
	require.NoError(t, err)
	assert.Contains(t, out.String(), "  references/a\nb.md\n  databricks skills read-file guide-skill references/ok.md\n")
}

// fakeSource is a libskills.Source that serves a fixed catalog and file contents.
type fakeSource struct {
	catalog *libskills.Catalog
	files   map[string][]byte
}

func (f fakeSource) Catalog(context.Context) (*libskills.Catalog, error) {
	return f.catalog, nil
}

func (f fakeSource) ReadFile(_ context.Context, skill, path string) ([]byte, error) {
	b, ok := f.files[skill+"/"+path]
	if !ok {
		return nil, errors.New("file not in the release")
	}
	return b, nil
}

// newReadTestCmd returns a command with the root's output flag, whose context
// captures stderr, and the buffer that captures stdout.
func newReadTestCmd(t *testing.T) (*cobra.Command, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	var stdout bytes.Buffer
	output := flags.OutputText
	cmd := &cobra.Command{}
	cmd.Flags().Var(&output, "output", "")
	cmd.SetOut(&stdout)
	cmd.SetContext(ctx)
	return cmd, &stdout, stderr
}

func TestRunReadStaleCatalogWritesProvenanceToStderr(t *testing.T) {
	src := fakeSource{
		catalog: &libskills.Catalog{
			Label:     "release cache",
			Version:   "0.0.1",
			FetchedAt: time.Date(2026, 11, 26, 10, 0, 0, 0, time.UTC),
			Stale:     true,
			Skills: []libskills.Skill{
				{Name: "demo-skill", Version: "0.0.1", Files: []string{"SKILL.md"}},
			},
		},
		files: map[string][]byte{"demo-skill/SKILL.md": []byte("# Demo\n")},
	}

	cmd, stdout, stderr := newReadTestCmd(t)
	err := runRead(cmd, src, "demo-skill")
	require.NoError(t, err)

	assert.Equal(t, "# Demo\n", stdout.String())
	assert.Equal(t, "Skills come from release cache (release 0.0.1, fetched 2026-11-26T10:00:00Z). "+
		"The source could not be refreshed, so the content may be out of date.\n", stderr.String())
}

func TestRunReadNamesReleaseWhenSkillHasNoSkillFile(t *testing.T) {
	src := fakeSource{
		catalog: &libskills.Catalog{
			Skills: []libskills.Skill{
				{Name: "broken-skill", Version: "0.0.1", Files: []string{"references/guide.md"}},
			},
		},
		files: map[string][]byte{},
	}

	cmd, stdout, _ := newReadTestCmd(t)
	err := runRead(cmd, src, "broken-skill")
	require.Error(t, err)
	assert.Contains(t, err.Error(), `skill "broken-skill" has no SKILL.md in this release`)
	assert.Empty(t, stdout.String())
}

package skills

import (
	"fmt"
	"io"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/shellquote"
	libskills "github.com/databricks/cli/libs/skills"
	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"
)

// skillFile is the file that holds a skill's instructions. Every other file of
// a skill is listed after it.
const skillFile = "SKILL.md"

func newReadCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "read <name>",
		Short: "Read a skill's instructions",
		Long: `Print a skill's SKILL.md, then the other files of the skill with the command
that reads each one. Nothing is installed.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			src, err := libskills.Resolve(cmd.Context())
			if err != nil {
				return err
			}
			return runRead(cmd, src, args[0])
		},
	}
}

// readOutput is the JSON output of `skills read`.
type readOutput struct {
	Name    string   `json:"name"`
	Version string   `json:"version"`
	Source  string   `json:"source"`
	Content string   `json:"content"`
	Files   []string `json:"files"`
}

func runRead(cmd *cobra.Command, src libskills.Source, name string) error {
	ctx := cmd.Context()
	catalog, err := src.Catalog(ctx)
	if err != nil {
		return err
	}
	if catalog.Stale {
		cmdio.LogString(ctx, readProvenance(catalog))
	}

	i := slices.IndexFunc(catalog.Skills, func(s libskills.Skill) bool { return s.Name == name })
	if i < 0 {
		return fmt.Errorf("%w %q: run \"databricks skills list\" to see the available skills", libskills.ErrUnknownSkill, name)
	}
	skill := catalog.Skills[i]

	content, err := src.ReadFile(ctx, name, skillFile)
	if err != nil {
		if !slices.Contains(skill.Files, skillFile) {
			return fmt.Errorf("skill %q has no %s in this release: %w", name, skillFile, err)
		}
		return err
	}

	info := build.GetInfo()
	if skill.MinCLIVer != "" && semver.Compare("v"+info.Version, "v"+skill.MinCLIVer) < 0 {
		cmdio.LogString(ctx, fmt.Sprintf("Warning: skill %q needs CLI version %s or newer, this CLI is %s. Update the CLI to use it.",
			name, skill.MinCLIVer, info.Version))
	}

	files := otherFiles(skill.Files)
	if root.OutputType(cmd) == flags.OutputJSON {
		return cmdio.Render(ctx, readOutput{
			Name:    name,
			Version: skill.Version,
			Source:  catalog.Label,
			Content: string(content),
			Files:   files,
		})
	}
	return writeReadText(cmd.OutOrStdout(), runtime.GOOS, name, content, files)
}

// otherFiles returns the skill's files other than SKILL.md, in manifest order.
// It is never nil, so the JSON output has an empty list rather than null.
func otherFiles(files []string) []string {
	others := make([]string, 0, len(files))
	for _, f := range files {
		if f != skillFile {
			others = append(others, f)
		}
	}
	return others
}

// writeReadText writes SKILL.md, with a trailing newline added if it is missing,
// then the other files with the command that reads each one. The commands are
// quoted for the shell of goos. A name or file that no quoting keeps literal
// is listed without its command. Nothing else goes to stdout.
func writeReadText(w io.Writer, goos, name string, content []byte, files []string) error {
	var b strings.Builder
	b.Write(content)
	if len(content) > 0 && content[len(content)-1] != '\n' {
		b.WriteString("\n")
	}
	if len(files) > 0 {
		b.WriteString("\nOther files in this skill, each read with its command:\n")
		quotedName, nameOK := shellquote.Arg(goos, name)
		for _, f := range files {
			quotedFile, fileOK := shellquote.Arg(goos, f)
			if !nameOK || !fileOK {
				fmt.Fprintf(&b, "  %s\n", f)
				continue
			}
			fmt.Fprintf(&b, "  databricks skills read-file %s %s\n", quotedName, quotedFile)
		}
	}
	_, err := io.WriteString(w, b.String())
	return err
}

// readProvenance says where a stale catalog came from. A stale catalog is a
// cached copy the live source could not refresh.
func readProvenance(catalog *libskills.Catalog) string {
	return fmt.Sprintf("Skills come from %s (release %s, fetched %s). The source could not be refreshed, so the content may be out of date.",
		catalog.Label, catalog.Version, catalog.FetchedAt.Format(time.RFC3339))
}

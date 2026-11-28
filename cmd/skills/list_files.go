package skills

import (
	"errors"
	"fmt"
	"slices"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	libskills "github.com/databricks/cli/libs/skills"
	"github.com/spf13/cobra"
)

// listedFile is one file of a skill in the JSON output. Size is set only when
// the source knows the size.
type listedFile struct {
	Path string `json:"path"`
	Size *int64 `json:"size,omitempty"`
}

func newListFilesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list-files <name>",
		Short: "List the files of a skill",
		Args:  cobra.ExactArgs(1),
		RunE:  runListFiles,
	}
}

func runListFiles(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	src, err := libskills.Resolve(ctx)
	if err != nil {
		return err
	}
	catalog, err := src.Catalog(ctx)
	if err != nil {
		return err
	}
	skill, err := findSkill(catalog, args[0])
	if err != nil {
		return err
	}
	if root.OutputType(cmd) != flags.OutputJSON {
		for _, file := range skill.Files {
			fmt.Fprintln(cmd.OutOrStdout(), file)
		}
		return nil
	}

	sizer, sized := src.(libskills.FileSizer)
	files := make([]listedFile, 0, len(skill.Files))
	for _, file := range skill.Files {
		entry := listedFile{Path: file}
		if sized {
			size, err := sizer.FileSize(ctx, skill.Name, file)
			switch {
			case errors.Is(err, libskills.ErrNoSize):
				// The file is listed without a size.
			case err != nil:
				return err
			default:
				entry.Size = &size
			}
		}
		files = append(files, entry)
	}
	return cmdio.Render(ctx, files)
}

// findSkill returns the catalog entry of the named skill.
func findSkill(catalog *libskills.Catalog, name string) (libskills.Skill, error) {
	i := slices.IndexFunc(catalog.Skills, func(s libskills.Skill) bool { return s.Name == name })
	if i < 0 {
		return libskills.Skill{}, fmt.Errorf("%w: %q", libskills.ErrUnknownSkill, name)
	}
	return catalog.Skills[i], nil
}

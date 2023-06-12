package fs

import (
	"context"
	"errors"
	"io/fs"
	"path"
	"path/filepath"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/filer"
	"github.com/spf13/cobra"
)

func cpWriteCallback(ctx context.Context, sourceFiler, targetFiler filer.Filer, sourceDir, targetDir string) func(string, fs.DirEntry, error) error {
	return func(relPath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		sourcePath := path.Join(sourceDir, relPath)
		targetPath := filepath.Join(targetDir, relPath)

		// create directory and return early
		if d.IsDir() {
			return targetFiler.Mkdir(ctx, relPath)
		}

		// get reader for source file
		r, err := sourceFiler.Read(ctx, relPath)
		if err != nil {
			return err
		}

		// write to target file
		err = targetFiler.Write(ctx, relPath, r)
		if errors.Is(err, fs.ErrExist) {
			fileSkippedEvent := newFileSkippedEvent(sourcePath, targetPath)
			template := "{{.SourcePath}} -> {{.TargetPath}} (skipped; already exists)\n"
			return cmdio.RenderWithTemplate(ctx, fileSkippedEvent, template)
		}

	}
}

var cpOverwrite bool
var cpRecursive bool

// TODO: error out if source is a directory and recursive is not specified

// cpCmd represents the fs cp command
var cpCmd = &cobra.Command{
	Use:     "cp SOURCE_PATH TARGET_PATH",
	Short:   "Copy files to and from DBFS.",
	Long:    `TODO`,
	Args:    cobra.ExactArgs(1),
	PreRunE: root.MustWorkspaceClient,

	RunE: func(cmd *cobra.Command, args []string) error {

	},
}

func init() {
	cpCmd.Flags().BoolVar(&cpOverwrite, "overwrite", false, "overwrite existing files")
	cpCmd.Flags().BoolVarP(&cpRecursive, "recursive", "r", false, "recursively copy files from directory")
	fsCmd.AddCommand(cpCmd)
}

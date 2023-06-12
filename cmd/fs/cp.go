package fs

import (
	"context"
	"io/fs"
	"path"
	"path/filepath"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/filer"
	"github.com/spf13/cobra"
)

func dbfsWriteCallback(ctx context.Context, sourceFiler filer.Filer, sourceDir, targetDir string) func(string, fs.DirEntry, error) error {
	return func(sourcePath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		sourceName := 

		if d.IsDir() {

		}
	}
}

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
	fsCmd.AddCommand(cpCmd)
}

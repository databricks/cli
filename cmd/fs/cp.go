package fs

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/filer"
	"github.com/spf13/cobra"
)

func cpWriteCallback(ctx context.Context, sourceFiler, targetFiler filer.Filer, sourceDir, targetDir string) func(string, fs.DirEntry, error) error {
	return func(sourcePath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Compute path relative to the target directory
		relPath, err := filepath.Rel(sourceDir, sourcePath)
		if err != nil {
			return err
		}
		relPath = filepath.ToSlash(relPath)

		// Compute target path for the file
		targetPath := path.Join(targetDir, relPath)

		fmt.Printf("[CLI_COPY] relPath computation. (sourcePath: %s, targetPath: %s, sourceDir: %s, targetDir: %s, relPath: %s)\n", sourcePath, targetPath, sourceDir, targetDir, relPath)

		// create directory and return early
		if d.IsDir() {
			fmt.Printf("[CLI_COPY] target mkdir FILER call. (path: %s)\n", targetPath)
			return targetFiler.Mkdir(ctx, targetPath)
		}

		return cpFileToFile(ctx, sourcePath, targetPath, sourceFiler, targetFiler)
	}
}

func cpDirToDir(ctx context.Context, sourceFiler, targetFiler filer.Filer, sourceDir, targetDir string) error {
	if !cpRecursive {
		return fmt.Errorf("source path %s is a directory. Please specify the --recursive flag", sourceDir)
	}

	sourceFs := filer.NewFS(ctx, sourceFiler)
	return fs.WalkDir(sourceFs, sourceDir, cpWriteCallback(ctx, sourceFiler, targetFiler, sourceDir, targetDir))
}

func cpFileToDir(ctx context.Context, sourcePath, targetDir string, sourceFiler, targetFiler filer.Filer) error {
	fileName := path.Base(sourcePath)
	targetPath := path.Join(targetDir, fileName)

	return cpFileToFile(ctx, sourcePath, targetPath, sourceFiler, targetFiler)
}

func cpFileToFile(ctx context.Context, sourcePath, targetPath string, sourceFiler, targetFiler filer.Filer) error {
	// Get reader for file at source path
	r, err := sourceFiler.Read(ctx, sourcePath)
	if err != nil {
		return err
	}
	defer r.Close()

	fmt.Printf("[CLI_COPY] target write FILER call. (path: %s)\n", targetPath)
	if cpOverwrite {
		err = targetFiler.Write(ctx, targetPath, r, filer.OverwriteIfExists)
		if err != nil {
			return err
		}
	} else {
		err = targetFiler.Write(ctx, targetPath, r)
		// skip if file already exists
		if err != nil && errors.Is(err, fs.ErrExist) {
			return emitCpFileSkippedEvent(ctx, sourcePath, targetPath)
		}
		if err != nil {
			return err
		}
	}
	return emitCpFileCopiedEvent(ctx, sourcePath, targetPath)
}

// TODO: emit these events on stderr
// TODO: add integration tests for these events
func emitCpFileSkippedEvent(ctx context.Context, sourcePath, targetPath string) error {
	event := newFileSkippedEvent(sourcePath, targetPath)
	template := "{{.SourcePath}} -> {{.TargetPath}} (skipped; already exists)\n"

	return cmdio.RenderWithTemplate(ctx, event, template)
}

func emitCpFileCopiedEvent(ctx context.Context, sourcePath, targetPath string) error {
	event := newFileCopiedEvent(sourcePath, targetPath)
	template := "{{.SourcePath}} -> {{.TargetPath}}\n"

	return cmdio.RenderWithTemplate(ctx, event, template)
}

var cpOverwrite bool
var cpRecursive bool

// cpCmd represents the fs cp command
var cpCmd = &cobra.Command{
	Use:   "cp SOURCE_PATH TARGET_PATH",
	Short: "Copy files and directories to and from DBFS.",
	Long: `Copy files to and from DBFS.

  It is required that you specify the scheme "file" for local files and
  "dbfs" for dbfs files. For example: file:/foo/bar, file:/c:/foo/bar or dbfs:/foo/bar.

  Recursively copying a directory will copy all files inside directory
  at SOURCE_PATH to the directory at TARGET_PATH.

  When copying a file, if TARGET_PATH is a directory, the file will be created
  inside the directory, otherwise the file is created at TARGET_PATH.
`,
	Args:    cobra.ExactArgs(2),
	PreRunE: root.MustWorkspaceClient,

	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		// TODO: Error if a user uses '\' as path separator on windows when "file"
		// scheme is specified (https://github.com/databricks/cli/issues/485)

		// Get source filer and source path without scheme
		fullSourcePath := args[0]
		sourceFiler, sourcePath, err := filerForPath(ctx, fullSourcePath)
		if err != nil {
			return err
		}

		// Get target filer and target path without scheme
		fullTargetPath := args[1]
		targetFiler, targetPath, err := filerForPath(ctx, fullTargetPath)
		if err != nil {
			return err
		}

		// Get information about file at source path
		fmt.Printf("[CLI_COPY] source stat FILER call. (path: %s)\n", sourcePath)
		sourceInfo, err := sourceFiler.Stat(ctx, sourcePath)
		if err != nil {
			return err
		}

		// case 1: source path is a directory, then recursively create files at target path
		if sourceInfo.IsDir() {
			return cpDirToDir(ctx, sourceFiler, targetFiler, sourcePath, targetPath)
		}

		// case 2: source path is a file, and target path is a directory. In this case
		// we copy the file to inside the directory
		fmt.Printf("[CLI_COPY] target stat FILER call. (path: %s)\n", targetPath)
		if targetInfo, err := targetFiler.Stat(ctx, targetPath); err == nil && targetInfo.IsDir() {
			return cpFileToDir(ctx, sourcePath, targetPath, sourceFiler, targetFiler)
		}

		// case 3: source path is a file, and target path is a file
		return cpFileToFile(ctx, sourcePath, targetPath, sourceFiler, targetFiler)
	},
}

func init() {
	cpCmd.Flags().BoolVar(&cpOverwrite, "overwrite", false, "overwrite existing files")
	cpCmd.Flags().BoolVarP(&cpRecursive, "recursive", "r", false, "recursively copy files from directory")
	fsCmd.AddCommand(cpCmd)
}

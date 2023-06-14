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
		targetPath := filepath.Join(targetDir, relPath)

		// create directory and return early
		if d.IsDir() {
			return targetFiler.Mkdir(ctx, targetPath)
		}

		// get reader for source file
		r, err := sourceFiler.Read(ctx, sourcePath)
		if err != nil {
			return err
		}

		// write to target file
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

func validateScheme(path string) error {
	scheme := scheme(path)
	if scheme == NoScheme {
		return fmt.Errorf(`no scheme specified for path %s. Please specify scheme "dbfs" or "file". Example: file:/foo/bar`, path)
	}
	if scheme != LocalScheme && scheme != DbfsScheme {
		return fmt.Errorf(`unsupported scheme %s specified for path %s. Please specify scheme "dbfs" or "file". Example: file:/foo/bar`, scheme, path)
	}
	return nil
}

// cpCmd represents the fs cp command
var cpCmd = &cobra.Command{
	Use:     "cp SOURCE_PATH TARGET_PATH",
	Short:   "Copy files to and from DBFS.",
	Long:    `TODO`,
	Args:    cobra.ExactArgs(2),
	PreRunE: root.MustWorkspaceClient,

	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		// Validate source path scheme, and compute path without the scheme
		fullSourcePath := args[0]
		sourceScheme := scheme(args[0])
		if err := validateScheme(fullSourcePath); err != nil {
			return err
		}
		sourcePath, err := removeScheme(fullSourcePath, sourceScheme)
		if err != nil {
			return err
		}

		// Validate target path scheme, and compute path without the scheme
		fullTargetPath := args[1]
		targetScheme := scheme(args[1])
		if err := validateScheme(fullTargetPath); err != nil {
			return err
		}
		targetPath, err := removeScheme(fullTargetPath, targetScheme)
		if err != nil {
			return err
		}

		// Initialize filers according to source and target path schemes
		sourceFiler, err := filerForScheme(ctx, sourceScheme)
		if err != nil {
			return err
		}
		targetFiler, err := filerForScheme(ctx, targetScheme)
		if err != nil {
			return err
		}

		// Get information about file at source path
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

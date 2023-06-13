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
		if cpOverwrite {
			err = targetFiler.Write(ctx, relPath, r, filer.OverwriteIfExists)
			if err != nil {
				return err
			}
		} else {
			err = targetFiler.Write(ctx, relPath, r)
			// skip if file already exists
			if err != nil && errors.Is(err, fs.ErrExist) {
				fileSkippedEvent := newFileSkippedEvent(sourcePath, targetPath)
				template := "{{.SourcePath}} -> {{.TargetPath}} (skipped; already exists)\n"
				return cmdio.RenderWithTemplate(ctx, fileSkippedEvent, template)
			}
			if err != nil {
				return err
			}
		}

		return cmdio.RenderWithTemplate(ctx, newFileCopiedEvent(sourcePath, targetPath), "{{.SourcePath}} -> {{.TargetPath}}")
	}
}

// TODO: just use the root filer
func copyDirToDir(ctx context.Context, sourceDir string, targetDir string) error {
	sourceFiler, err := setupFiler(ctx, sourceDir)
	if err != nil {
		return err
	}
	sourceFs := filer.NewFS(ctx, sourceFiler)
	targetFiler, err := setupFiler(ctx, targetDir)
	if err != nil {
		return err
	}

	return fs.WalkDir(sourceFs, ".", cpWriteCallback(ctx, sourceFiler, targetFiler, sourceDir, targetDir))
}

var cpOverwrite bool
var cpRecursive bool

func validateScheme(path string) error {
	scheme := scheme(path)
	if scheme != LocalScheme && scheme != DbfsScheme {
		return fmt.Errorf(`no scheme specified for path %s. Please specify scheme "dbfs" or "file". Example: file:/foo/bar`, path)
	}
	return nil
}

// TODO: error out if source is a directory and recursive is not specified

// cpCmd represents the fs cp command
var cpCmd = &cobra.Command{
	Use:     "cp SOURCE_PATH TARGET_PATH",
	Short:   "Copy files to and from DBFS.",
	Long:    `TODO`,
	Args:    cobra.ExactArgs(2),
	PreRunE: root.MustWorkspaceClient,

	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		// Validate input path scheme
		sourceScheme := scheme(args[0])
		sourcePath := args[0]
		if err := validateScheme(sourcePath); err != nil {
			return err
		}
		targetPath := args[1]
		targetScheme := scheme(args[1])
		if err := validateScheme(targetPath); err != nil {
			return err
		}

		cleanSourcePath, err := removeScheme(sourcePath, sourceScheme)
		if err != nil {
			return err
		}
		sourceRootFiler, err := setupRootFiler(ctx, sourceScheme)
		if err != nil {
			return err
		}
		sourceStat, err := sourceRootFiler.Stat(ctx, cleanSourcePath)
		if err != nil {
			return err
		}

		targetExists := true
		cleanTargetPath, err := removeScheme(targetPath, targetScheme)
		if err != nil {
			return err
		}
		targetRootFiler, err := setupRootFiler(ctx, targetScheme)
		if err != nil {
			return err
		}
		targetStat, err := targetRootFiler.Stat(ctx, cleanTargetPath)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				targetExists = false
			} else {
				return err
			}
		}

		if sourceStat.IsDir() {
			if !cpRecursive {
				return fmt.Errorf("source path %s is a directory. Please specify the --recursive flag", sourcePath)
			}
			return copyDirToDir(ctx, sourcePath, targetPath)
		}

		r, err := sourceRootFiler.Read(ctx, cleanSourcePath)
		if err != nil {
			return err
		}
		if targetExists && targetStat.IsDir() {
			name := path.Base(cleanSourcePath)
			if cpOverwrite {
				return targetRootFiler.Write(ctx, path.Join(cleanTargetPath, name), r, filer.OverwriteIfExists)
			}
			return targetRootFiler.Write(ctx, path.Join(cleanTargetPath, name), r)
		}

		if cpOverwrite {
			return targetRootFiler.Write(ctx, cleanTargetPath, r, filer.OverwriteIfExists)
		}
		return targetRootFiler.Write(ctx, cleanTargetPath, r)
	},
}

func init() {
	cpCmd.Flags().BoolVar(&cpOverwrite, "overwrite", false, "overwrite existing files")
	cpCmd.Flags().BoolVarP(&cpRecursive, "recursive", "r", false, "recursively copy files from directory")
	fsCmd.AddCommand(cpCmd)
}

/*
+
+source path:
+1. Read scheme
+2. Check if path is file or directory. If it does not exist, return error
+3. If it's a directory, check recursive flag is specified
+4.
+
+*/

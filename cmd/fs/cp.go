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

type copy struct {
	overwrite bool
	recursive bool

	ctx          context.Context
	sourceFiler  filer.Filer
	targetFiler  filer.Filer
	sourceScheme string
	targetScheme string
}

func (c *copy) cpWriteCallback(sourceDir, targetDir string) fs.WalkDirFunc {
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

		// create directory and return early
		if d.IsDir() {
			return c.targetFiler.Mkdir(c.ctx, targetPath)
		}

		return c.cpFileToFile(sourcePath, targetPath)
	}
}

func (c *copy) cpDirToDir(sourceDir, targetDir string) error {
	if !c.recursive {
		return fmt.Errorf("source path %s is a directory. Please specify the --recursive flag", sourceDir)
	}

	sourceFs := filer.NewFS(c.ctx, c.sourceFiler)
	return fs.WalkDir(sourceFs, sourceDir, c.cpWriteCallback(sourceDir, targetDir))
}

func (c *copy) cpFileToDir(sourcePath, targetDir string) error {
	fileName := filepath.Base(sourcePath)
	targetPath := path.Join(targetDir, fileName)

	return c.cpFileToFile(sourcePath, targetPath)
}

func (c *copy) cpFileToFile(sourcePath, targetPath string) error {
	// Get reader for file at source path
	r, err := c.sourceFiler.Read(c.ctx, sourcePath)
	if err != nil {
		return err
	}
	defer r.Close()

	if c.overwrite {
		err = c.targetFiler.Write(c.ctx, targetPath, r, filer.OverwriteIfExists)
		if err != nil {
			return err
		}
	} else {
		err = c.targetFiler.Write(c.ctx, targetPath, r)
		// skip if file already exists
		if err != nil && errors.Is(err, fs.ErrExist) {
			return c.emitFileSkippedEvent(sourcePath, targetPath)
		}
		if err != nil {
			return err
		}
	}
	return c.emitFileCopiedEvent(sourcePath, targetPath)
}

// TODO: emit these events on stderr
// TODO: add integration tests for these events
func (c *copy) emitFileSkippedEvent(sourcePath, targetPath string) error {
	fullSourcePath := sourcePath
	if c.sourceScheme != "" {
		fullSourcePath = path.Join(c.sourceScheme+":", sourcePath)
	}
	fullTargetPath := targetPath
	if c.targetScheme != "" {
		fullTargetPath = path.Join(c.targetScheme+":", targetPath)
	}

	event := newFileSkippedEvent(fullSourcePath, fullTargetPath)
	template := "{{.SourcePath}} -> {{.TargetPath}} (skipped; already exists)\n"

	return cmdio.RenderWithTemplate(c.ctx, event, "", template)
}

func (c *copy) emitFileCopiedEvent(sourcePath, targetPath string) error {
	fullSourcePath := sourcePath
	if c.sourceScheme != "" {
		fullSourcePath = path.Join(c.sourceScheme+":", sourcePath)
	}
	fullTargetPath := targetPath
	if c.targetScheme != "" {
		fullTargetPath = path.Join(c.targetScheme+":", targetPath)
	}

	event := newFileCopiedEvent(fullSourcePath, fullTargetPath)
	template := "{{.SourcePath}} -> {{.TargetPath}}\n"

	return cmdio.RenderWithTemplate(c.ctx, event, "", template)
}

func newCpCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cp SOURCE_PATH TARGET_PATH",
		Short: "Copy files and directories.",
		Long: `Copy files and directories to and from any paths on DBFS, UC Volumes or your local filesystem.

	  For paths in DBFS and UC Volumes, it is required that you specify the "dbfs" scheme.
	  For example: dbfs:/foo/bar.

	  Recursively copying a directory will copy all files inside directory
	  at SOURCE_PATH to the directory at TARGET_PATH.

	  When copying a file, if TARGET_PATH is a directory, the file will be created
	  inside the directory, otherwise the file is created at TARGET_PATH.
	`,
		Args:    root.ExactArgs(2),
		PreRunE: root.MustWorkspaceClient,
	}

	var c copy
	cmd.Flags().BoolVar(&c.overwrite, "overwrite", false, "overwrite existing files")
	cmd.Flags().BoolVarP(&c.recursive, "recursive", "r", false, "recursively copy files from directory")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

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

		c.sourceScheme = ""
		if isDbfsPath(fullSourcePath) {
			c.sourceScheme = "dbfs"
		}
		c.targetScheme = ""
		if isDbfsPath(fullTargetPath) {
			c.targetScheme = "dbfs"
		}

		c.ctx = ctx
		c.sourceFiler = sourceFiler
		c.targetFiler = targetFiler

		// Get information about file at source path
		sourceInfo, err := sourceFiler.Stat(ctx, sourcePath)
		if err != nil {
			return err
		}

		// case 1: source path is a directory, then recursively create files at target path
		if sourceInfo.IsDir() {
			return c.cpDirToDir(sourcePath, targetPath)
		}

		// case 2: source path is a file, and target path is a directory. In this case
		// we copy the file to inside the directory
		if targetInfo, err := targetFiler.Stat(ctx, targetPath); err == nil && targetInfo.IsDir() {
			return c.cpFileToDir(sourcePath, targetPath)
		}

		// case 3: source path is a file, and target path is a file
		return c.cpFileToFile(sourcePath, targetPath)
	}

	v := newValidArgs()
	// The copy command has two paths that can be completed (SOURCE_PATH & TARGET_PATH)
	v.pathArgCount = 2
	cmd.ValidArgsFunction = v.Validate

	return cmd
}

package fs

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/filer"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

// Default number of concurrent file copy operations.
const defaultConcurrency = 16

// errInvalidConcurrency is returned when the value of the concurrency
// flag is invalid.
var errInvalidConcurrency = errors.New("--concurrency must be at least 1")

type copy struct {
	overwrite   bool
	recursive   bool
	concurrency int

	sourceFiler  filer.Filer
	targetFiler  filer.Filer
	sourceScheme string
	targetScheme string

	mu sync.Mutex // protect output from concurrent writes
}

// cpDirToDir recursively copies the contents of a directory to another
// directory.
//
// There is no guarantee on the order in which the files are copied.
//
// The method does not take care of retrying on error; this is considered to
// be the responsibility of the Filer implementation. If a file copy fails,
// the error is returned and the other copies are cancelled.
func (c *copy) cpDirToDir(ctx context.Context, sourceDir, targetDir string) error {
	if !c.recursive {
		return fmt.Errorf("source path %s is a directory. Please specify the --recursive flag", sourceDir)
	}

	// Create cancellable context to ensure cleanup and that all goroutines
	// are stopped when the function exits on any error path (e.g. permission
	// denied when walking the source directory).
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Pool of workers to process copy operations in parallel.
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(c.concurrency)

	// Walk the source directory, queueing file copy operations for processing.
	sourceFs := filer.NewFS(ctx, c.sourceFiler)
	err := fs.WalkDir(sourceFs, sourceDir, func(sourcePath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Compute path relative to the source directory.
		relPath, err := filepath.Rel(sourceDir, sourcePath)
		if err != nil {
			return err
		}
		relPath = filepath.ToSlash(relPath)

		// Compute target path for the file.
		targetPath := path.Join(targetDir, relPath)

		// Create the directory synchronously. This must happen before files
		// are copied into it, and WalkDir guarantees directories are visited
		// before their contents.
		if d.IsDir() {
			return c.targetFiler.Mkdir(ctx, targetPath)
		}

		// Queue file copy operation for processing.
		g.Go(func() error {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return c.cpFileToFile(ctx, sourcePath, targetPath)
		})
		return nil
	})
	if err != nil {
		return err
	}
	return g.Wait()
}

func (c *copy) cpFileToDir(ctx context.Context, sourcePath, targetDir string) error {
	fileName := filepath.Base(sourcePath)
	targetPath := path.Join(targetDir, fileName)

	return c.cpFileToFile(ctx, sourcePath, targetPath)
}

func (c *copy) cpFileToFile(ctx context.Context, sourcePath, targetPath string) error {
	// Get reader for file at source path
	r, err := c.sourceFiler.Read(ctx, sourcePath)
	if err != nil {
		return err
	}
	defer r.Close()

	if c.overwrite {
		err = c.targetFiler.Write(ctx, targetPath, r, filer.OverwriteIfExists)
		if err != nil {
			return err
		}
	} else {
		err = c.targetFiler.Write(ctx, targetPath, r)
		// skip if file already exists
		if err != nil && errors.Is(err, fs.ErrExist) {
			return c.emitFileSkippedEvent(ctx, sourcePath, targetPath)
		}
		if err != nil {
			return err
		}
	}
	return c.emitFileCopiedEvent(ctx, sourcePath, targetPath)
}

// TODO: emit these events on stderr
// TODO: add integration tests for these events
func (c *copy) emitFileSkippedEvent(ctx context.Context, sourcePath, targetPath string) error {
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

	c.mu.Lock()
	defer c.mu.Unlock()
	return cmdio.RenderWithTemplate(ctx, event, "", template)
}

func (c *copy) emitFileCopiedEvent(ctx context.Context, sourcePath, targetPath string) error {
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

	c.mu.Lock()
	defer c.mu.Unlock()
	return cmdio.RenderWithTemplate(ctx, event, "", template)
}

// hasTrailingDirSeparator checks if a path ends with a directory separator.
func hasTrailingDirSeparator(path string) bool {
	return strings.HasSuffix(path, string(os.PathSeparator))
}

// trimTrailingDirSeparators removes all trailing directory separators from a path.
func trimTrailingDirSeparators(path string) string {
	return strings.TrimRight(path, string(os.PathSeparator))
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
		Args: root.ExactArgs(2),
	}

	var c copy
	cmd.Flags().BoolVar(&c.overwrite, "overwrite", false, "overwrite existing files")
	cmd.Flags().BoolVarP(&c.recursive, "recursive", "r", false, "recursively copy files from directory")
	cmd.Flags().IntVar(&c.concurrency, "concurrency", defaultConcurrency, "number of parallel copy operations")

	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if c.concurrency <= 0 {
			return errInvalidConcurrency
		}
		return root.MustWorkspaceClient(cmd, args)
	}

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

		c.sourceFiler = sourceFiler
		c.targetFiler = targetFiler

		// Get information about file at source path
		sourceInfo, err := sourceFiler.Stat(ctx, sourcePath)
		if err != nil {
			return err
		}

		// case 1: source path is a directory, then recursively create files at target path
		if sourceInfo.IsDir() {
			return c.cpDirToDir(ctx, sourcePath, targetPath)
		}

		// If target path has a trailing separator, trim it and let case 2 handle it
		if hasTrailingDirSeparator(fullTargetPath) {
			targetPath = trimTrailingDirSeparators(targetPath)
		}

		// case 2: source path is a file, and target path is a directory. In this case
		// we copy the file to inside the directory
		if targetInfo, err := targetFiler.Stat(ctx, targetPath); err == nil && targetInfo.IsDir() {
			return c.cpFileToDir(ctx, sourcePath, targetPath)
		}

		// case 3: source path is a file, and target path is a file
		return c.cpFileToFile(ctx, sourcePath, targetPath)
	}

	v := newValidArgs()
	// The copy command has two paths that can be completed (SOURCE_PATH & TARGET_PATH)
	v.pathArgCount = 2
	cmd.ValidArgsFunction = v.Validate

	return cmd
}

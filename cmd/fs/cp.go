package fs

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"
)

// defaultConcurrency is the number of files copied in parallel. Each in-flight
// file drives Files API requests (a single-shot PUT, or the multipart
// control-plane calls), which allow only ~10 concurrent requests, so this stays
// conservative regardless of whether multipart upload is enabled.
const defaultConcurrency = 8

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

	// showProgress renders an upload progress bar. It is set only for a single
	// large-file copy to a Volume, not for recursive copies (where many files
	// would each fight for the spinner line).
	showProgress bool

	// outputJSON is set when the command renders JSON. A recursive copy then
	// leaves out its one-line summary of a complete copy.
	outputJSON bool

	mu sync.Mutex // protect output from concurrent writes
}

// cpDirToDir recursively copies the content of a directory to another
// directory. There is no guarantee on the order in which the files are copied.
// It keeps going past failed files; see [recursiveCopy].
func (c *copy) cpDirToDir(ctx context.Context, sourceDir, targetDir string) error {
	if !c.recursive {
		return fmt.Errorf("source path %s is a directory. Please specify the --recursive flag", sourceDir)
	}
	return (&recursiveCopy{c: c}).run(ctx, sourceDir, targetDir)
}

func (c *copy) cpFileToDir(ctx context.Context, sourcePath, targetDir string) error {
	fileName := filepath.Base(sourcePath)
	targetPath := path.Join(targetDir, fileName)

	return c.cpFileToFile(ctx, sourcePath, targetPath)
}

func (c *copy) cpFileToFile(ctx context.Context, sourcePath, targetPath string) error {
	skipped, err := c.writeFile(ctx, sourcePath, targetPath)
	if err != nil {
		return err
	}
	return c.emitResult(ctx, sourcePath, targetPath, skipped)
}

// writeFile copies one file without printing its line, and reports whether it
// was skipped because the target exists. extraModes are passed to Write, in
// addition to OverwriteIfExists when --overwrite is set.
func (c *copy) writeFile(ctx context.Context, sourcePath, targetPath string, extraModes ...filer.WriteMode) (bool, error) {
	// Get reader for file at source path
	r, err := c.sourceFiler.Read(ctx, sourcePath)
	if err != nil {
		return false, err
	}
	defer r.Close()

	// For a single large-file copy, attach a progress callback to the context
	// that the Files filer forwards to the upload engine, rendering an upload bar.
	// The spinner is stopped before any event line is emitted so its final frame
	// does not overwrite it.
	closeProgress := func() {}
	if c.showProgress {
		fn, stop := newProgressFunc(ctx)
		ctx = filer.WithUploadProgress(ctx, fn)
		closeProgress = stop
	}

	modes := slices.Clone(extraModes)
	if c.overwrite {
		modes = append(modes, filer.OverwriteIfExists)
	}
	writeErr := c.targetFiler.Write(ctx, targetPath, r, modes...)
	closeProgress()

	// skip if file already exists
	if !c.overwrite && writeErr != nil && errors.Is(writeErr, fs.ErrExist) {
		return true, nil
	}
	return false, writeErr
}

// emitResult prints the line for a copied or skipped file.
func (c *copy) emitResult(ctx context.Context, sourcePath, targetPath string, skipped bool) error {
	if skipped {
		return c.emitFileSkippedEvent(ctx, sourcePath, targetPath)
	}
	return c.emitFileCopiedEvent(ctx, sourcePath, targetPath)
}

// fullPaths returns the source and target paths with their scheme, as shown to
// the user.
func (c *copy) fullPaths(sourcePath, targetPath string) (string, string) {
	if c.sourceScheme != "" {
		sourcePath = path.Join(c.sourceScheme+":", sourcePath)
	}
	if c.targetScheme != "" {
		targetPath = path.Join(c.targetScheme+":", targetPath)
	}
	return sourcePath, targetPath
}

// TODO: emit these events on stderr
// TODO: add integration tests for these events
func (c *copy) emitFileSkippedEvent(ctx context.Context, sourcePath, targetPath string) error {
	fullSourcePath, fullTargetPath := c.fullPaths(sourcePath, targetPath)

	event := newFileSkippedEvent(fullSourcePath, fullTargetPath)
	template := "{{.SourcePath}} -> {{.TargetPath}} (skipped; already exists)\n"

	c.mu.Lock()
	defer c.mu.Unlock()
	return cmdio.RenderWithTemplate(ctx, event, "", template)
}

func (c *copy) emitFileCopiedEvent(ctx context.Context, sourcePath, targetPath string) error {
	fullSourcePath, fullTargetPath := c.fullPaths(sourcePath, targetPath)

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
		Long: `Copy files and directories to and from any paths on DBFS, UC Volumes, UC Skills or your local filesystem.

	  For paths in DBFS, UC Volumes and UC Skills, it is required that you specify the "dbfs" scheme.
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
		c.outputJSON = root.OutputType(cmd) == flags.OutputJSON

		// Get source filer and source path without scheme
		fullSourcePath := args[0]
		sourceFiler, sourcePath, err := filerForPath(ctx, fullSourcePath)
		if err != nil {
			return err
		}

		// Get target filer and target path without scheme.
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

		// A single large file copied to a Volume goes through the multipart engine,
		// which reports progress; render an upload bar for it.
		c.showProgress = filer.MultipartUploadEnabled(ctx) && strings.HasPrefix(targetPath, "/Volumes/")

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

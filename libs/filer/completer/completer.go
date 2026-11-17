package completer

import (
	"context"
	"path"
	"path/filepath"

	"github.com/databricks/cli/libs/filer"
	"github.com/spf13/cobra"
)

type completer struct {
	ctx context.Context

	// The filer to use for completing remote or local paths.
	filer filer.Filer

	// CompletePath will only return directories when onlyDirs is true.
	onlyDirs bool

	// Prefix to prepend to completions.
	prefix string

	// Whether the path is local or remote. If the path is local we use the `filepath`
	// package for path manipulation. Otherwise we use the `path` package.
	isLocalPath bool
}

// General completer that takes a filer to complete remote paths when TAB-ing through a path.
func New(ctx context.Context, filer filer.Filer, onlyDirs bool) *completer {
	return &completer{ctx: ctx, filer: filer, onlyDirs: onlyDirs, prefix: "", isLocalPath: true}
}

func (c *completer) SetPrefix(p string) {
	c.prefix = p
}

func (c *completer) SetIsLocalPath(i bool) {
	c.isLocalPath = i
}

func (c *completer) CompletePath(p string) ([]string, cobra.ShellCompDirective, error) {
	trailingSeparator := "/"
	joinFunc := path.Join
	splitFunc := path.Split

	// Use filepath functions if we are in a local path. On Windows,
	// filepath.Split splits on both '/' and '\'.
	if c.isLocalPath {
		joinFunc = filepath.Join
		splitFunc = filepath.Split
		trailingSeparator = string(filepath.Separator)
	}

	// Shells only offer completions that start with the word being completed,
	// so keep the directory part of p as typed. Joining it with the entry name
	// would clean it and turn "./sr" into "src", which the shell then discards.
	typedDir, _ := splitFunc(p)

	// List the directory part of p: nested entries after a trailing separator,
	// adjacent ones for a partial name. Split it the same way as typedDir;
	// path.Dir only splits on '/', so on Windows it would list the current
	// directory for ".\src\s" and pair its entries with the typed ".\src\".
	dirPath := typedDir
	if dirPath == "" {
		dirPath = "."
	}

	entries, err := c.filer.ReadDir(c.ctx, dirPath)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError, err
	}

	var completions []string
	for _, entry := range entries {
		if c.onlyDirs && !entry.IsDir() {
			continue
		}

		completion := typedDir + entry.Name()

		// Prepend prefix if it has been set
		if c.prefix != "" {
			completion = joinFunc(c.prefix, completion)
		}

		// Add trailing separator for directories.
		if entry.IsDir() {
			completion += trailingSeparator
		}

		completions = append(completions, completion)
	}

	// If the path is local, we add the dbfs:/ prefix suggestion as an option
	if c.isLocalPath {
		completions = append(completions, "dbfs:/")
	}

	return completions, cobra.ShellCompDirectiveNoSpace, err
}

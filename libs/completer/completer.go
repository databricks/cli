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

	prefix string

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
	// If the user is TAB-ing their way through a path, the path in `toComplete`
	// is valid and we should list nested directories.
	// If the path in `toComplete` is incomplete, however,
	// then we should list adjacent directories.
	dirPath := p
	completions, err := fetchCompletions(c, dirPath)
	if completions == nil && err == nil && dirPath != path.Dir(p) {
		dirPath = path.Dir(p)
		completions, err = fetchCompletions(c, dirPath)
	}

	return completions, cobra.ShellCompDirectiveNoSpace, err
}

// List files and directories under the specified path.
// Returns a channel such that we can list multiple paths in parallel.
func fetchCompletions(
	c *completer,
	dirPath string,
) ([]string, error) {
	entries, err := c.filer.ReadDir(c.ctx, dirPath)
	if err != nil {
		if _, ok := err.(filer.NoSuchDirectoryError); ok {
			return nil, nil
		} else {
			return nil, err
		}
	}

	trailingSeparator := "/"
	joinFunc := path.Join

	// Use filepath functions if we are in a local path.
	if c.isLocalPath {
		joinFunc = filepath.Join
		trailingSeparator = string(filepath.Separator)
	}

	completions := []string{}
	for _, entry := range entries {
		if c.onlyDirs && !entry.IsDir() {
			continue
		}

		completion := joinFunc(dirPath, entry.Name())

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

	return completions, nil
}

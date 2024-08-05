package completer

import (
	"context"
	"path"

	"github.com/databricks/cli/libs/filer"
	"github.com/spf13/cobra"
)

type completer struct {
	ctx      context.Context
	filer    filer.Filer
	onlyDirs bool
}

// General completer that takes a Filer to complete remote paths when TAB-ing through a path.
func New(ctx context.Context, filer filer.Filer, onlyDirs bool) *completer {
	return &completer{ctx: ctx, filer: filer, onlyDirs: onlyDirs}
}

func (c *completer) CompletePath(p string) ([]string, string, cobra.ShellCompDirective) {
	// If the user is TAB-ing their way through a path, the path in `toComplete`
	// is valid and we should list nested directories.
	// If the path in `toComplete` is incomplete, however,
	// then we should list adjacent directories.
	dirPath := p
	completions := fetchCompletions(c, dirPath)
	if completions == nil {
		dirPath = path.Dir(p)
		completions = fetchCompletions(c, dirPath)
	}

	return completions, dirPath, cobra.ShellCompDirectiveNoSpace
}

// List files and directories under the specified path.
// Returns a channel such that we can list multiple paths in parallel.
func fetchCompletions(
	c *completer,
	path string,
) []string {
	entries, err := c.filer.ReadDir(c.ctx, path)
	if err != nil {
		return nil
	}

	completions := []string{}
	for _, entry := range entries {
		if c.onlyDirs && !entry.IsDir() {
			continue
		}

		completions = append(completions, entry.Name())
	}

	return completions
}

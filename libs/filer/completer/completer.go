package completer

import (
	"context"
	"path"
	"path/filepath"
	"strings"

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
	separators := "/"
	joinFunc := path.Join

	// Use filepath functions if we are in a local path.
	if c.isLocalPath {
		joinFunc = filepath.Join
		trailingSeparator = string(filepath.Separator)
		separators += trailingSeparator
	}

	// Shells only offer completions that start with the word being completed,
	// so keep the directory part of p as typed. Joining it with the entry name
	// would clean it and turn "./sr" into "src", which the shell then discards.
	typedDir := p[:strings.LastIndexAny(p, separators)+1]

	// If the user is TAB-ing their way through a path and the
	// path ends in a trailing slash, we should list nested directories.
	// If the path is incomplete, however, then we should list adjacent
	// directories.
	dirPath := p
	if !strings.HasSuffix(p, trailingSeparator) {
		dirPath = path.Dir(p)
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

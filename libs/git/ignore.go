package git

import (
	"errors"
	"io/fs"
	"strings"
	"time"

	"github.com/databricks/cli/libs/vfs"
	ignore "github.com/sabhiram/go-gitignore"
)

// ignoreRules implements the interface for a gitignore-like file.
// It is backed by implementations for a file at a specific path (it may not exist)
// and an implementation for a set of in-memory ignore patterns.
type ignoreRules interface {
	MatchesPath(path string) (bool, error)

	// Taint forces checking if the underlying file needs to be reloaded.
	// It checks the mtime of the file to see if has been modified after loading.
	Taint()
}

// ignoreFile represents a gitignore file backed by a path.
// If the path doesn't exist (yet), it is treated as an empty file.
type ignoreFile struct {
	root vfs.Path
	path string

	// Signal a reload of this file.
	// Set this to call [os.Stat] and a potential reload
	// of the file's contents next time it is used.
	checkForReload bool

	// Modified time for this file.
	modTime time.Time

	// Ignore patterns contained in this file.
	patterns *ignore.GitIgnore
}

func newIgnoreFile(root vfs.Path, path string) ignoreRules {
	return &ignoreFile{
		root:           root,
		path:           path,
		checkForReload: true,
	}
}

func (f *ignoreFile) MatchesPath(path string) (bool, error) {
	if f.checkForReload {
		err := f.load()
		if err != nil {
			return false, err
		}
		// Don't check again in next call.
		f.checkForReload = false
	}

	// A file that doesn't exist doesn't have ignore patterns.
	if f.patterns == nil {
		return false, nil
	}

	return f.patterns.MatchesPath(path), nil
}

func (f *ignoreFile) Taint() {
	f.checkForReload = true
}

func (f *ignoreFile) load() error {
	// The file must be stat-able.
	// If it doesn't exist, treat it as an empty file.
	stat, err := fs.Stat(f.root, f.path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	}

	// If the underlying file has not been modified
	// it does not need to be reloaded.
	if stat.ModTime() == f.modTime {
		return nil
	}

	f.modTime = stat.ModTime()
	f.patterns, err = f.loadGitignore()
	if err != nil {
		return err
	}

	return nil
}

func (f *ignoreFile) loadGitignore() (*ignore.GitIgnore, error) {
	data, err := fs.ReadFile(f.root, f.path)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(data), "\n")
	return ignore.CompileIgnoreLines(lines...), nil
}

// stringIgnoreRules implements the [ignoreRules] interface
// for a set of in-memory ignore patterns.
type stringIgnoreRules struct {
	patterns *ignore.GitIgnore
}

func newStringIgnoreRules(patterns []string) ignoreRules {
	return &stringIgnoreRules{
		patterns: ignore.CompileIgnoreLines(patterns...),
	}
}

func (r *stringIgnoreRules) MatchesPath(path string) (bool, error) {
	return r.patterns.MatchesPath(path), nil
}

func (r *stringIgnoreRules) Taint() {
	// Tainting in-memory ignore patterns is a nop.
}

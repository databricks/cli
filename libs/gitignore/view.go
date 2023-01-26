package gitignore

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/bricks/folders"
	ignore "github.com/sabhiram/go-gitignore"
)

// View represents a view on a directory tree that takes into account
// all applicable .gitignore files. The directory tree does NOT need
// to be the repository root.
//
// For example: with a repository root at "myrepo", a view can be
// anchored at "myrepo/someproject" and still respect the ignore
// rules defined at "myrepo/.gitignore".
//
// We use this functionality to synchronize files from a path nested
// in a repository while respecting the repository's ignore rules.
type View struct {
	// rootPath is the absolute path to the Git repository root.
	rootPath string

	// targetPath is the relative path within the repository we care about.
	// For example: "." or "a/b".
	targetPath string

	// files contains a list of ignore patterns indexed by the
	// path prefix relative to the repository root.
	files map[string][]*ignore.GitIgnore
}

// gitignorePaths compiles the list of paths relative to the repository root where
// `.gitignore` files could be found. For example: []string{".", "foo", "foo/bar"}.
func (v *View) gitignorePaths() []string {
	paths := []string{
		".",
	}
	for _, path := range strings.Split(v.targetPath, string(os.PathSeparator)) {
		path = filepath.Join(paths[len(paths)-1], path)
		// May be identical if targetPath == ".".
		if path != paths[len(paths)-1] {
			paths = append(paths, path)
		}
	}
	return paths
}

func (v *View) loadIgnoreFile(relativeIgnoreFilePath string, relativeTo string) error {
	path := filepath.Join(v.rootPath, relativeIgnoreFilePath)

	// The file must be stat-able and not a directory.
	stat, err := os.Stat(path)
	if err != nil || stat.IsDir() {
		return nil
	}

	ignore, err := ignore.CompileIgnoreFile(path)
	if err != nil {
		return err
	}

	relativeTo = filepath.ToSlash(relativeTo)
	v.files[relativeTo] = append(v.files[relativeTo], ignore)
	return nil
}

func (v *View) reload() error {
	v.files = make(map[string][]*ignore.GitIgnore)

	// Load repository-wide excludes file.
	err := v.loadIgnoreFile(filepath.Join(".git", "info", "excludes"), ".")
	if err != nil {
		return err
	}

	// Load `.gitignore` files.
	for _, path := range v.gitignorePaths() {
		err := v.loadIgnoreFile(filepath.Join(path, ".gitignore"), path)
		if err != nil {
			return err
		}
	}

	// Recurse into directories under target path.
	err = filepath.WalkDir(
		filepath.Join(v.rootPath, v.targetPath),
		func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				// If reading the target path fails bubble up the error.
				if d == nil {
					return err
				}
				// Ignore failure to read paths nested under the target path.
				return filepath.SkipDir
			}

			// Get path relative to root path.
			pathRelativeToRoot, err := filepath.Rel(v.rootPath, path)
			if err != nil {
				return err
			}

			// Get path relative to target path.
			pathRelativeToTarget, err := filepath.Rel(v.targetPath, pathRelativeToRoot)
			if err != nil {
				return err
			}

			// Ignore target path itself (its .gitignore was already loaded)
			if pathRelativeToTarget == "." {
				return nil
			}

			// Check if directory is ignored before recursing into it.
			if d.IsDir() && v.Ignore(pathRelativeToTarget) {
				return filepath.SkipDir
			}

			// Load .gitignore if we find one.
			if d.Name() == ".gitignore" {
				err := v.loadIgnoreFile(pathRelativeToRoot, filepath.Dir(pathRelativeToRoot))
				if err != nil {
					return err
				}
			}

			return nil
		})
	if err != nil {
		return fmt.Errorf("unable to walk directory: %w", err)
	}

	return nil
}

// Ignore computes whether to ignore the specified path.
// The specified path is relative to the view's target path.
func (v *View) Ignore(path string) bool {
	if filepath.IsAbs(path) {
		panic("abs given")
	}

	// Make path relative to repository root.
	relPath := filepath.Join(v.targetPath, path)

	// Walk over path prefixes to check applicable gitignore files.
	parts := strings.Split(relPath, string(os.PathSeparator))
	for i := range parts {
		prefix := strings.Join(parts[:i], "/")
		if prefix == "" {
			prefix = "."
		}
		suffix := strings.Join(parts[i:], "/")

		// For this prefix (e.g. ".", or "dir1/dir2") we check if the
		// suffix is matched in the respective ignore files.
		fs, ok := v.files[prefix]
		if ok {
			for _, f := range fs {
				if f.MatchesPath(suffix) {
					return true
				}
			}
		}
	}

	return false
}

func NewView(path string) (*View, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	root, err := folders.FindDirWithLeaf(path, ".git")
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		// Cannot find `.git` directory. Use current directory.
		root = path
	}

	// Target must be relative to the repository root.
	target, err := filepath.Rel(root, path)
	if err != nil {
		return nil, err
	}

	view := &View{
		rootPath:   root,
		targetPath: target,
	}

	err = view.reload()
	if err != nil {
		return nil, err
	}

	return view, nil
}

package git

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/databricks/bricks/folders"
	ignore "github.com/sabhiram/go-gitignore"
)

const gitIgnoreFileName = ".gitignore"

// Repository represents a Git repository or a directory
// that could later be initialized as Git repository.
type Repository struct {
	// real indicates if this is a real repository or a non-Git
	// directory where we process .gitignore files.
	real bool

	// rootPath is the absolute path to the repository root.
	rootPath string

	// ignore contains a list of ignore patterns indexed by the
	// path prefix relative to the repository root.
	//
	// Example prefixes: ".", "foo/bar"
	//
	// Note: prefixes use the forward slash instead of the
	// OS-specific path separator. This matches Git convention.
	ignore map[string][]*ignore.GitIgnore
}

func (r *Repository) includeIgnoreFile(relativeIgnoreFilePath string, relativeTo string) error {
	absPath := filepath.Join(r.rootPath, relativeIgnoreFilePath)

	// The file must be stat-able and not a directory.
	// If it doesn't exist or is a directory, do nothing.
	stat, err := os.Stat(absPath)
	if err != nil || stat.IsDir() {
		return nil
	}

	ignore, err := ignore.CompileIgnoreFile(absPath)
	if err != nil {
		return err
	}

	relativeTo = path.Clean(filepath.ToSlash(relativeTo))
	r.ignore[relativeTo] = append(r.ignore[relativeTo], ignore)
	return nil
}

// Include ignore files in directories that are parent to `relPath`.
//
// If equal to "foo/bar" this loads ignore files
// located at the repository root and in the directory "foo".
//
// If equal to "." this function does nothing.
func (r *Repository) includeIgnoreFilesUpToPath(relPath string) error {
	// Accumulate list of directories to load ignore file from.
	paths := []string{
		".",
	}
	for _, path := range strings.Split(relPath, string(os.PathSeparator)) {
		path = filepath.Join(paths[len(paths)-1], path)

		// May be identical if relPath == ".".
		if path != relPath {
			paths = append(paths, path)
		}
	}

	// Load ignore files.
	for _, path := range paths {
		err := r.includeIgnoreFile(filepath.Join(path, gitIgnoreFileName), path)
		if err != nil {
			return err
		}
	}

	return nil
}

// Include ignore files in directories that are equal to or nested under `relPath`.
func (r *Repository) includeIgnoreFilesUnderPath(relPath string) error {
	absPath := filepath.Join(r.rootPath, relPath)
	err := filepath.WalkDir(absPath, r.includeIgnoreFilesWalkDirFn)
	if err != nil {
		return fmt.Errorf("unable to walk directory: %w", err)
	}
	return nil
}

// includeIgnoreFilesWalkDirFn is called from [filepath.WalkDir] in includeIgnoreFilesUnderPath.
func (r *Repository) includeIgnoreFilesWalkDirFn(absPath string, d fs.DirEntry, err error) error {
	if err != nil {
		// If reading the target path fails bubble up the error.
		if d == nil {
			return err
		}
		// Ignore failure to read paths nested under the target path.
		return filepath.SkipDir
	}

	// Get path relative to root path.
	pathRelativeToRoot, err := filepath.Rel(r.rootPath, absPath)
	if err != nil {
		return err
	}

	// Check if directory is ignored before recursing into it.
	if d.IsDir() && r.Ignore(pathRelativeToRoot) {
		return filepath.SkipDir
	}

	// Load .gitignore if we find one.
	if d.Name() == gitIgnoreFileName {
		err := r.includeIgnoreFile(pathRelativeToRoot, filepath.Dir(pathRelativeToRoot))
		if err != nil {
			return err
		}
	}

	return nil
}

// Include ignore files relevant for files nested under `relPath`.
func (r *Repository) includeIgnoreFilesForPath(relPath string) error {
	err := r.includeIgnoreFilesUpToPath(relPath)
	if err != nil {
		return err
	}
	return r.includeIgnoreFilesUnderPath(relPath)
}

// Ignore computes whether to ignore the specified path.
// The specified path is relative to the repository root path.
func (r *Repository) Ignore(relPath string) bool {
	// Walk over path prefixes to check applicable gitignore files.
	parts := strings.Split(relPath, string(os.PathSeparator))
	for i := range parts {
		prefix := path.Clean(strings.Join(parts[:i], "/"))
		suffix := path.Clean(strings.Join(parts[i:], "/"))

		// For this prefix (e.g. ".", or "dir1/dir2") we check if the
		// suffix is matched in the respective ignore files.
		fs, ok := r.ignore[prefix]
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

func NewRepository(path string) (*Repository, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	real := true
	rootPath, err := folders.FindDirWithLeaf(path, ".git")
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		// Cannot find `.git` directory.
		// Treat the specified path as a potential repository root.
		real = false
		rootPath = path
	}

	repo := &Repository{
		real:     real,
		rootPath: rootPath,
		ignore:   make(map[string][]*ignore.GitIgnore),
	}

	// Always ignore ".git" directory.
	repo.ignore["."] = append(repo.ignore["."], ignore.CompileIgnoreLines(".git"))

	// Load repository-wide excludes file.
	err = repo.includeIgnoreFile(filepath.Join(".git", "info", "excludes"), ".")
	if err != nil {
		return nil, err
	}

	return repo, nil
}

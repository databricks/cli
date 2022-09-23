package git

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	ignore "github.com/sabhiram/go-gitignore"
)

type File struct {
	fs.DirEntry
	Absolute, Relative string
}

func (f File) Modified() (ts time.Time) {
	info, err := f.Info()
	if err != nil {
		// return default time, beginning of epoch
		return ts
	}
	return info.ModTime()
}

// FileSet facilitates fast recursive tracked file listing
// with respect to patterns defined in `.gitignore` file
//
// root:   Root of the git repository
// ignore: List of patterns defined in `.gitignore`.
//  	   We do not sync files that match this pattern
type FileSet struct {
	root   string
	ignore *ignore.GitIgnore
	// if include is not nil, it will override ignore
	include *ignore.GitIgnore
}

// GetFileSet retrieves FileSet from Git repository checkout root
// or panics if no root is detected.
func GetFileSet() (FileSet, error) {
	root, err := Root()
	return NewFileSet(root, nil), err
}

// Retuns FileSet for the repository located at `root`
func NewFileSet(root string, syncInclude *[]string) FileSet {
	if syncInclude != nil {
		return FileSet{
			root: root,
			include: ignore.CompileIgnoreLines(*syncInclude...),
		}
	}
	lines := []string{".git", ".bricks"}
	rawIgnore, err := os.ReadFile(fmt.Sprintf("%s/.gitignore", root))
	if err == nil {
		// add entries from .gitignore if the file exists (did read correctly)
		for _, line := range strings.Split(string(rawIgnore), "\n") {
			// underlying library doesn't behave well with Rule 5 of .gitignore,
			// hence this workaround
			lines = append(lines, strings.Trim(line, "/"))
		}
	}
	return FileSet{
		root:   root,
		ignore: ignore.CompileIgnoreLines(lines...),
	}
}

// Return root for fileset.
func (w *FileSet) Root() string {
	return w.root
}

// Return all tracked files for Repo
func (w *FileSet) All() ([]File, error) {
	return w.RecursiveListFiles(w.root)
}

// Recursively traverses dir in a depth first manner and returns a list of all files
// that are being tracked in the FileSet (ie not being ignored for matching one of the
// patterns in w.ignore)
func (w *FileSet) RecursiveListFiles(dir string) (fileList []File, err error) {
	queue, err := readDir(dir, w.root)
	if err != nil {
		return nil, err
	}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		// If w.include is empty and the current file matches one of the
		// patterns in w.ignore then we do not list the file
		if w.include == nil && w.ignore.MatchesPath(current.Relative) {
			continue
		}

		// if w.include is not empty and current file does not match any
		// patterns in w.include then we do not list the file
		if w.include != nil && !w.include.MatchesPath(current.Relative) {
			continue
		}

		// we list the file if one of the following holds true
		// 1. w.include is empty and current file does not match any of the
		// patterns in w.ignore
		// 2. w.include is not empty and current file matches any one of the
		// patters in w.include
		if !current.IsDir() {
			fileList = append(fileList, current)
			continue
		}
		children, err := readDir(current.Absolute, w.root)
		if err != nil {
			return nil, err
		}
		queue = append(queue, children...)
	}
	return fileList, nil
}

func readDir(dir, root string) (queue []File, err error) {
	f, err := os.Open(dir)
	if err != nil {
		return
	}
	defer f.Close()
	dirs, err := f.ReadDir(-1)
	if err != nil {
		return
	}
	for _, v := range dirs {
		absolute := filepath.Join(dir, v.Name())
		relative, err := filepath.Rel(root, absolute)
		if err != nil {
			return nil, err
		}
		queue = append(queue, File{v, absolute, relative})
	}
	return
}

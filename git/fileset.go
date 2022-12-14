package git

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	ignore "github.com/sabhiram/go-gitignore"
	"golang.org/x/exp/slices"
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
//
//	We do not sync files that match this pattern
type FileSet struct {
	root   string
	ignore *ignore.GitIgnore
}

// Retuns FileSet for the git repo located at `root`
func NewFileSet(root string) *FileSet {
	syncIgnoreLines := append(getGitIgnoreLines(root), getSyncIgnoreLines()...)
	return &FileSet{
		root:   root,
		ignore: ignore.CompileIgnoreLines(syncIgnoreLines...),
	}
}

func getGitIgnoreLines(root string) []string {
	gitIgnoreLines := []string{}
	gitIgnorePath := fmt.Sprintf("%s/.gitignore", root)
	rawIgnore, err := os.ReadFile(gitIgnorePath)

	if err == nil {
		// add entries from .gitignore if the file exists (did read correctly)
		for _, line := range strings.Split(string(rawIgnore), "\n") {
			// underlying library doesn't behave well with Rule 5 of .gitignore,
			// hence this workaround
			gitIgnoreLines = append(gitIgnoreLines, strings.Trim(line, "/"))
		}
	}
	return gitIgnoreLines
}

// This contains additional files/dirs to be ignored while syncing that are
// not mentioned in .gitignore
func getSyncIgnoreLines() []string {
	return []string{".git"}
}

// Only call this function for a bricks project root
// since it will create a .gitignore file if missing
func (w *FileSet) EnsureValidGitIgnoreExists() error {
	gitIgnoreLines := getGitIgnoreLines(w.root)
	gitIgnorePath := fmt.Sprintf("%s/.gitignore", w.root)
	if slices.Contains(gitIgnoreLines, ".databricks") {
		return nil
	}
	f, err := os.OpenFile(gitIgnorePath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString("\n.databricks\n")
	if err != nil {
		return err
	}
	gitIgnoreLines = append(gitIgnoreLines, ".databricks")
	w.ignore = ignore.CompileIgnoreLines(append(gitIgnoreLines, getSyncIgnoreLines()...)...)
	return nil
}

// Return root for fileset.
func (w *FileSet) Root() string {
	return w.root
}

// Return all tracked files for Repo
func (w *FileSet) All() ([]File, error) {
	return w.RecursiveListFiles(w.root)
}

func (w *FileSet) IsGitIgnored(pattern string) bool {
	return w.ignore.MatchesPath(pattern)
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
		if w.ignore.MatchesPath(current.Relative) {
			continue
		}
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

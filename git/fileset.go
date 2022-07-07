package git

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path"
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

// FileSet facilitates fast recursive file listing with
// respect to patterns defined in `.gitignore` file
type FileSet struct {
	root   string
	ignore *ignore.GitIgnore
}

// MustGetFileSet retrieves FileSet from Git repository checkout root
// or panics if no root is detected.
func MustGetFileSet() FileSet {
	root, err := Root()
	if err != nil {
		panic(err)
	}
	return New(root)
}

func New(root string) FileSet {
	lines := []string{".git"}
	rawIgnore, err := ioutil.ReadFile(fmt.Sprintf("%s/.gitignore", root))
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

func (w *FileSet) All() ([]File, error) {
	return w.RecursiveChildren(w.root)
}

func (w *FileSet) RecursiveChildren(dir string) (found []File, err error) {
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
			found = append(found, current)
			continue
		}
		children, err := readDir(current.Absolute, w.root)
		if err != nil {
			return nil, err
		}
		queue = append(queue, children...)
	}
	return found, nil
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
		absolute := path.Join(dir, v.Name())
		relative := strings.TrimLeft(strings.Replace(absolute, root, "", 1), "/")
		queue = append(queue, File{v, absolute, relative})
	}
	return
}

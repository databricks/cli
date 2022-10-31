package fileset

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

type FileSet []File

func (fi FileSet) Root() string {
	return strings.TrimSuffix(
		strings.ReplaceAll(fi[0].Absolute, fi[0].Relative, ""),
		"/")
}

func (fi FileSet) FirstMatch(pathRegex, needleRegex string) *File {
	path := regexp.MustCompile(pathRegex)
	needle := regexp.MustCompile(needleRegex)
	for _, v := range fi {
		if !path.MatchString(v.Absolute) {
			continue
		}
		if v.Match(needle) {
			return &v
		}
	}
	return nil
}

func (fi FileSet) FindAll(pathRegex, needleRegex string) (map[File][]string, error) {
	path := regexp.MustCompile(pathRegex)
	needle := regexp.MustCompile(needleRegex)
	all := map[File][]string{}
	for _, v := range fi {
		if !path.MatchString(v.Absolute) {
			continue
		}
		vall, err := v.FindAll(needle)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", v.Relative, err)
		}
		all[v] = vall
	}
	return all, nil
}

func (fi FileSet) Exists(pathRegex, needleRegex string) bool {
	m := fi.FirstMatch(pathRegex, needleRegex)
	return m != nil
}

func RecursiveChildren(dir, root string) (found FileSet, err error) {
	// TODO: add options to skip, like current.Name() == "vendor"
	queue, err := ReadDir(dir, root)
	if err != nil {
		return nil, err
	}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if !current.IsDir() {
			current.Relative = strings.ReplaceAll(current.Absolute, dir+"/", "")
			found = append(found, current)
			continue
		}
		children, err := ReadDir(current.Absolute, root)
		if err != nil {
			return nil, err
		}
		queue = append(queue, children...)
	}
	return found, nil
}

func ReadDir(dir, root string) (queue []File, err error) {
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
		relative, err := filepath.Rel(root, absolute)
		if err != nil {
			return nil, err
		}
		queue = append(queue, File{v, absolute, relative})
	}
	return
}

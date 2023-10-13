package fileset

import (
	"path/filepath"
)

func NewGlobSet(root string, includes []string) (*FileSet, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}

	for k := range includes {
		includes[k] = filepath.ToSlash(filepath.Clean(includes[k]))
	}

	fs := &FileSet{
		absRoot,
		newIncluder(includes),
	}
	return fs, nil
}

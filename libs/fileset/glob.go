package fileset

import (
	"path/filepath"
)

type GlobSet struct {
	fs       *FileSet
	root     string
	patterns []string
}

func NewGlobSet(root string, includes []string) (*GlobSet, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}

	for k := range includes {
		includes[k] = filepath.Clean(filepath.FromSlash(includes[k]))
	}

	fs := &FileSet{
		absRoot,
		newIncluder(includes),
	}
	return &GlobSet{fs, absRoot, includes}, nil
}

// Return all files which matches defined glob patterns
func (s *GlobSet) All() ([]File, error) {
	return s.fs.All()
}

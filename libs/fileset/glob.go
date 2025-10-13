package fileset

import (
	"path"

	"github.com/databricks/cli/libs/vfs"
)

func NewGlobSet(root vfs.Path, includes []string) (*FileSet, error) {
	if len(includes) == 0 {
		return Empty(), nil
	}
	for k := range includes {
		includes[k] = path.Clean(includes[k])
	}

	fs := New(root)
	fs.SetIgnorer(newIncluder(includes))
	return fs, nil
}

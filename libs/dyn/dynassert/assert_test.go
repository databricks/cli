package dynassert

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestThatThisTestPackageIsUsed(t *testing.T) {
	base := ".."
	var files []string
	err := fs.WalkDir(os.DirFS(base), ".", func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			// Filter this directory.
			if filepath.Base(path) == "dynassert" {
				return fs.SkipDir
			}
		}
		if ok, _ := filepath.Match("*_test.go", d.Name()); ok {
			files = append(files, filepath.Join(base, path))
		}
		return nil
	})
	require.NoError(t, err)

	// Confirm that none of the test files under `libs/dyn` import the
	// `testify/assert` package and instead import this package for asserts.
	fset := token.NewFileSet()
	for _, file := range files {
		f, err := parser.ParseFile(fset, file, nil, parser.ParseComments)
		require.NoError(t, err)

		for _, imp := range f.Imports {
			if strings.Contains(imp.Path.Value, `github.com/stretchr/testify/assert`) {
				t.Errorf("File %s should not import github.com/stretchr/testify/assert", file)
			}
		}
	}
}

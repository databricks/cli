package testutil_test

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNoTestingImport checks that no file in the package imports the testing package.
// All exported functions must use the TestingT interface instead of *testing.T.
func TestNoTestingImport(t *testing.T) {
	// Parse the package
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", nil, parser.AllErrors)
	require.NoError(t, err)

	// Iterate through the files in the package
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			// Skip test files
			if strings.HasSuffix(fset.Position(file.Pos()).Filename, "_test.go") {
				continue
			}
			// Check the imports of each file
			for _, imp := range file.Imports {
				if imp.Path.Value == `"testing"` {
					assert.Fail(t, "File imports the testing package", "File %s imports the testing package", fset.Position(file.Pos()).Filename)
				}
			}
		}
	}
}

package integration

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	"golang.org/x/exp/maps"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type packageInfo struct {
	Name  string
	Files []string
}

func enumeratePackages(t *testing.T) map[string]packageInfo {
	pkgmap := make(map[string]packageInfo)
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip files.
		if !info.IsDir() {
			return nil
		}

		// Skip the root directory and the "internal" directory.
		if path == "." || strings.HasPrefix(path, "internal") {
			return nil
		}

		fset := token.NewFileSet()
		pkgs, err := parser.ParseDir(fset, path, nil, parser.ParseComments)
		require.NoError(t, err)
		if len(pkgs) == 0 {
			return nil
		}

		// Expect one package per directory.
		require.Len(t, pkgs, 1, "Directory %s contains more than one package", path)
		v := maps.Values(pkgs)[0]

		// Record the package.
		pkgmap[path] = packageInfo{
			Name:  v.Name,
			Files: maps.Keys(v.Files),
		}
		return nil
	})
	require.NoError(t, err)
	return pkgmap
}

// TestEnforcePackageNames checks that all integration test package names use the "_test" suffix.
// We enforce this package name to avoid package name aliasing.
func TestEnforcePackageNames(t *testing.T) {
	pkgmap := enumeratePackages(t)
	for _, pkg := range pkgmap {
		assert.True(t, strings.HasSuffix(pkg.Name, "_test"), "Package name %s does not end with _test", pkg.Name)
	}
}

var mainTestTemplate = template.Must(template.New("main_test").Parse(
	`package {{.Name}}

import (
	"testing"

	"github.com/databricks/cli/integration/internal"
)

// TestMain is the entrypoint executed by the test runner.
// See [internal.Main] for prerequisites for running integration tests.
func TestMain(m *testing.M) {
	internal.Main(m)
}
`))

func TestEnforceMainTest(t *testing.T) {
	pkgmap := enumeratePackages(t)
	for dir, pkg := range pkgmap {
		found := false
		for _, file := range pkg.Files {
			if filepath.Base(file) == "main_test.go" {
				found = true
				break
			}
		}

		// Expect a "main_test.go" file in each package.
		assert.True(t, found, "Directory %s does not contain a main_test.go file", dir)
	}
}

func TestWriteMainTest(t *testing.T) {
	t.Skip("Uncomment to write main_test.go files")

	pkgmap := enumeratePackages(t)
	for dir, pkg := range pkgmap {
		// Write a "main_test.go" file to the package.
		// This file is required to run the integration tests.
		f, err := os.Create(filepath.Join(dir, "main_test.go"))
		require.NoError(t, err)
		defer f.Close()
		err = mainTestTemplate.Execute(f, pkg)
		require.NoError(t, err)
	}
}

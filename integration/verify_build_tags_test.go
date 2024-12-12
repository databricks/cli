package integration

import (
	"go/scanner"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func verifyIntegrationTest(t *testing.T, path string) {
	var s scanner.Scanner
	fset := token.NewFileSet()
	src, err := os.ReadFile(path)
	require.NoError(t, err)
	file := fset.AddFile(path, fset.Base(), len(src))
	s.Init(file, src, nil, scanner.ScanComments)

	var buildTag string
	var packageName string

	var tok token.Token
	var lit string

	// Keep scanning until we find the package name and build tag.
	for tok != token.EOF && (buildTag == "" || packageName == "") {
		_, tok, lit = s.Scan()
		switch tok {
		case token.PACKAGE:
			_, tok, lit = s.Scan()
			if tok == token.IDENT {
				packageName = lit
			}
		case token.COMMENT:
			if strings.HasPrefix(lit, "//go:build ") {
				buildTag = strings.TrimPrefix(lit, "//go:build ")
			}
		case token.EOF:
			break
		}
	}

	// Verify that the build tag is present.
	assert.Equal(t, "integration", buildTag, "File %s does not specify the 'integration' build tag", path)

	// Verify that the package name matches the expected format.
	expected := filepath.Base(filepath.Dir(path)) + "_integration"
	assert.Equal(t, expected, packageName, "File %s package name '%s' does not match directory name '%s'", path, packageName, expected)
}

// TestVerifyBuildTags checks that all test files in the integration package specify the "integration" build tag
// and that the package name matches the basename of the containing directory with "_integration" appended.
//
// We enforce this package name to avoid package name aliasing.
func TestVerifyBuildTags(t *testing.T) {
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories.
		if info.IsDir() {
			return nil
		}

		// Skip this file.
		if path == "verify_build_tags_test.go" {
			return nil
		}

		// Skip files that are not test files.
		if !strings.HasSuffix(info.Name(), "_test.go") {
			return nil
		}

		verifyIntegrationTest(t, path)
		return nil
	})

	require.NoError(t, err)
}

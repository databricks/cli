package integration

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVerifyBuildTags checks that all test files in the integration package specify the "integration" build tag.
// This ensures that `go test ./...` doesn't run integration tests.
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

		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
		require.NoError(t, err)

		// Check for the "integration" build tag in the file comments.
		found := false
		for _, comment := range file.Comments {
			if strings.Contains(comment.Text(), "+build integration") {
				found = true
				break
			}
		}

		assert.True(t, found, "File %s does not specify the 'integration' build tag", path)
		return nil
	})

	require.NoError(t, err)
}

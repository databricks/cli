package integration

import (
	"bufio"
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

		f, err := os.Open(path)
		require.NoError(t, err)
		defer f.Close()

		// Read the first line
		scanner := bufio.NewScanner(f)
		scanner.Scan()
		assert.Equal(t, "//go:build integration", scanner.Text(), "File %s does not specify the 'integration' build tag", path)
		return nil
	})

	require.NoError(t, err)
}

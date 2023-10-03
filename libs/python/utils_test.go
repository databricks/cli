package python

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFindFilesWithSuffixInPath(t *testing.T) {
	dir, err := os.Getwd()
	require.NoError(t, err)

	files := FindFilesWithSuffixInPath(dir, "test.go")

	matches, err := filepath.Glob(filepath.Join(dir, "*test.go"))
	require.NoError(t, err)

	require.ElementsMatch(t, files, matches)
}

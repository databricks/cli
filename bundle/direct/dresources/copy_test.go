package dresources

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/testdiff"
	"github.com/stretchr/testify/require"
)

// TestFieldCopies checks every Copier of this package and compares what they copy with
// testdata/field_copies.txt. An SDK bump that adds a field changes the file, so the bump
// shows which requests and states pick the field up; `./task generate-direct` updates it.
func TestFieldCopies(t *testing.T) {
	require.NotEmpty(t, fieldCopiers)

	descriptions := make([]string, 0, len(fieldCopiers))
	for _, copier := range fieldCopiers {
		description, err := copier.Describe()
		require.NoError(t, err)
		descriptions = append(descriptions, description)
	}
	slices.Sort(descriptions)
	got := strings.Join(descriptions, "\n")

	path := filepath.Join("testdata", "field_copies.txt")
	if testdiff.OverwriteMode {
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(got), 0o644))
		return
	}

	want, err := os.ReadFile(path)
	require.NoError(t, err)
	testdiff.AssertEqualTexts(t, path, "actual", testdiff.NormalizeNewlines(string(want)), got)
}

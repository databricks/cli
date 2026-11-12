package aitools

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBinaryPathFor(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	absolute := filepath.Join(t.TempDir(), "bin", "databricks")
	foundPath := filepath.Join(t.TempDir(), "databricks")
	relativeOnPath := filepath.Join("bin", "databricks")

	tests := []struct {
		name string
		arg0 string
		// onPath is what the PATH lookup finds, or "" when the name is not on PATH.
		onPath string
		// wantLookup is the name the PATH lookup receives, or "" when the lookup must not run.
		wantLookup string
		wantPath   string
		wantAbs    bool
	}{
		{
			// The invoked path is kept as is, so a Homebrew symlink is not replaced by its target.
			name:     "absolute arg0 is kept",
			arg0:     absolute,
			wantPath: absolute,
			wantAbs:  true,
		},
		{
			name:     "relative arg0 with a directory is made absolute, not looked up",
			arg0:     filepath.Join("bin", "databricks"),
			onPath:   foundPath,
			wantPath: mustAbs(t, filepath.Join("bin", "databricks")),
			wantAbs:  true,
		},
		{
			name:     "dot-relative arg0 with a directory is made absolute",
			arg0:     "." + string(filepath.Separator) + "databricks",
			onPath:   foundPath,
			wantPath: wd + string(filepath.Separator) + "databricks",
			wantAbs:  true,
		},
		{
			// The OS reads "link/.." through the link's target, so the path must not be cleaned before the symlink is followed.
			name:     "relative arg0 through a symlink keeps its dot-dot",
			arg0:     "link" + string(filepath.Separator) + ".." + string(filepath.Separator) + "databricks",
			onPath:   foundPath,
			wantPath: wd + string(filepath.Separator) + "link" + string(filepath.Separator) + ".." + string(filepath.Separator) + "databricks",
			wantAbs:  true,
		},
		{
			name:       "bare name is looked up on PATH",
			arg0:       "databricks",
			onPath:     foundPath,
			wantLookup: "databricks",
			wantPath:   foundPath,
			wantAbs:    true,
		},
		{
			name:       "bare name not on PATH falls back to the bare name",
			arg0:       "databricks",
			onPath:     "",
			wantLookup: "databricks",
			wantPath:   "databricks",
		},
		{
			// Only a databricks argv[0] may fall back to the bare name; another name is kept, so the caller refuses it.
			name:       "other bare name not on PATH is kept",
			arg0:       "dbx",
			onPath:     "",
			wantLookup: "dbx",
			wantPath:   "dbx",
		},
		{
			name:       "relative PATH entry falls back to the bare name",
			arg0:       "databricks",
			onPath:     relativeOnPath,
			wantLookup: "databricks",
			wantPath:   "databricks",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lookup := &pathLookup{path: tt.onPath}
			path, abs := binaryPathFor(tt.arg0, lookup.lookPath)
			assert.Equal(t, tt.wantPath, path)
			assert.Equal(t, tt.wantAbs, abs)
			assert.Equal(t, tt.wantLookup, lookup.asked)
		})
	}
}

func mustAbs(t *testing.T, path string) string {
	t.Helper()
	abs, err := filepath.Abs(path)
	require.NoError(t, err)
	return abs
}

// pathLookup stands in for the PATH lookup. It finds the name at path, or reports it not found when path is "". It
// records the name it was asked for, so a case can check whether the lookup ran at all.
type pathLookup struct {
	path  string
	asked string
}

func (l *pathLookup) lookPath(name string) (string, error) {
	l.asked = name
	if l.path == "" {
		return "", errors.New("not found")
	}
	return l.path, nil
}

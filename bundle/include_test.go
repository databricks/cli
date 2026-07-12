package bundle

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsFileIncluded(t *testing.T) {
	// Lay out a bundle root with a couple of resource files on disk, because
	// IsFileIncluded resolves the include patterns with filepath.Glob against
	// the files that actually exist.
	root := t.TempDir()
	included := filepath.Join(root, "resources", "my.job.yml")
	nested := filepath.Join(root, "resources", "sub", "other.job.yml")
	for _, f := range []string{included, nested} {
		require.NoError(t, os.MkdirAll(filepath.Dir(f), 0o755))
		require.NoError(t, os.WriteFile(f, []byte("{}"), 0o644))
	}

	tests := []struct {
		name     string
		patterns []string
		filename string
		want     bool
	}{
		{
			name:     "matched by glob",
			patterns: []string{"resources/*.yml"},
			filename: included,
			want:     true,
		},
		{
			name:     "not matched, empty include",
			patterns: nil,
			filename: included,
			want:     false,
		},
		{
			name:     "glob is not recursive",
			patterns: []string{"resources/*.yml"},
			filename: nested,
			want:     false,
		},
		{
			name:     "matched by exact path",
			patterns: []string{"resources/my.job.yml"},
			filename: included,
			want:     true,
		},
		{
			name:     "absolute include pattern is ignored",
			patterns: []string{included},
			filename: included,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Bundle{BundleRootPath: root}
			b.SetIncludePatterns(tt.patterns)
			require.Equal(t, tt.want, b.IsFileIncluded(tt.filename))
		})
	}
}

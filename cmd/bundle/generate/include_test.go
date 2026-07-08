package generate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsIncluded(t *testing.T) {
	// Lay out a bundle root with a couple of resource files on disk, because
	// isIncluded resolves the include patterns with filepath.Glob against the
	// files that actually exist.
	root := t.TempDir()
	included := filepath.Join(root, "resources", "my.job.yml")
	nested := filepath.Join(root, "resources", "sub", "other.job.yml")
	for _, f := range []string{included, nested} {
		require.NoError(t, os.MkdirAll(filepath.Dir(f), 0o755))
		require.NoError(t, os.WriteFile(f, []byte("{}"), 0o644))
	}

	tests := []struct {
		name     string
		include  []string
		filename string
		want     bool
	}{
		{
			name:     "matched by glob",
			include:  []string{"resources/*.yml"},
			filename: included,
			want:     true,
		},
		{
			name:     "not matched, empty include",
			include:  nil,
			filename: included,
			want:     false,
		},
		{
			name:     "glob is not recursive",
			include:  []string{"resources/*.yml"},
			filename: nested,
			want:     false,
		},
		{
			name:     "matched by exact path",
			include:  []string{"resources/my.job.yml"},
			filename: included,
			want:     true,
		},
		{
			name:     "absolute include pattern is ignored",
			include:  []string{included},
			filename: included,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, isIncluded(root, tt.include, tt.filename))
		})
	}
}

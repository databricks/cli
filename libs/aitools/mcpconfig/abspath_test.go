package mcpconfig

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAbsoluteAsWrittenKeepsAbsolutePaths(t *testing.T) {
	abs := filepath.Join(t.TempDir(), "config.json")
	got, err := AbsoluteAsWritten(abs)
	require.NoError(t, err)
	assert.Equal(t, abs, got)
}

func TestAbsoluteAsWrittenJoinsRelativePathToWorkingDirectory(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)
	got, err := AbsoluteAsWritten("cfg/../.claude.json")
	require.NoError(t, err)
	// Not cleaned: the ".." stays, so that it resolves through a symlink as the OS does.
	assert.Equal(t, wd+string(filepath.Separator)+filepath.FromSlash("cfg/../.claude.json"), got)
}

// TestJoinWorkingDirKeepsTheVolumeOfRootRelativePaths checks a path that starts with a separator but has no volume, such as
// "\claude" on Windows. It is on the working directory's volume, not under the whole directory. The expectations use
// filepath only, so the table means the same on every OS.
func TestJoinWorkingDirKeepsTheVolumeOfRootRelativePaths(t *testing.T) {
	wd := filepath.FromSlash("C:/work/project")
	cases := []struct {
		name string
		path string
		want string
	}{
		{
			name: "relative path joins the working directory",
			path: "cfg/../.claude.json",
			want: wd + string(filepath.Separator) + filepath.FromSlash("cfg/../.claude.json"),
		},
		{
			name: "root-relative path takes the working directory's volume",
			path: "/claude",
			want: filepath.VolumeName(wd) + filepath.FromSlash("/claude"),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, joinWorkingDir(filepath.FromSlash(tc.path), wd))
		})
	}
}

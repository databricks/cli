package installer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeAiDevKitMarker writes a .ai-dev-kit/version marker under baseDir (a temp
// dir standing in for a project root or the global home).
func writeAiDevKitMarker(t *testing.T, baseDir, contents string) {
	t.Helper()
	dir := filepath.Join(baseDir, aiDevKitStateDir)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, aiDevKitVersionFile), []byte(contents), 0o644))
}

func TestAiDevKitVersion(t *testing.T) {
	tests := []struct {
		name string
		// global/project contents; "" means no marker written for that scope.
		global      string
		project     string
		wantVersion string
		wantOK      bool
	}{
		{"not installed", "", "", "", false},
		{"global marker", "1.2.3\n", "", "1.2.3", true},
		{"project marker", "", "2.0.0\n", "2.0.0", true},
		{"project overrides global", "1.2.3\n", "2.0.0\n", "2.0.0", true},
		{"dev version", "dev\n", "", "dev", true},
		{"empty marker still installed", "\n", "", "", true},
		{"whitespace-only marker", "  \n", "", "", true},
		// A blank project marker must not mask a real global version: still
		// installed, but the global version wins over project's "unknown".
		{"empty project does not mask global", "1.2.3\n", "\n", "1.2.3", true},
		{"trailing newline trimmed", "1.2.3\n", "", "1.2.3", true},
		{"multi-line keeps first line", "1.2.3\nextra\n", "", "1.2.3", true},
		{"multi-line CRLF has no trailing hyphen", "1.2.3\r\nextra\r\n", "", "1.2.3", true},
		{"sanitizes illegal chars", "1.2.3 (beta/rc)\n", "", "1.2.3--beta-rc-", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			project := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("USERPROFILE", home)
			t.Chdir(project)

			if tc.global != "" {
				writeAiDevKitMarker(t, home, tc.global)
			}
			if tc.project != "" {
				writeAiDevKitMarker(t, project, tc.project)
			}

			version, ok := AiDevKitVersion(t.Context())
			assert.Equal(t, tc.wantOK, ok)
			assert.Equal(t, tc.wantVersion, version)
		})
	}
}

// TestAiDevKitVersionHomeOverride verifies AIDEVKIT_HOME points directly at the
// install root, so the marker is read from $AIDEVKIT_HOME/version, not
// $AIDEVKIT_HOME/.ai-dev-kit/version.
func TestAiDevKitVersionHomeOverride(t *testing.T) {
	home := t.TempDir()
	installRoot := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv(aiDevKitHomeEnv, installRoot)
	t.Chdir(t.TempDir())

	require.NoError(t, os.WriteFile(filepath.Join(installRoot, aiDevKitVersionFile), []byte("3.1.4\n"), 0o644))
	// A marker under ~/.ai-dev-kit must be ignored once AIDEVKIT_HOME is set.
	writeAiDevKitMarker(t, home, "9.9.9\n")

	version, ok := AiDevKitVersion(t.Context())
	assert.True(t, ok)
	assert.Equal(t, "3.1.4", version)
}

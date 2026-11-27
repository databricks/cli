//go:build unix

package aircmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertToDabsRejectsFIFORoot(t *testing.T) {
	if os.Getenv("TEST_AIR_CONVERT_FIFO_CHILD") == "" {
		// Opening a FIFO for ownership lookup can block; bound that regression in a child.
		ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestConvertToDabsRejectsFIFORoot$")
		cmd.Env = append(os.Environ(), "TEST_AIR_CONVERT_FIFO_CHILD=1")
		output, err := cmd.CombinedOutput()
		require.NoError(t, ctx.Err(), "conversion blocked reading a FIFO: %s", output)
		require.NoError(t, err, "%s", output)
		return
	}
	for _, tc := range []struct {
		name    string
		symlink bool
		force   bool
	}{
		{"fifo", false, false},
		{"forced-fifo", false, true},
		{"symlink", true, false},
		{"forced-symlink", true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := writeConfig(t, minimalConfig)
			cfg, err := loadRunConfig(path)
			require.NoError(t, err)
			dir := filepath.Dir(path)
			root := filepath.Join(dir, bundleFileName)
			fifo := root
			if tc.symlink {
				fifo = filepath.Join(dir, "fifo")
				require.NoError(t, os.Symlink(fifo, root))
			}
			require.NoError(t, syscall.Mkfifo(fifo, 0o600))

			_, err = writeBundle(t.Context(), cfg, path, dir, tc.force)
			assert.EqualError(t, err, fmt.Sprintf("databricks.yml already exists in %s; move or remove it before converting", filepath.ToSlash(dir)))
			assert.NoDirExists(t, filepath.Join(dir, generatedArtifactsDir))
			info, err := os.Stat(root)
			require.NoError(t, err)
			assert.NotZero(t, info.Mode()&os.ModeNamedPipe)
		})
	}
}

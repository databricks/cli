package terraform

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func downloadAndChecksum(t *testing.T, url, expectedChecksum string) {
	resp, err := http.Get(url)
	require.NoError(t, err)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("failed to download %s: %s", url, resp.Status)
	}

	tmpDir := t.TempDir()
	tmpFile, err := os.Create(filepath.Join(tmpDir, "archive.zip"))
	require.NoError(t, err)
	defer tmpFile.Close()

	_, err = io.Copy(tmpFile, resp.Body)
	require.NoError(t, err)

	_, err = tmpFile.Seek(0, 0) // go back to the start of the file
	require.NoError(t, err)

	hash := sha256.New()
	_, err = io.Copy(hash, tmpFile)
	require.NoError(t, err)

	checksum := hex.EncodeToString(hash.Sum(nil))
	assert.Equal(t, expectedChecksum, checksum)
}

func TestTerraformArchiveChecksums(t *testing.T) {
	armUrl := fmt.Sprintf("https://releases.hashicorp.com/terraform/%s/terraform_%s_linux_arm64.zip", TerraformVersion, TerraformVersion)
	amdUrl := fmt.Sprintf("https://releases.hashicorp.com/terraform/%s/terraform_%s_linux_amd64.zip", TerraformVersion, TerraformVersion)

	downloadAndChecksum(t, amdUrl, checksumLinuxAmd64)
	downloadAndChecksum(t, armUrl, checksumLinuxArm64)
}

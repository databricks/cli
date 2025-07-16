package terraform

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/env"
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
	tv, isDefault, err := GetTerraformVersion(context.Background())
	require.NoError(t, err)
	assert.True(t, isDefault)
	armUrl := fmt.Sprintf("https://releases.hashicorp.com/terraform/%s/terraform_%s_linux_arm64.zip", tv.Version.String(), tv.Version.String())
	amdUrl := fmt.Sprintf("https://releases.hashicorp.com/terraform/%s/terraform_%s_linux_amd64.zip", tv.Version.String(), tv.Version.String())

	downloadAndChecksum(t, amdUrl, tv.ChecksumLinuxAmd64)
	downloadAndChecksum(t, armUrl, tv.ChecksumLinuxArm64)
}

func TestGetTerraformVersionDefault(t *testing.T) {
	// Verify that the default version is used
	tv, isDefault, err := GetTerraformVersion(context.Background())
	require.NoError(t, err)
	assert.True(t, isDefault)
	assert.Equal(t, defaultTerraformVersion.Version.String(), tv.Version.String())
	assert.Equal(t, defaultTerraformVersion.ChecksumLinuxArm64, tv.ChecksumLinuxArm64)
	assert.Equal(t, defaultTerraformVersion.ChecksumLinuxAmd64, tv.ChecksumLinuxAmd64)
}

func TestGetTerraformVersionOverride(t *testing.T) {
	// Set the override version
	overrideVersion := "1.12.2"
	ctx := env.Set(context.Background(), TerraformVersionEnv, overrideVersion)

	// Verify that the override version is used
	tv, isDefault, err := GetTerraformVersion(ctx)
	require.NoError(t, err)
	assert.False(t, isDefault)
	assert.Equal(t, overrideVersion, tv.Version.String())
	assert.Empty(t, tv.ChecksumLinuxArm64)
	assert.Empty(t, tv.ChecksumLinuxAmd64)
}

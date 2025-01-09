package acceptance_test

import (
	"crypto/md5"
	"encoding/hex"
	"hash"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func isCached(t *testing.T, cacheDir, initialHash, scriptContents, dir string) (bool, string) {
	if cacheDir == "" {
		return false, ""
	}

	hash := md5.New()
	_, err := hash.Write([]byte(initialHash))
	require.NoError(t, err)
	_, err = hash.Write([]byte(scriptContents))
	require.NoError(t, err)

	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		require.NoError(t, err)
		if !info.IsDir() {
			AddFile(t, hash, path)
		}
		return nil
	})
	require.NoError(t, err)
	checksum := GetChecksum(hash)

	checksumFile := filepath.Join(cacheDir, checksum)

	_, err = os.Stat(checksumFile)
	if err != nil {
		if !os.IsNotExist(err) {
			t.Logf("Failed to read cache: %s", err)
		}
		return false, checksumFile
	}

	return true, checksumFile
}

func writeCache(t *testing.T, checksumFile string) {
	if checksumFile == "" {
		return
	}

	currentTime := time.Now()
	formattedTime := currentTime.Format(time.RFC3339)

	err := os.WriteFile(checksumFile, []byte(formattedTime), 0o644)
	if err != nil {
		t.Logf("Failed to write cache %s: %s", checksumFile, err)
	}
}

func GetCacheLocation(t *testing.T) string {
	dir := GetGoCache(t)
	dir = filepath.Join(dir, "ff")
	err := os.MkdirAll(dir, 0o755)
	if err != nil {
		t.Logf("Failed to create cache dir %s: %s", dir, err)
		return ""
	}
	return dir
}

func GetGoCache(t *testing.T) string {
	defaultDir := os.Getenv("GOCACHE")
	if defaultDir != "" {
		if filepath.IsAbs(defaultDir) {
			return defaultDir
		} else {
			t.Logf("GOCACHE is not absolute path: %s", defaultDir)
		}
	}

	dir, err := os.UserCacheDir()
	if err != nil {
		t.Logf("UserCacheDir failed: %s", err)
		return ""
	}

	return filepath.Join(dir, "go-build")
}

func MustCalculateMD5(t *testing.T, filePath string) string {
	start := time.Now()
	hash := md5.New()
	AddFile(t, hash, filePath)
	checksum := hash.Sum(nil)
	elapsed := time.Since(start)
	result := hex.EncodeToString(checksum)
	t.Logf("Calculated md5 of %s in %s: %s", filePath, elapsed, result)
	return result
}

func AddFile(t *testing.T, h hash.Hash, filePath string) {
	file, err := os.Open(filePath)
	require.NoError(t, err)
	defer file.Close()
	_, err = io.Copy(h, file)
	require.NoError(t, err)
}

func GetChecksum(h hash.Hash) string {
	checksum := h.Sum(nil)
	return hex.EncodeToString(checksum)
}

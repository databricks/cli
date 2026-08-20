//go:build windows

package dockercredentials

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/require"
	"golang.org/x/sys/windows"
)

func TestCreateOwnerOnlyTempFileRestrictsWindowsDACL(t *testing.T) {
	file, err := createOwnerOnlyTempFile(t.TempDir(), "config.json.*")
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = file.Close()
	})

	descriptor, err := windows.GetSecurityInfo(
		windows.Handle(file.Fd()),
		windows.SE_FILE_OBJECT,
		windows.DACL_SECURITY_INFORMATION,
	)
	require.NoError(t, err)
	control, _, err := descriptor.Control()
	require.NoError(t, err)
	require.NotZero(t, control&windows.SE_DACL_PROTECTED)

	dacl, _, err := descriptor.DACL()
	require.NoError(t, err)
	require.Equal(t, uint16(1), dacl.AceCount)
	var ace *windows.ACCESS_ALLOWED_ACE
	require.NoError(t, windows.GetAce(dacl, 0, &ace))
	const fileAllAccess = windows.STANDARD_RIGHTS_REQUIRED | windows.SYNCHRONIZE | 0x1ff
	require.True(t, ace.Mask&windows.GENERIC_ALL != 0 || ace.Mask&fileAllAccess == fileAllAccess)

	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	require.NoError(t, err)
	aceSID := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
	require.True(t, user.User.Sid.Equals(aceSID))
}

func TestWindowsCreateFilePathSupportsExtendedLengthPaths(t *testing.T) {
	longDrivePath := `C:\` + strings.Repeat("a", 260)
	longUNCPath := `\\server\share\` + strings.Repeat("a", 260)
	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "short", path: `C:\config.json`, want: `C:\config.json`},
		{name: "drive", path: longDrivePath, want: `\\?\` + longDrivePath},
		{name: "UNC", path: longUNCPath, want: `\\?\UNC\server\share\` + strings.Repeat("a", 260)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := windowsCreateFilePath(tt.path)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestWindowsCreateFilePathResolvesRelativePathBeforeLengthCheck(t *testing.T) {
	baseDir := t.TempDir()
	componentLength := 247 - len(baseDir) - 1
	if componentLength <= 0 {
		t.Skip("temporary directory already exceeds the legacy Windows path threshold")
	}
	longDir := filepath.Join(baseDir, strings.Repeat("a", componentLength))
	require.Equal(t, 247, len(longDir))
	require.GreaterOrEqual(t, len(filepath.Join(longDir, "config.json")), 248)
	require.NoError(t, os.MkdirAll(longDir, 0o755))
	extendedDir := longDir
	if !strings.HasPrefix(longDir, `\\?\`) {
		if strings.HasPrefix(longDir, `\\`) {
			extendedDir = `\\?\UNC\` + strings.TrimPrefix(longDir, `\\`)
		} else {
			extendedDir = `\\?\` + longDir
		}
	}
	t.Chdir(extendedDir)

	got, err := windowsCreateFilePath("config.json")
	require.NoError(t, err)
	require.Equal(t, filepath.Join(extendedDir, "config.json"), got)
}

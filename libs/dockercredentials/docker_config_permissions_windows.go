//go:build windows

package dockercredentials

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

const maxTempFileAttempts = 100

// windowsCreateFilePath adds an extended-length prefix at Go's conservative legacy Windows path threshold.
// See https://learn.microsoft.com/en-us/windows/win32/fileio/naming-a-file#maximum-path-length-limitation.
func windowsCreateFilePath(path string) (string, error) {
	if strings.HasPrefix(path, `\\?\`) || strings.HasPrefix(path, `\??\`) || strings.HasPrefix(path, `\\.\`) {
		return path, nil
	}
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(absolutePath, `\\?\`) || strings.HasPrefix(absolutePath, `\??\`) || strings.HasPrefix(absolutePath, `\\.\`) {
		return absolutePath, nil
	}
	if len(absolutePath) < 248 {
		return path, nil
	}
	if strings.HasPrefix(absolutePath, `\\`) {
		return `\\?\UNC\` + strings.TrimPrefix(absolutePath, `\\`), nil
	}
	return `\\?\` + absolutePath, nil
}

// createOwnerOnlyTempFile creates the Windows temp file with a protected current-user DACL at handle creation.
// See https://learn.microsoft.com/en-us/windows/win32/api/fileapi/nf-fileapi-createfilew.
func createOwnerOnlyTempFile(dir, pattern string) (*os.File, error) {
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return nil, fmt.Errorf("get current Windows user: %w", err)
	}
	descriptor, err := windows.SecurityDescriptorFromString(fmt.Sprintf("D:P(A;;GA;;;%s)", user.User.Sid.String()))
	if err != nil {
		return nil, fmt.Errorf("create owner-only Windows security descriptor: %w", err)
	}

	var pinner runtime.Pinner
	pinner.Pin(descriptor)
	defer pinner.Unpin()
	attributes := windows.SecurityAttributes{
		Length:             uint32(unsafe.Sizeof(windows.SecurityAttributes{})),
		SecurityDescriptor: descriptor,
	}

	for range maxTempFileAttempts {
		// The empty placeholder borrows os.CreateTemp's naming; CREATE_NEW prevents adopting a raced replacement.
		placeholder, err := os.CreateTemp(dir, pattern)
		if err != nil {
			return nil, err
		}
		path := placeholder.Name()
		if err := placeholder.Close(); err != nil {
			_ = os.Remove(path)
			return nil, fmt.Errorf("close temporary filename reservation %s: %w", path, err)
		}
		if err := os.Remove(path); err != nil {
			return nil, fmt.Errorf("remove temporary filename reservation %s: %w", path, err)
		}

		createPath, err := windowsCreateFilePath(path)
		if err != nil {
			return nil, fmt.Errorf("resolve temporary Docker config path %s: %w", path, err)
		}
		pathPtr, err := windows.UTF16PtrFromString(createPath)
		if err != nil {
			return nil, fmt.Errorf("encode temporary Docker config path %s: %w", path, err)
		}
		handle, err := windows.CreateFile(
			pathPtr,
			windows.GENERIC_READ|windows.GENERIC_WRITE,
			windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE,
			&attributes,
			windows.CREATE_NEW,
			windows.FILE_ATTRIBUTE_NORMAL,
			0,
		)
		if errors.Is(err, windows.ERROR_FILE_EXISTS) || errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("create owner-only temporary Docker config %s: %w", path, err)
		}
		return os.NewFile(uintptr(handle), path), nil
	}
	return nil, fmt.Errorf("create owner-only temporary Docker config in %s: too many name collisions", dir)
}

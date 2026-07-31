//go:build windows

package dockercredentials

import "os"

func isExecutableForPath(_ string, mode os.FileMode, goos string) bool {
	if goos == "windows" {
		return true
	}
	return mode&0o111 != 0
}

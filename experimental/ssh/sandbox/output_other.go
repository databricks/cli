//go:build !linux

package sandbox

import (
	"errors"
	"io"
	"os"
)

func duplicateServerOutput(*os.File) (io.WriteCloser, error) {
	return nil, errors.New("server output must be a Linux command-execution pipe")
}

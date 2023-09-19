package fileset

import (
	"io/fs"
	"path/filepath"
	"strings"
	"time"

	"github.com/databricks/cli/libs/notebook"
)

type File struct {
	fs.DirEntry
	Absolute, Relative string
}

func (f File) Modified() (ts time.Time) {
	info, err := f.Info()
	if err != nil {
		// return default time, beginning of epoch
		return ts
	}
	return info.ModTime()
}

func (f File) RemotePath() (string, error) {
	workspacePath := filepath.ToSlash(f.Relative)
	isNotebook, _, err := notebook.Detect(f.Absolute)
	if err != nil {
		return "", err
	}
	if isNotebook {
		ext := filepath.Ext(workspacePath)
		workspacePath = strings.TrimSuffix(workspacePath, ext)
	}
	return workspacePath, nil
}

package fs

import (
	"io/fs"
	"time"
)

type lsOutput struct {
	Name    string    `json:"name"`
	IsDir   bool      `json:"is_directory"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"last_modified"`
}

func toLsOutput(f fs.DirEntry) (*lsOutput, error) {
	info, err := f.Info()
	if err != nil {
		return nil, err
	}

	return &lsOutput{
		Name:    f.Name(),
		IsDir:   f.IsDir(),
		Size:    info.Size(),
		ModTime: info.ModTime(),
	}, nil
}

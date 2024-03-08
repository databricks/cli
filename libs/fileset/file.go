package fileset

import (
	"io/fs"
	"time"
)

type File struct {
	fs.DirEntry
	Absolute string `json:"absolute"`
	Relative string `json:"relative"`
}

func (f File) Modified() (ts time.Time) {
	info, err := f.Info()
	if err != nil {
		// return default time, beginning of epoch
		return ts
	}
	return info.ModTime()
}

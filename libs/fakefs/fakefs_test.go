package fakefs

import (
	"io/fs"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFile(t *testing.T) {
	var fakefile fs.File = File{
		FileInfo: FileInfo{
			FakeName: "file",
		},
	}

	_, err := fakefile.Read([]byte{})
	assert.ErrorIs(t, err, ErrNotImplemented)

	fi, err := fakefile.Stat()
	assert.NoError(t, err)
	assert.Equal(t, "file", fi.Name())

	err = fakefile.Close()
	assert.NoError(t, err)
}

func TestFS(t *testing.T) {
	var fakefs fs.FS = FS{
		"file": File{},
	}

	_, err := fakefs.Open("doesntexist")
	assert.ErrorIs(t, err, fs.ErrNotExist)

	_, err = fakefs.Open("file")
	assert.NoError(t, err)
}

package testfile

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Use this struct to work with files in a github actions test environment
type TestFile struct {
	mtime time.Time
	fd    *os.File
	path  string
	// to make close idempotent
	isOpen bool
}

func CreateFile(t *testing.T, path string) *TestFile {
	f, err := os.Create(path)
	assert.NoError(t, err)

	fileInfo, err := os.Stat(path)
	assert.NoError(t, err)

	return &TestFile{
		path:   path,
		fd:     f,
		mtime:  fileInfo.ModTime(),
		isOpen: true,
	}
}

func (f *TestFile) Close(t *testing.T) {
	if f.isOpen {
		err := f.fd.Close()
		assert.NoError(t, err)
		f.isOpen = false
	}
}

func (f *TestFile) Overwrite(t *testing.T, s string) {
	err := os.Truncate(f.path, 0)
	assert.NoError(t, err)

	_, err = f.fd.Seek(0, 0)
	assert.NoError(t, err)

	_, err = f.fd.WriteString(s)
	assert.NoError(t, err)

	// We manually update mtime after write because github actions file
	// system does not :')
	err = os.Chtimes(f.path, f.mtime.Add(time.Minute), f.mtime.Add(time.Minute))
	assert.NoError(t, err)
	f.mtime = f.mtime.Add(time.Minute)
}

func (f *TestFile) Remove(t *testing.T) {
	f.Close(t)
	// On Windows, a file that was just written can be briefly held open by another
	// process (e.g. antivirus or the search indexer), so remove fails with a sharing
	// violation. Retry for a bounded time before failing.
	deadline := time.Now().Add(10 * time.Second)
	for {
		err := os.Remove(f.path)
		if err == nil || time.Now().After(deadline) {
			assert.NoError(t, err)
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
}

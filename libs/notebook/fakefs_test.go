package notebook

import (
	"fmt"
	"io/fs"
	"time"

	"github.com/databricks/databricks-sdk-go/service/workspace"
)

type fakeFS struct {
	fakeFile
}

type fakeFile struct {
	fakeFileInfo
}

func (f fakeFile) Close() error {
	return nil
}

func (f fakeFile) Read(p []byte) (n int, err error) {
	return 0, fmt.Errorf("not implemented")
}

func (f fakeFile) Stat() (fs.FileInfo, error) {
	return f.fakeFileInfo, nil
}

type fakeFileInfo struct {
	oi workspace.ObjectInfo
}

func (f fakeFileInfo) WorkspaceObjectInfo() workspace.ObjectInfo {
	return f.oi
}

func (f fakeFileInfo) Name() string {
	return ""
}

func (f fakeFileInfo) Size() int64 {
	return 0
}

func (f fakeFileInfo) Mode() fs.FileMode {
	return 0
}

func (f fakeFileInfo) ModTime() time.Time {
	return time.Time{}
}

func (f fakeFileInfo) IsDir() bool {
	return false
}

func (f fakeFileInfo) Sys() any {
	return nil
}

func (f fakeFS) Open(name string) (fs.File, error) {
	return f.fakeFile, nil
}

func (f fakeFS) Stat(name string) (fs.FileInfo, error) {
	panic("not implemented")
}

func (f fakeFS) ReadDir(name string) ([]fs.DirEntry, error) {
	panic("not implemented")
}

func (f fakeFS) ReadFile(name string) ([]byte, error) {
	panic("not implemented")
}

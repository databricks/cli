package testserver

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"path"
	"slices"
	"strconv"
	"strings"

	"github.com/databricks/databricks-sdk-go/service/files"
)

// fakeDbfs is the DBFS namespace (dbfs:/...), separate from workspace files: a path written
// through the DBFS API is not visible through the workspace or Files APIs, as on a real workspace.
type fakeDbfs struct {
	files      map[string]dbfsFile
	dirs       map[string]int64 // path -> modification time (ms)
	handles    map[int64]*dbfsHandle
	nextHandle int64
}

type dbfsFile struct {
	data    []byte
	modTime int64
}

// dbfsHandle is an open streaming write: blocks accumulate until close commits the file.
type dbfsHandle struct {
	path string
	data []byte
}

func (s *FakeWorkspace) dbfsState() *fakeDbfs {
	if s.dbfs == nil {
		s.dbfs = &fakeDbfs{
			files:   map[string]dbfsFile{},
			dirs:    map[string]int64{"/": nowMilli()},
			handles: map[int64]*dbfsHandle{},
		}
	}
	return s.dbfs
}

func dbfsError(statusCode int, errorCode, message string) Response {
	return Response{
		StatusCode: statusCode,
		Body:       map[string]string{"error_code": errorCode, "message": message},
	}
}

func dbfsNotFound(p string) Response {
	return dbfsError(404, "RESOURCE_DOES_NOT_EXIST", fmt.Sprintf("No file or directory exists on path %s.", p))
}

func dbfsCleanPath(p string) string {
	p = strings.TrimPrefix(p, "dbfs:")
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return path.Clean(p)
}

// mkdirs creates p and its parents. It fails if any of them is a file.
func (d *fakeDbfs) mkdirs(p string) *Response {
	var missing []string
	for dir := p; ; dir = path.Dir(dir) {
		if _, isFile := d.files[dir]; isFile {
			resp := dbfsError(400, "RESOURCE_ALREADY_EXISTS", fmt.Sprintf("A file or directory already exists at the input path %s.", dir))
			return &resp
		}
		if _, isDir := d.dirs[dir]; isDir {
			break
		}
		missing = append(missing, dir)
	}
	now := nowMilli()
	for _, dir := range missing {
		d.dirs[dir] = now
	}
	return nil
}

func (s *FakeWorkspace) DbfsGetStatus(req Request) Response {
	defer s.LockUnlock()()
	d := s.dbfsState()
	p := dbfsCleanPath(req.URL.Query().Get("path"))

	if f, ok := d.files[p]; ok {
		return Response{Body: files.FileInfo{Path: p, FileSize: int64(len(f.data)), ModificationTime: f.modTime}}
	}
	if modTime, ok := d.dirs[p]; ok {
		return Response{Body: files.FileInfo{Path: p, IsDir: true, ModificationTime: modTime}}
	}
	return dbfsNotFound(p)
}

func (s *FakeWorkspace) DbfsList(req Request) Response {
	defer s.LockUnlock()()
	d := s.dbfsState()
	p := dbfsCleanPath(req.URL.Query().Get("path"))

	// Listing a file returns the file itself, which the CLI uses to detect "not a directory".
	if f, ok := d.files[p]; ok {
		return Response{Body: files.ListStatusResponse{Files: []files.FileInfo{
			{Path: p, FileSize: int64(len(f.data)), ModificationTime: f.modTime},
		}}}
	}
	if _, ok := d.dirs[p]; !ok {
		return dbfsNotFound(p)
	}

	var entries []files.FileInfo
	for filePath, f := range d.files {
		if path.Dir(filePath) == p {
			entries = append(entries, files.FileInfo{Path: filePath, FileSize: int64(len(f.data)), ModificationTime: f.modTime})
		}
	}
	for dirPath, modTime := range d.dirs {
		if dirPath != p && path.Dir(dirPath) == p {
			entries = append(entries, files.FileInfo{Path: dirPath, IsDir: true, ModificationTime: modTime})
		}
	}
	slices.SortFunc(entries, func(a, b files.FileInfo) int { return strings.Compare(a.Path, b.Path) })
	return Response{Body: files.ListStatusResponse{Files: entries}}
}

func (s *FakeWorkspace) DbfsMkdirs(req Request) Response {
	var request files.MkDirs
	if err := json.Unmarshal(req.Body, &request); err != nil {
		return dbfsError(400, "INVALID_PARAMETER_VALUE", err.Error())
	}
	defer s.LockUnlock()()
	if resp := s.dbfsState().mkdirs(dbfsCleanPath(request.Path)); resp != nil {
		return *resp
	}
	return Response{Body: map[string]any{}}
}

func (s *FakeWorkspace) DbfsCreate(req Request) Response {
	var request files.Create
	if err := json.Unmarshal(req.Body, &request); err != nil {
		return dbfsError(400, "INVALID_PARAMETER_VALUE", err.Error())
	}
	defer s.LockUnlock()()
	d := s.dbfsState()
	p := dbfsCleanPath(request.Path)

	if _, isDir := d.dirs[p]; isDir {
		return dbfsError(400, "RESOURCE_ALREADY_EXISTS", fmt.Sprintf("A directory already exists at the input path %s.", p))
	}
	if _, isFile := d.files[p]; isFile && !request.Overwrite {
		return dbfsError(400, "RESOURCE_ALREADY_EXISTS", fmt.Sprintf("A file or directory already exists at the input path %s.", p))
	}
	// Like the real API, creating a file creates its missing parent directories.
	if resp := d.mkdirs(path.Dir(p)); resp != nil {
		return *resp
	}

	d.nextHandle++
	d.handles[d.nextHandle] = &dbfsHandle{path: p}
	return Response{Body: files.CreateResponse{Handle: d.nextHandle}}
}

func (s *FakeWorkspace) DbfsAddBlock(req Request) Response {
	var request files.AddBlock
	if err := json.Unmarshal(req.Body, &request); err != nil {
		return dbfsError(400, "INVALID_PARAMETER_VALUE", err.Error())
	}
	data, err := base64.StdEncoding.DecodeString(request.Data)
	if err != nil {
		return dbfsError(400, "INVALID_PARAMETER_VALUE", "data is not valid base64: "+err.Error())
	}
	defer s.LockUnlock()()
	h, ok := s.dbfsState().handles[request.Handle]
	if !ok {
		return dbfsError(404, "RESOURCE_DOES_NOT_EXIST", fmt.Sprintf("No handle %d.", request.Handle))
	}
	h.data = append(h.data, data...)
	return Response{Body: map[string]any{}}
}

func (s *FakeWorkspace) DbfsClose(req Request) Response {
	var request files.Close
	if err := json.Unmarshal(req.Body, &request); err != nil {
		return dbfsError(400, "INVALID_PARAMETER_VALUE", err.Error())
	}
	defer s.LockUnlock()()
	d := s.dbfsState()
	h, ok := d.handles[request.Handle]
	if !ok {
		return dbfsError(404, "RESOURCE_DOES_NOT_EXIST", fmt.Sprintf("No handle %d.", request.Handle))
	}
	delete(d.handles, request.Handle)
	d.files[h.path] = dbfsFile{data: h.data, modTime: nowMilli()}
	return Response{Body: map[string]any{}}
}

func (s *FakeWorkspace) DbfsRead(req Request) Response {
	defer s.LockUnlock()()
	d := s.dbfsState()
	q := req.URL.Query()
	p := dbfsCleanPath(q.Get("path"))

	if _, isDir := d.dirs[p]; isDir {
		return dbfsError(400, "INVALID_PARAMETER_VALUE", fmt.Sprintf("The path %s is a directory.", p))
	}
	f, ok := d.files[p]
	if !ok {
		return dbfsNotFound(p)
	}

	offset, _ := strconv.ParseInt(q.Get("offset"), 10, 64)
	length, err := strconv.ParseInt(q.Get("length"), 10, 64)
	if err != nil || length <= 0 {
		length = int64(len(f.data))
	}
	start := min(offset, int64(len(f.data)))
	end := min(start+length, int64(len(f.data)))
	chunk := f.data[start:end]
	return Response{Body: files.ReadResponse{
		BytesRead: int64(len(chunk)),
		Data:      base64.StdEncoding.EncodeToString(chunk),
	}}
}

func (s *FakeWorkspace) DbfsDelete(req Request) Response {
	var request files.Delete
	if err := json.Unmarshal(req.Body, &request); err != nil {
		return dbfsError(400, "INVALID_PARAMETER_VALUE", err.Error())
	}
	defer s.LockUnlock()()
	d := s.dbfsState()
	p := dbfsCleanPath(request.Path)

	if _, isFile := d.files[p]; isFile {
		delete(d.files, p)
		return Response{Body: map[string]any{}}
	}
	if _, isDir := d.dirs[p]; !isDir {
		// The real API succeeds when the path does not exist.
		return Response{Body: map[string]any{}}
	}

	prefix := strings.TrimSuffix(p, "/") + "/"
	var children []string
	for filePath := range d.files {
		if strings.HasPrefix(filePath, prefix) {
			children = append(children, filePath)
		}
	}
	for dirPath := range d.dirs {
		if strings.HasPrefix(dirPath, prefix) {
			children = append(children, dirPath)
		}
	}
	if len(children) > 0 && !request.Recursive {
		return dbfsError(400, "IO_ERROR", "Directory is not empty. Use recursive delete or remove contents first.")
	}
	for _, child := range children {
		delete(d.files, child)
		delete(d.dirs, child)
	}
	if p != "/" {
		delete(d.dirs, p)
	}
	return Response{Body: map[string]any{}}
}

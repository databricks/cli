package testserver

import (
	"encoding/json"
	"maps"
	"net/http"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"

	sdktime "github.com/databricks/databricks-sdk-go/common/types/time"
	"github.com/databricks/databricks-sdk-go/service/sandbox"
)

// TestDefaultSandboxId is the sandbox that acceptance tests address locally (TEST_SANDBOX_ID).
const TestDefaultSandboxId = "test-sandbox-1234"

// The sandbox file APIs report most filesystem failures with this one reason and message,
// whatever the condition; only the HTTP status differs. The download and upload endpoints
// use their own messages and no reason.
const (
	sandboxFilesFsReason   = "SANDBOX_FILES_FILESYSTEM_ERROR"
	sandboxFilesFsMessage  = "sandbox filesystem operation failed"
	sandboxFilesNotFound   = "The sandbox or file was not found."
	sandboxFilesWrongState = "The sandbox or file is not in the required state."
	sandboxFilesBadUpload  = "Invalid file request. Use an absolute UTF-8 path and supported headers."

	sandboxFilesMaxPageSize = 1000
	sandboxFilesDirSize     = 4096
	sandboxFilesOwner       = "sandbox-agent"
)

// fakeSandboxFs is the filesystem of one sandbox, keyed by absolute path. It models the
// server's semantics, which are looser than the Filer contract: uploads create parents,
// and deletes of absent paths succeed. The filer must close those gaps itself, so
// the fake must not.
type fakeSandboxFs struct {
	entries map[string]*sandboxFsEntry
}

type sandboxFsEntry struct {
	isDir       bool
	data        []byte
	mode        int
	permissions string
	owner       string
	modTime     time.Time
}

// sandboxFs returns the filesystem of sandboxID, creating it on first use. The fake doesn't
// model sandbox lifecycle, so every sandbox ID exists; a real server returns 404 for unknown IDs.
func (s *FakeWorkspace) sandboxFs(sandboxID string) *fakeSandboxFs {
	if s.sandboxFiles == nil {
		s.sandboxFiles = map[string]*fakeSandboxFs{}
	}
	sfs, ok := s.sandboxFiles[sandboxID]
	if !ok {
		now := time.Now()
		sfs = &fakeSandboxFs{entries: map[string]*sandboxFsEntry{
			"/":    {isDir: true, mode: 0o755, permissions: "drwxr-xr-x", owner: "root", modTime: now},
			"/tmp": {isDir: true, mode: 0o1777, permissions: "drwxrwxrwt", owner: "root", modTime: now},
		}}
		s.sandboxFiles[sandboxID] = sfs
	}
	return sfs
}

func newSandboxFileEntry(data []byte) *sandboxFsEntry {
	return &sandboxFsEntry{data: data, mode: 0o644, permissions: "-rw-r--r--", owner: sandboxFilesOwner, modTime: time.Now()}
}

func newSandboxDirEntry() *sandboxFsEntry {
	return &sandboxFsEntry{isDir: true, mode: 0o755, permissions: "drwxr-xr-x", owner: sandboxFilesOwner, modTime: time.Now()}
}

func sandboxFilesError(statusCode int, errorCode, message string) Response {
	return Response{
		StatusCode: statusCode,
		Body:       map[string]string{"error_code": errorCode, "message": message},
	}
}

// sandboxFsError is the generic filesystem error, which carries an ErrorInfo reason.
func sandboxFsError(statusCode int, errorCode string) Response {
	return Response{
		StatusCode: statusCode,
		Body: map[string]any{
			"error_code": errorCode,
			"message":    sandboxFilesFsMessage,
			"details": []map[string]any{{
				"@type":  "type.googleapis.com/google.rpc.ErrorInfo",
				"reason": sandboxFilesFsReason,
				"domain": "databricks.sandbox.files",
			}},
		},
	}
}

// sandboxFilesPath validates and cleans an absolute path argument named field.
func sandboxFilesPath(field, p string) (string, *Response) {
	if !strings.HasPrefix(p, "/") {
		resp := sandboxFilesError(400, "INVALID_PARAMETER_VALUE", field+" must be an absolute UTF-8 path")
		return "", &resp
	}
	// A ".." segment is rejected before the request reaches the service, with a bare 400.
	if slices.Contains(strings.Split(p, "/"), "..") {
		resp := Response{StatusCode: 400, Body: "Bad Request"}
		return "", &resp
	}
	return path.Clean(p), nil
}

// lookup returns the entry at p and, if p is absent, whether a parent component is a file.
func (sfs *fakeSandboxFs) lookup(p string) (entry *sandboxFsEntry, underFile bool) {
	if e, ok := sfs.entries[p]; ok {
		return e, false
	}
	for dir := path.Dir(p); dir != "/"; dir = path.Dir(dir) {
		if e, ok := sfs.entries[dir]; ok {
			return nil, !e.isDir
		}
	}
	return nil, false
}

func (sfs *fakeSandboxFs) isEmptyDir(p string) bool {
	prefix := strings.TrimSuffix(p, "/") + "/"
	for name := range sfs.entries {
		if strings.HasPrefix(name, prefix) {
			return false
		}
	}
	return true
}

// mkdirAll creates p and its missing parents. It reports false if p or a parent is a file.
func (sfs *fakeSandboxFs) mkdirAll(p string) bool {
	var missing []string
	for dir := p; ; dir = path.Dir(dir) {
		if e, ok := sfs.entries[dir]; ok {
			if !e.isDir {
				return false
			}
			break
		}
		missing = append(missing, dir)
		// A recursive delete of "/" removes the root entry too.
		if dir == "/" {
			break
		}
	}
	for _, dir := range missing {
		sfs.entries[dir] = newSandboxDirEntry()
	}
	return true
}

func (e *sandboxFsEntry) info(p string) sandbox.SandboxFileInfo {
	info := sandbox.SandboxFileInfo{
		FileType:    sandbox.SandboxFileTypeSandboxFileTypeFile,
		Group:       e.owner,
		Mode:        e.mode,
		ModifyTime:  sdktime.New(e.modTime),
		Name:        path.Base(p),
		Owner:       e.owner,
		Path:        p,
		Permissions: e.permissions,
		Size:        int64(len(e.data)),
		// The server sends these even when zero (an empty file has size 0).
		ForceSendFields: []string{"FileType", "Group", "Mode", "Name", "Owner", "Path", "Permissions", "Size"},
	}
	if e.isDir {
		info.FileType = sandbox.SandboxFileTypeSandboxFileTypeDirectory
		info.Size = sandboxFilesDirSize
	}
	return info
}

// SandboxFilesUpload creates or replaces a file, creating missing parents.
func (s *FakeWorkspace) SandboxFilesUpload(req Request, sandboxID string) Response {
	query := req.URL.Query()
	for name := range query {
		if name != "path" && name != "overwrite" {
			return sandboxFilesError(400, "INVALID_PARAMETER_VALUE", sandboxFilesBadUpload)
		}
	}
	overwrite := true
	if values, ok := query["overwrite"]; ok {
		if len(values) != 1 || (values[0] != "true" && values[0] != "false") {
			return sandboxFilesError(400, "INVALID_PARAMETER_VALUE", sandboxFilesBadUpload)
		}
		overwrite = values[0] == "true"
	}
	p, errResp := sandboxFilesPath("path", query.Get("path"))
	if errResp != nil {
		return *errResp
	}

	defer s.LockUnlock()()
	sfs := s.sandboxFs(sandboxID)
	existing, underFile := sfs.lookup(p)
	if !overwrite && existing != nil {
		return sandboxFilesError(http.StatusPreconditionFailed, "BAD_REQUEST", "The file does not satisfy the request preconditions.")
	}
	if underFile || (existing != nil && existing.isDir) {
		return sandboxFilesError(409, "BAD_REQUEST", sandboxFilesWrongState)
	}
	if !sfs.mkdirAll(path.Dir(p)) {
		return sandboxFilesError(409, "BAD_REQUEST", sandboxFilesWrongState)
	}
	sfs.entries[p] = newSandboxFileEntry(req.Body)
	if existing != nil {
		return Response{StatusCode: http.StatusNoContent}
	}
	return Response{StatusCode: http.StatusCreated}
}

// SandboxFilesDownload returns the contents of a regular file.
func (s *FakeWorkspace) SandboxFilesDownload(req Request, sandboxID string) Response {
	p, errResp := sandboxFilesPath("path", req.URL.Query().Get("path"))
	if errResp != nil {
		return *errResp
	}

	defer s.LockUnlock()()
	entry, underFile := s.sandboxFs(sandboxID).lookup(p)
	if underFile || (entry != nil && entry.isDir) {
		return sandboxFilesError(409, "BAD_REQUEST", sandboxFilesWrongState)
	}
	if entry == nil {
		return sandboxFilesError(404, "NOT_FOUND", sandboxFilesNotFound)
	}
	return Response{
		Headers: http.Header{"Content-Type": {"application/octet-stream"}},
		Body:    slices.Clone(entry.data),
	}
}

// SandboxFilesGetMetadata describes a path.
func (s *FakeWorkspace) SandboxFilesGetMetadata(req Request, sandboxID string) Response {
	p, errResp := sandboxFilesPath("path", req.URL.Query().Get("path"))
	if errResp != nil {
		return *errResp
	}

	defer s.LockUnlock()()
	entry, underFile := s.sandboxFs(sandboxID).lookup(p)
	if underFile {
		return sandboxFsError(400, "BAD_REQUEST")
	}
	if entry == nil {
		return sandboxFsError(404, "NOT_FOUND")
	}
	info := entry.info(p)
	return Response{Body: sandbox.GetMetadataResponse{File: &info}}
}

// SandboxFilesList lists a directory's children in name order, one page at a time.
func (s *FakeWorkspace) SandboxFilesList(req Request, sandboxID string) Response {
	query := req.URL.Query()
	p, errResp := sandboxFilesPath("path", query.Get("path"))
	if errResp != nil {
		return *errResp
	}
	pageSize, _ := strconv.Atoi(query.Get("page_size"))
	if pageSize <= 0 || pageSize > sandboxFilesMaxPageSize {
		pageSize = sandboxFilesMaxPageSize
	}
	// The page token is an offset into the sorted listing; real tokens are opaque.
	offset, _ := strconv.Atoi(query.Get("page_token"))

	defer s.LockUnlock()()
	sfs := s.sandboxFs(sandboxID)
	entry, underFile := sfs.lookup(p)
	if underFile || (entry != nil && !entry.isDir) {
		return sandboxFsError(400, "BAD_REQUEST")
	}
	if entry == nil {
		return sandboxFsError(404, "NOT_FOUND")
	}

	var names []string
	for name := range sfs.entries {
		if name != "/" && path.Dir(name) == p {
			names = append(names, name)
		}
	}
	// Byte order of the paths is UTF-8 name order, since all share the same parent.
	slices.Sort(names)

	resp := sandbox.ListDirectoryContentsResponse{}
	end := min(offset+pageSize, len(names))
	for _, name := range names[min(offset, end):end] {
		resp.Contents = append(resp.Contents, sfs.entries[name].info(name))
	}
	if end < len(names) {
		resp.NextPageToken = strconv.Itoa(end)
	}
	return Response{Body: resp}
}

// SandboxFilesMkdir creates a directory, with missing parents unless parents is false.
func (s *FakeWorkspace) SandboxFilesMkdir(req Request, sandboxID string) Response {
	var body struct {
		Path    string `json:"path"`
		Parents *bool  `json:"parents"`
	}
	if err := json.Unmarshal(req.Body, &body); err != nil {
		return sandboxFilesError(400, "INVALID_PARAMETER_VALUE", err.Error())
	}
	p, errResp := sandboxFilesPath("path", body.Path)
	if errResp != nil {
		return *errResp
	}

	defer s.LockUnlock()()
	sfs := s.sandboxFs(sandboxID)
	entry, underFile := sfs.lookup(p)
	// An omitted parents defaults to true, per the API docs; the SDK omits false
	// unless it is in ForceSendFields. With parents, an existing directory is fine.
	if body.Parents == nil || *body.Parents {
		if !sfs.mkdirAll(p) {
			return sandboxFsError(400, "BAD_REQUEST")
		}
		return Response{Body: sandbox.CreateDirectoryResponse{}}
	}

	switch {
	case entry != nil:
		return sandboxFsError(409, "ALREADY_EXISTS")
	case underFile:
		return sandboxFsError(400, "BAD_REQUEST")
	case sfs.entries[path.Dir(p)] == nil:
		return sandboxFsError(404, "NOT_FOUND")
	}
	sfs.entries[p] = newSandboxDirEntry()
	return Response{Body: sandbox.CreateDirectoryResponse{}}
}

// SandboxFilesDelete deletes a path. It succeeds if the path is absent.
func (s *FakeWorkspace) SandboxFilesDelete(req Request, sandboxID string) Response {
	var body sandbox.DeletePathRequest
	if err := json.Unmarshal(req.Body, &body); err != nil {
		return sandboxFilesError(400, "INVALID_PARAMETER_VALUE", err.Error())
	}
	p, errResp := sandboxFilesPath("path", body.Path)
	if errResp != nil {
		return *errResp
	}

	defer s.LockUnlock()()
	sfs := s.sandboxFs(sandboxID)
	entry, _ := sfs.lookup(p)
	if entry == nil {
		return Response{Body: sandbox.DeletePathResponse{}}
	}
	if entry.isDir && !body.Recursive && !sfs.isEmptyDir(p) {
		return sandboxFsError(400, "BAD_REQUEST")
	}
	prefix := p + "/"
	for name := range sfs.entries {
		if name == p || strings.HasPrefix(name, prefix) {
			delete(sfs.entries, name)
		}
	}
	return Response{Body: sandbox.DeletePathResponse{}}
}

// SandboxFilesRename moves a path. The destination must be absent: the check and the move
// happen under one lock, so of several concurrent renames onto the same destination exactly
// one succeeds, as on the real server.
func (s *FakeWorkspace) SandboxFilesRename(req Request, sandboxID string) Response {
	var body sandbox.RenamePathRequest
	if err := json.Unmarshal(req.Body, &body); err != nil {
		return sandboxFilesError(400, "INVALID_PARAMETER_VALUE", err.Error())
	}
	src, errResp := sandboxFilesPath("source", body.Source)
	if errResp != nil {
		return *errResp
	}
	// The server rejects a destination with a trailing slash instead of cleaning it.
	if strings.HasSuffix(body.Destination, "/") && body.Destination != "/" {
		return sandboxFsError(400, "BAD_REQUEST")
	}
	dst, errResp := sandboxFilesPath("destination", body.Destination)
	if errResp != nil {
		return *errResp
	}

	defer s.LockUnlock()()
	sfs := s.sandboxFs(sandboxID)
	srcEntry, _ := sfs.lookup(src)
	dstEntry, dstUnderFile := sfs.lookup(dst)
	switch {
	case srcEntry == nil:
		return sandboxFsError(404, "NOT_FOUND")
	case dstEntry != nil:
		return sandboxFsError(409, "ALREADY_EXISTS")
	case dstUnderFile, strings.HasPrefix(dst, src+"/"):
		return sandboxFsError(400, "BAD_REQUEST")
	case sfs.entries[path.Dir(dst)] == nil:
		return sandboxFsError(404, "NOT_FOUND")
	}

	moved := map[string]*sandboxFsEntry{}
	prefix := src + "/"
	for name, entry := range sfs.entries {
		switch {
		case name == src:
			moved[dst] = entry
		case strings.HasPrefix(name, prefix):
			moved[dst+"/"+strings.TrimPrefix(name, prefix)] = entry
		default:
			continue
		}
		delete(sfs.entries, name)
	}
	maps.Copy(sfs.entries, moved)
	return Response{Body: sandbox.RenamePathResponse{}}
}

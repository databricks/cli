package testserver

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/databricks/databricks-sdk-go/service/workspace"
)

// SnapshotRootUUID is the workspace user UUID the snapshot root is under. The real API uses a
// UUID that is not the deploying identity's user name (an email or a service principal's
// application ID), so snapshot paths never contain the user name.
const SnapshotRootUUID = "3f9d2c1b-7a6e-4b5d-8c4f-0e1a2b3c4d5e"

func init() {
	registerRoutes("snapshots", addSnapshotsRoutes)
}

func addSnapshotsRoutes(server *Server) {
	server.Handle("GET", "/api/2.0/repos/snapshots/rootpath", func(req Request) any {
		return map[string]any{
			"path": "/Workspace/Users/" + SnapshotRootUUID + "/.snapshots/",
		}
	})

	server.Handle("POST", "/api/2.0/snapshots", func(req Request) any {
		contentType := req.Headers.Get("Content-Type")
		mediaType, params, err := mime.ParseMediaType(contentType)
		if err != nil || !strings.HasPrefix(mediaType, "multipart/") {
			return Response{StatusCode: http.StatusBadRequest}
		}

		mr := multipart.NewReader(bytes.NewReader(req.Body), params["boundary"])
		var relativePath string
		var acl, canManage []map[string]any
		for {
			p, err := mr.NextPart()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				return Response{StatusCode: http.StatusInternalServerError}
			}
			data, err := io.ReadAll(p)
			if err != nil {
				return Response{StatusCode: http.StatusInternalServerError}
			}
			switch p.FormName() {
			case "relative_path":
				relativePath = string(data)
			case "access_control_list":
				_ = json.Unmarshal(data, &acl)
			case "can_manage_principals":
				_ = json.Unmarshal(data, &canManage)
			}
		}

		// The server stores content under a fixed "snapshot" subfolder of the caller's
		// relative_path. The real API uses the workspace user UUID (not email) in the path,
		// see SnapshotRootUUID.
		contentPath := fmt.Sprintf("/Workspace/Users/%s/.snapshots/%s/snapshot", SnapshotRootUUID, relativePath)
		// Snapshots are immutable, so a relative_path can be created only once. The real API
		// answers with a 400 that carries the code in the message and sets no error_code.
		if _, exists := req.Workspace.SnapshotInfo(contentPath); exists {
			return Response{
				StatusCode: http.StatusBadRequest,
				Body: map[string]string{
					"message": "RESOURCE_ALREADY_EXISTS: relative_path already exists",
				},
			}
		}
		req.Workspace.WorkspaceMkdirs(workspace.Mkdirs{Path: contentPath})
		req.Workspace.RecordSnapshot(contentPath, relativePath, acl, canManage)
		return map[string]any{
			"name":     "workspaces/snapshotOperations/" + relativePath,
			"done":     true,
			"snapshot": map[string]any{"path": contentPath},
		}
	})

	// Returns the create-request fields (ACL, can_manage_principals) stored for the snapshot
	// at the given relative_path. Intended for acceptance test assertions.
	server.Handle("GET", "/api/2.0/snapshots/info", func(req Request) any {
		relPath := req.URL.Query().Get("relative_path")
		// Build the content path the same way the create handler does.
		contentPath := fmt.Sprintf("/Workspace/Users/%s/.snapshots/%s/snapshot", SnapshotRootUUID, relPath)
		rec, ok := req.Workspace.SnapshotInfo(contentPath)
		if !ok {
			return Response{
				StatusCode: http.StatusNotFound,
				Body: map[string]string{
					"error_code": "RESOURCE_DOES_NOT_EXIST",
					"message":    "Snapshot not found: " + relPath,
				},
			}
		}
		return rec
	})

	// Reports whether a snapshot's content was modified out of band.
	server.Handle("GET", "/api/2.0/snapshots:inspect", func(req Request) any {
		contentPath := req.URL.Query().Get("snapshot_content_path")
		dirty, ok := req.Workspace.SnapshotDirty(contentPath)
		// A path that holds no snapshot is not inspectable, so the API reports NOT_FOUND
		// rather than calling it clean.
		if !ok {
			return snapshotNotFound(contentPath)
		}
		return snapshotStatus(contentPath, dirty)
	})

	// Breaks the glass on a snapshot: grants write access to its content and marks it dirty.
	server.Handle("POST", "/api/2.0/snapshots:breakGlass", func(req Request) any {
		var request struct {
			SnapshotContentPath string `json:"snapshot_content_path"`
		}
		if err := json.Unmarshal(req.Body, &request); err != nil {
			return Response{
				Body:       fmt.Sprintf("internal error: %s", err),
				StatusCode: http.StatusInternalServerError,
			}
		}
		if !req.Workspace.BreakGlassSnapshot(request.SnapshotContentPath) {
			return snapshotNotFound(request.SnapshotContentPath)
		}
		return snapshotStatus(request.SnapshotContentPath, true)
	})
}

// snapshotRecord holds the create-request fields and the out-of-band modification flag for one
// immutable snapshot. Its JSON tags match what the accept test's info endpoint returns so the
// test script can call the endpoint and assert the fields with jq.
type snapshotRecord struct {
	Dirty     bool             `json:"dirty"`
	RelPath   string           `json:"relative_path"`
	ACL       []map[string]any `json:"access_control_list"`
	CanManage []map[string]any `json:"can_manage_principals"`
}

// RecordSnapshot stores a create-request record for the snapshot at contentPath.
func (s *FakeWorkspace) RecordSnapshot(contentPath, relPath string, acl, canManage []map[string]any) {
	defer s.LockUnlock()()
	s.snapshots[contentPath] = &snapshotRecord{
		Dirty:     false,
		RelPath:   relPath,
		ACL:       acl,
		CanManage: canManage,
	}
}

// SnapshotDirty reports whether the immutable snapshot at contentPath was modified out of band,
// and whether a snapshot is recorded there at all.
func (s *FakeWorkspace) SnapshotDirty(contentPath string) (dirty, ok bool) {
	defer s.LockUnlock()()
	rec, ok := s.snapshots[contentPath]
	if !ok {
		return false, false
	}
	return rec.Dirty, true
}

// SnapshotInfo returns the stored create-request record for the snapshot at contentPath.
// ok is false when no snapshot is recorded there.
func (s *FakeWorkspace) SnapshotInfo(contentPath string) (*snapshotRecord, bool) {
	defer s.LockUnlock()()
	rec, ok := s.snapshots[contentPath]
	return rec, ok
}

// BreakGlassSnapshot marks the snapshot at contentPath dirty. The real API flips the flag when
// the glass is broken, not when a file changes: until then the content is read-only to callers.
// ok is false when no snapshot is recorded there.
func (s *FakeWorkspace) BreakGlassSnapshot(contentPath string) (ok bool) {
	defer s.LockUnlock()()
	rec, ok := s.snapshots[contentPath]
	if ok {
		rec.Dirty = true
	}
	return ok
}

// snapshotStatus is the body the snapshot inspect and break glass APIs answer with.
func snapshotStatus(contentPath string, dirty bool) map[string]any {
	return map[string]any{
		"status": map[string]any{
			"snapshot_content_path": contentPath,
			"dirty":                 dirty,
			"permissions":           []string{"SNAPSHOT_PERMISSION_CAN_BREAK_GLASS"},
		},
	}
}

// snapshotNotFound is how the snapshot inspect and break glass APIs answer for a path that
// holds no snapshot.
func snapshotNotFound(contentPath string) Response {
	return Response{
		StatusCode: http.StatusNotFound,
		Body: map[string]string{
			"error_code": "RESOURCE_DOES_NOT_EXIST",
			"message":    "Snapshot not found: " + contentPath,
		},
	}
}

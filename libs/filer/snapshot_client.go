package filer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/textproto"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/client"
)

// SnapshotInfo holds the result of a successful snapshot upload.
type SnapshotInfo struct {
	// Path is the immutable workspace path for the uploaded snapshot content.
	Path string
}

// SnapshotUploader abstracts the /api/2.0/repos/snapshots endpoint.
// snapshotID is the content-addressed key supplied by the caller; the API uses
// it as the final path component so that identical content always resolves to
// the same workspace location.
// This interface exists so the implementation can later be replaced with a Go SDK call.
type SnapshotUploader interface {
	Upload(ctx context.Context, bundleID, snapshotID, currentUser string, zipContent []byte) (*SnapshotInfo, error)
}

// snapshotAPIClient implements SnapshotUploader against /api/2.0/repos/snapshots.
type snapshotAPIClient struct {
	apiClient apiClient
}

// snapshotUploadResponse mirrors the /api/2.0/repos/snapshots response body.
type snapshotUploadResponse struct {
	Snapshot struct {
		Path string `json:"path"`
	} `json:"snapshot"`
}

// NewSnapshotUploader creates a SnapshotUploader backed by /api/2.0/repos/snapshots.
func NewSnapshotUploader(w *databricks.WorkspaceClient) (SnapshotUploader, error) {
	apiClient, err := client.New(w.Config)
	if err != nil {
		return nil, err
	}
	return &snapshotAPIClient{apiClient: apiClient}, nil
}

// Upload uploads zipContent as an immutable snapshot identified by snapshotID.
// snapshotID is the SHA-256 of the files-only zip and is used by the server as
// the content-addressed path component. currentUser is granted CAN_READ on the snapshot.
func (c *snapshotAPIClient) Upload(ctx context.Context, bundleID, snapshotID, currentUser string, zipContent []byte) (*SnapshotInfo, error) {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)

	if err := mw.WriteField("snapshot_id", snapshotID); err != nil {
		return nil, fmt.Errorf("failed to write snapshot_id: %w", err)
	}
	if err := mw.WriteField("bundle_id", bundleID); err != nil {
		return nil, fmt.Errorf("failed to write bundle_id: %w", err)
	}

	// The API requires an access_control_list granting the current user read access.
	acl, err := json.Marshal([]map[string]string{
		{"user_name": currentUser, "permission_level": "CAN_READ"},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal access_control_list: %w", err)
	}
	if err := mw.WriteField("access_control_list", string(acl)); err != nil {
		return nil, fmt.Errorf("failed to write access_control_list: %w", err)
	}

	// Attach the zip with an explicit content-type so the server treats it as binary.
	fh := make(textproto.MIMEHeader)
	fh.Set("Content-Disposition", `form-data; name="file"; filename="snapshot.zip"`)
	fh.Set("Content-Type", "application/zip")
	part, err := mw.CreatePart(fh)
	if err != nil {
		return nil, fmt.Errorf("failed to create file part: %w", err)
	}
	if _, err := part.Write(zipContent); err != nil {
		return nil, fmt.Errorf("failed to write zip content: %w", err)
	}
	if err := mw.Close(); err != nil {
		return nil, fmt.Errorf("failed to finalize multipart body: %w", err)
	}

	headers := map[string]string{
		"Content-Type": mw.FormDataContentType(),
	}

	var resp snapshotUploadResponse
	err = c.apiClient.Do(ctx, http.MethodPost, "/api/2.0/repos/snapshots", headers, nil, body.Bytes(), &resp)
	if err != nil {
		return nil, fmt.Errorf("snapshot upload: %w", err)
	}

	return &SnapshotInfo{Path: resp.Snapshot.Path}, nil
}

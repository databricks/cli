package snapshot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"path"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/databricks-sdk-go"
	databricksclient "github.com/databricks/databricks-sdk-go/client"
)

// SnapshotInfo holds the result of a successful snapshot upload.
type SnapshotInfo struct {
	// Path is the immutable workspace path for the uploaded snapshot content.
	Path string
}

// ACLEntry is one element of the access_control_list sent to the snapshot API.
// All entries are granted CAN_READ; the snapshot API does not support other levels.
type ACLEntry struct {
	UserName             string `json:"user_name,omitempty"`
	GroupName            string `json:"group_name,omitempty"`
	ServicePrincipalName string `json:"service_principal_name,omitempty"`
	PermissionLevel      string `json:"permission_level"`
}

// SnapshotClient implements the /api/2.0/repos/snapshots endpoint.
type SnapshotClient struct {
	workspaceClient *databricks.WorkspaceClient
	client          *databricksclient.DatabricksClient
}

// snapshotUploadResponse mirrors the /api/2.0/repos/snapshots response body.
type snapshotUploadResponse struct {
	Snapshot struct {
		Path string `json:"path"`
	} `json:"snapshot"`
}

type snapshotRootPathResponse struct {
	Path string `json:"path"`
}

// NewSnapshotClient creates a SnapshotClient backed by /api/2.0/repos/snapshots.
func NewSnapshotClient(w *databricks.WorkspaceClient) (*SnapshotClient, error) {
	c, err := databricksclient.New(w.Config)
	if err != nil {
		return nil, err
	}
	return &SnapshotClient{workspaceClient: w, client: c}, nil
}

// Upload uploads zipContent as an immutable snapshot. The server derives the
// content-addressed path from the SHA-256 of the zip, and acl grants CAN_READ to
// each listed principal. path is reserved for forthcoming API support for a
// caller-specified upload location; it is not sent yet.
func (c *SnapshotClient) Upload(ctx context.Context, path, bundleID string, acl []ACLEntry, zipContent []byte) (*SnapshotInfo, error) {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)

	if err := mw.WriteField("snapshot_id", HashFromContent(zipContent)); err != nil {
		return nil, fmt.Errorf("failed to write snapshot_id: %w", err)
	}
	if err := mw.WriteField("bundle_id", bundleID); err != nil {
		return nil, fmt.Errorf("failed to write bundle_id: %w", err)
	}

	aclJSON, err := json.Marshal(acl)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal access_control_list: %w", err)
	}
	if err := mw.WriteField("access_control_list", string(aclJSON)); err != nil {
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

	// Workspace routing header is required so the server can locate the correct
	// ASP (application service principal) that owns the snapshot directory.
	headers := auth.WorkspaceIDHeaders(c.client.Config)
	if headers == nil {
		headers = make(map[string]string)
	}
	headers["Content-Type"] = mw.FormDataContentType()

	var resp snapshotUploadResponse
	err = c.client.Do(ctx, http.MethodPost, "/api/2.0/repos/snapshots", headers, nil, body.Bytes(), &resp)
	if err != nil {
		return nil, fmt.Errorf("snapshot upload: %w", err)
	}

	return &SnapshotInfo{Path: resp.Snapshot.Path}, nil
}

func (c *SnapshotClient) Get(ctx context.Context, snapshotRelativePath string) (*SnapshotInfo, error) {
	rootPath, err := c.GetSnapshotRootPath(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshot root path: %w", err)
	}
	snapshotPath := path.Join(rootPath, snapshotRelativePath)
	resp, err := c.workspaceClient.Workspace.GetStatusByPath(ctx, snapshotPath)
	if err != nil {
		return nil, fmt.Errorf("snapshot get: %w", err)
	}
	return &SnapshotInfo{Path: resp.Path}, nil
}

func (c *SnapshotClient) GetSnapshotRootPath(ctx context.Context) (string, error) {
	var resp snapshotRootPathResponse
	err := c.client.Do(ctx, http.MethodGet, "/api/2.0/repos/snapshots/rootpath", auth.WorkspaceIDHeaders(c.client.Config), nil, nil, &resp)
	if err != nil {
		return "", fmt.Errorf("snapshot root path get: %w", err)
	}
	return path.Clean(resp.Path), nil
}

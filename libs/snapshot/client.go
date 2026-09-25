package snapshot

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"path"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/databricks-sdk-go"
	databricksclient "github.com/databricks/databricks-sdk-go/client"
)

// ContentSubdir is the fixed subfolder the snapshot API appends to a caller-supplied
// relative_path to form the immutable content path (e.g. "<root>/<relative_path>/snapshot").
// The content path is what the create API echoes back and what InspectSnapshot inspects.
const ContentSubdir = "snapshot"

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

// ManagePrincipal is one element of can_manage_principals: a principal allowed to break the
// glass on a snapshot, i.e. to modify otherwise-immutable content.
type ManagePrincipal struct {
	UserName             string `json:"user_name,omitempty"`
	GroupName            string `json:"group_name,omitempty"`
	ServicePrincipalName string `json:"service_principal_name,omitempty"`
}

// SnapshotStatus is the status of a snapshot's content path, as reported by InspectSnapshot.
// Dirty is true when the immutable content was modified out of band.
type SnapshotStatus struct {
	Dirty bool
}

// SnapshotClient implements the /api/2.0/snapshots endpoint.
type SnapshotClient struct {
	workspaceClient *databricks.WorkspaceClient
	client          *databricksclient.DatabricksClient
}

// snapshotUploadResponse mirrors the /api/2.0/snapshots response body. Only the content
// path is used; the operation name, done flag and creation metadata are ignored.
type snapshotUploadResponse struct {
	Snapshot struct {
		Path string `json:"path"`
	} `json:"snapshot"`
}

type snapshotRootPathResponse struct {
	Path string `json:"path"`
}

// inspectSnapshotPath reports whether a snapshot's content was modified out of band.
const inspectSnapshotPath = "/api/2.0/snapshots:inspect"

// inspectSnapshotRequest is the body of an inspect call.
type inspectSnapshotRequest struct {
	SnapshotContentPath string `json:"snapshot_content_path"`
}

// inspectSnapshotResponse mirrors the /api/2.0/snapshots:inspect response body. The
// permissions the status also carries are not used.
type inspectSnapshotResponse struct {
	Status *struct {
		SnapshotContentPath string `json:"snapshot_content_path"`
		Dirty               bool   `json:"dirty"`
	} `json:"status"`
}

// NewSnapshotClient creates a SnapshotClient backed by /api/2.0/snapshots.
func NewSnapshotClient(w *databricks.WorkspaceClient) (*SnapshotClient, error) {
	c, err := databricksclient.New(w.Config)
	if err != nil {
		return nil, err
	}
	return &SnapshotClient{workspaceClient: w, client: c}, nil
}

// Upload uploads zipContent as an immutable snapshot at relativePath. The server stores the
// content under "<root>/<relativePath>/snapshot" and returns that content path. acl grants
// CAN_READ to each listed principal; canManage lists the principals allowed to break the
// glass on the snapshot.
func (c *SnapshotClient) Upload(ctx context.Context, relativePath string, acl []ACLEntry, canManage []ManagePrincipal, zipContent []byte) (*SnapshotInfo, error) {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)

	if err := mw.WriteField("relative_path", relativePath); err != nil {
		return nil, fmt.Errorf("failed to write relative_path: %w", err)
	}

	aclJSON, err := json.Marshal(acl)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal access_control_list: %w", err)
	}
	if err := mw.WriteField("access_control_list", string(aclJSON)); err != nil {
		return nil, fmt.Errorf("failed to write access_control_list: %w", err)
	}

	// Always send an array, never null, so "no break-glass managers" is unambiguous.
	if canManage == nil {
		canManage = []ManagePrincipal{}
	}
	canManageJSON, err := json.Marshal(canManage)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal can_manage_principals: %w", err)
	}
	if err := mw.WriteField("can_manage_principals", string(canManageJSON)); err != nil {
		return nil, fmt.Errorf("failed to write can_manage_principals: %w", err)
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
	err = c.client.Do(ctx, http.MethodPost, "/api/2.0/snapshots", headers, nil, body.Bytes(), &resp)
	if err != nil {
		return nil, fmt.Errorf("snapshot upload: %w", err)
	}

	return &SnapshotInfo{Path: resp.Snapshot.Path}, nil
}

// InspectSnapshot reports the status of the snapshot content at contentPath, which tells the
// caller whether the immutable content was modified out of band.
func (c *SnapshotClient) InspectSnapshot(ctx context.Context, contentPath string) (*SnapshotStatus, error) {
	headers := auth.WorkspaceIDHeaders(c.client.Config)
	if headers == nil {
		headers = make(map[string]string)
	}
	headers["Content-Type"] = "application/json"

	payload, err := json.Marshal(inspectSnapshotRequest{SnapshotContentPath: contentPath})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal inspect request: %w", err)
	}

	var resp inspectSnapshotResponse
	err = c.client.Do(ctx, http.MethodGet, inspectSnapshotPath, headers, nil, nil, &resp, withJSONBody(payload))
	if err != nil {
		return nil, fmt.Errorf("snapshot inspect: %w", err)
	}
	// A 200 without a status would otherwise read as "not modified", silently disabling the
	// check, so treat it as an error instead.
	if resp.Status == nil {
		return nil, errors.New("snapshot inspect: response contained no status")
	}

	return &SnapshotStatus{Dirty: resp.Status.Dirty}, nil
}

// withJSONBody returns a request visitor that sends payload as the request body.
//
// The inspect RPC is a GET that carries its arguments in a JSON body, which the SDK cannot
// express: for GET it serializes the request value into the query string and sends an empty
// body (see makeRequestBody in databricks-sdk-go/httpclient/request.go). Visitors run against
// a freshly built request on every attempt, so installing the body here is retry-safe.
func withJSONBody(payload []byte) func(*http.Request) error {
	return func(r *http.Request) error {
		r.Body = io.NopCloser(bytes.NewReader(payload))
		r.ContentLength = int64(len(payload))
		return nil
	}
}

func (c *SnapshotClient) Get(ctx context.Context, snapshotRelativePath string) (*SnapshotInfo, error) {
	rootPath, err := c.GetSnapshotRootPath(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshot root path: %w", err)
	}
	snapshotPath := path.Join(rootPath, snapshotRelativePath, ContentSubdir)
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

// Package workspaceapi is a client for the workspace objects API
// (/api/2.0/workspace/...) and workspace object permissions, which sdk-go does
// not provide.
package workspaceapi

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"mime/multipart"
	"strings"

	"github.com/databricks/cli/libs/apiclient"
	"github.com/databricks/cli/libs/apierrs"
	"github.com/databricks/cli/libs/listing"
)

// WorkspaceInterface is the method set of [Client], for callers that accept
// a fake in tests.
type WorkspaceInterface interface {
	Delete(ctx context.Context, request Delete) error
	Export(ctx context.Context, request ExportRequest) (*ExportResponse, error)
	GetPermissionLevels(ctx context.Context, request GetWorkspaceObjectPermissionLevelsRequest) (*GetWorkspaceObjectPermissionLevelsResponse, error)
	GetPermissions(ctx context.Context, request GetWorkspaceObjectPermissionsRequest) (*WorkspaceObjectPermissions, error)
	GetStatus(ctx context.Context, request GetStatusRequest) (*ObjectInfo, error)
	Import(ctx context.Context, request Import) error
	List(ctx context.Context, request ListWorkspaceRequest) listing.Iterator[ObjectInfo]
	ListAll(ctx context.Context, request ListWorkspaceRequest) ([]ObjectInfo, error)
	Mkdirs(ctx context.Context, request Mkdirs) error
	SetPermissions(ctx context.Context, request WorkspaceObjectPermissionsRequest) (*WorkspaceObjectPermissions, error)
	UpdatePermissions(ctx context.Context, request WorkspaceObjectPermissionsRequest) (*WorkspaceObjectPermissions, error)

	GetStatusByPath(ctx context.Context, path string) (*ObjectInfo, error)
	MkdirsByPath(ctx context.Context, path string) error
	GetByPath(ctx context.Context, name string) (*ObjectInfo, error)
	ObjectInfoPathToObjectIdMap(ctx context.Context, request ListWorkspaceRequest) (map[string]int64, error)
	GetPermissionLevelsByWorkspaceObjectTypeAndWorkspaceObjectId(ctx context.Context, workspaceObjectType, workspaceObjectId string) (*GetWorkspaceObjectPermissionLevelsResponse, error)
	GetPermissionsByWorkspaceObjectTypeAndWorkspaceObjectId(ctx context.Context, workspaceObjectType, workspaceObjectId string) (*WorkspaceObjectPermissions, error)
	RecursiveList(ctx context.Context, path string) ([]ObjectInfo, error)
	Upload(ctx context.Context, path string, r io.Reader, opts ...UploadOption) error
	WriteFile(ctx context.Context, name string, data []byte) error
	Download(ctx context.Context, path string, opts ...DownloadOption) (io.ReadCloser, error)
	ReadFile(ctx context.Context, name string) ([]byte, error)
}

// Client calls the workspace objects API.
type Client struct {
	workspaceImpl
}

var _ WorkspaceInterface = (*Client)(nil)

func New(c *apiclient.Client) *Client {
	return &Client{workspaceImpl{client: c}}
}

func addWorkspaceID(c *apiclient.Client, headers map[string]string) {
	c.AddWorkspaceIDHeader(headers)
}

var b64 = base64.StdEncoding

// Bytes returns the decoded content of an export.
func (r *ExportResponse) Bytes() ([]byte, error) {
	return b64.DecodeString(r.Content)
}

func (a *Client) GetPermissionLevelsByWorkspaceObjectTypeAndWorkspaceObjectId(ctx context.Context, workspaceObjectType, workspaceObjectId string) (*GetWorkspaceObjectPermissionLevelsResponse, error) {
	return a.GetPermissionLevels(ctx, GetWorkspaceObjectPermissionLevelsRequest{
		WorkspaceObjectType: workspaceObjectType,
		WorkspaceObjectId:   workspaceObjectId,
	})
}

func (a *Client) GetPermissionsByWorkspaceObjectTypeAndWorkspaceObjectId(ctx context.Context, workspaceObjectType, workspaceObjectId string) (*WorkspaceObjectPermissions, error) {
	return a.GetPermissions(ctx, GetWorkspaceObjectPermissionsRequest{
		WorkspaceObjectType: workspaceObjectType,
		WorkspaceObjectId:   workspaceObjectId,
	})
}

// GetStatusByPath gets the status of an object or a directory.
func (a *Client) GetStatusByPath(ctx context.Context, path string) (*ObjectInfo, error) {
	return a.GetStatus(ctx, GetStatusRequest{Path: path})
}

// MkdirsByPath creates the directory and its missing parents.
func (a *Client) MkdirsByPath(ctx context.Context, path string) error {
	return a.Mkdirs(ctx, Mkdirs{Path: path})
}

// ObjectInfoPathToObjectIdMap lists objects and maps their paths to IDs.
func (a *Client) ObjectInfoPathToObjectIdMap(ctx context.Context, request ListWorkspaceRequest) (map[string]int64, error) {
	mapping := map[string]int64{}
	result, err := a.ListAll(ctx, request)
	if err != nil {
		return nil, err
	}
	for _, v := range result {
		if _, duplicate := mapping[v.Path]; duplicate {
			return nil, fmt.Errorf("duplicate .Path: %s", v.Path)
		}
		mapping[v.Path] = v.ObjectId
	}
	return mapping, nil
}

// GetByPath lists the workspace root and returns the object with the given path.
func (a *Client) GetByPath(ctx context.Context, name string) (*ObjectInfo, error) {
	result, err := a.ListAll(ctx, ListWorkspaceRequest{})
	if err != nil {
		return nil, err
	}
	var alternatives []ObjectInfo
	for _, v := range result {
		if v.Path == name {
			alternatives = append(alternatives, v)
		}
	}
	if len(alternatives) == 0 {
		return nil, fmt.Errorf("ObjectInfo named '%s' does not exist", name)
	}
	if len(alternatives) > 1 {
		return nil, fmt.Errorf("there are %d instances of ObjectInfo named '%s'", len(alternatives), name)
	}
	return &alternatives[0], nil
}

// RecursiveList traverses the workspace tree and returns all non-directory
// objects under the path.
func (a *Client) RecursiveList(ctx context.Context, path string) ([]ObjectInfo, error) {
	var results []ObjectInfo
	queue := []string{path}
	for len(queue) > 0 {
		path := queue[0]
		queue = queue[1:]
		batch, err := a.ListAll(ctx, ListWorkspaceRequest{Path: path})
		if apierrs.IsMissing(err) {
			// skip on path deleted during iteration
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("list %s: %w", path, err)
		}
		for _, v := range batch {
			if v.ObjectType == ObjectTypeDirectory {
				queue = append(queue, v.Path)
				continue
			}
			results = append(results, v)
		}
	}
	return results, nil
}

type UploadOption = func(*Import)

func UploadOverwrite() UploadOption {
	return func(i *Import) {
		i.Overwrite = true
	}
}

func UploadLanguage(l Language) UploadOption {
	return func(i *Import) {
		i.Language = l
	}
}

func UploadFormat(f ImportFormat) UploadOption {
	return func(i *Import) {
		i.Format = f
	}
}

// Upload imports a workspace object (for example, a notebook or file) or the
// contents of an entire directory (`DBC` format) with a multipart request.
//
// The format defaults to SOURCE; for SOURCE the language is inferred from the
// path's extension unless UploadLanguage is given.
func (a *Client) Upload(ctx context.Context, path string, r io.Reader, opts ...UploadOption) error {
	buf := &bytes.Buffer{}
	w := multipart.NewWriter(buf)
	err := w.WriteField("path", path)
	if err != nil {
		return fmt.Errorf("write path: %w", err)
	}
	content, err := w.CreateFormFile("content", "content")
	if err != nil {
		return fmt.Errorf("write content: %w", err)
	}
	_, err = io.Copy(content, r)
	if err != nil {
		return fmt.Errorf("copy io: %w", err)
	}
	i := &Import{}
	for _, v := range opts {
		v(i)
	}
	if i.Format == "" || i.Format == ImportFormatSource {
		for sfx, lang := range map[string]Language{
			".py":    LanguagePython,
			".sql":   LanguageSql,
			".scala": LanguageScala,
			".R":     LanguageR,
		} {
			if !strings.HasSuffix(path, sfx) {
				continue
			}
			i.Language = lang
		}
	}
	if i.Format != "" {
		err = w.WriteField("format", i.Format.String())
		if err != nil {
			return fmt.Errorf("write format: %w", err)
		}
	}
	if i.Language != "" {
		err = w.WriteField("language", i.Language.String())
		if err != nil {
			return fmt.Errorf("write language: %w", err)
		}
	}
	if i.Overwrite {
		err = w.WriteField("overwrite", "true")
		if err != nil {
			return fmt.Errorf("write overwrite: %w", err)
		}
	}
	err = w.Close()
	if err != nil {
		return fmt.Errorf("write close: %w", err)
	}
	headers := map[string]string{
		"Content-Type": w.FormDataContentType(),
	}
	addWorkspaceID(a.client, headers)
	return a.client.Do(ctx, "POST", "/api/2.0/workspace/import", headers, nil, buf.Bytes(), nil)
}

// WriteFile is identical to [os.WriteFile] but for workspace files. It always
// overwrites and uploads a file, not a notebook.
func (a *Client) WriteFile(ctx context.Context, name string, data []byte) error {
	return a.Upload(ctx, name, bytes.NewBuffer(data),
		UploadFormat(ImportFormatAuto),
		UploadOverwrite())
}

type DownloadOption = func(q map[string]any)

func DownloadFormat(f ExportFormat) DownloadOption {
	return func(q map[string]any) {
		q["format"] = f.String()
	}
}

// Download returns the contents of a notebook or file. The format defaults to
// SOURCE.
func (a *Client) Download(ctx context.Context, path string, opts ...DownloadOption) (io.ReadCloser, error) {
	var buf bytes.Buffer
	query := map[string]any{"path": path, "direct_download": true}
	for _, v := range opts {
		v(query)
	}
	headers := map[string]string{"Content-Type": "application/json"}
	addWorkspaceID(a.client, headers)
	err := a.client.Do(ctx, "GET", "/api/2.0/workspace/export", headers, nil, query, &buf)
	if err != nil {
		return nil, err
	}
	return io.NopCloser(&buf), nil
}

// ReadFile is identical to [os.ReadFile] but for workspace files.
func (a *Client) ReadFile(ctx context.Context, name string) ([]byte, error) {
	b, err := a.Download(ctx, name)
	if err != nil {
		return nil, err
	}
	defer b.Close()
	return io.ReadAll(b)
}

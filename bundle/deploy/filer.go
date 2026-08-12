package deploy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/databricks-sdk-go/client"
)

// FilerFactory is a function that returns a filer.Filer.
type FilerFactory func(ctx context.Context, b *bundle.Bundle) (filer.Filer, error)

type stateFiler struct {
	filer filer.Filer

	apiClient *client.DatabricksClient
	root      filer.WorkspaceRootPath
}

func (s stateFiler) Delete(ctx context.Context, path string, mode ...filer.DeleteMode) error {
	return s.filer.Delete(ctx, path, mode...)
}

// Mkdir implements filer.Filer.
func (s stateFiler) Mkdir(ctx context.Context, path string) error {
	return s.filer.Mkdir(ctx, path)
}

func (s stateFiler) Read(ctx context.Context, path string) (io.ReadCloser, error) {
	absPath, err := s.root.Join(path)
	if err != nil {
		return nil, err
	}

	stat, err := s.Stat(ctx, path)
	if err != nil {
		return nil, err
	}
	if stat.IsDir() {
		return nil, fmt.Errorf("not a file: %s", absPath)
	}

	var buf bytes.Buffer
	// Read via the raw apiClient.Do (not the SDK's Workspace.Download) so
	// auth.WorkspaceIDHeaders can drop the CLI-only "none" workspace-id sentinel
	// that Download would send literally. See PR #6149 for the write-path equivalent.
	urlPath := "/api/2.0/workspace/export?path=" + url.QueryEscape(absPath) + "&direct_download=true"
	err = s.apiClient.Do(ctx, http.MethodGet, urlPath, auth.WorkspaceIDHeaders(s.apiClient.Config), nil, nil, &buf)
	if err != nil {
		return nil, err
	}

	return io.NopCloser(&buf), nil
}

func (s stateFiler) ReadDir(ctx context.Context, path string) ([]fs.DirEntry, error) {
	return s.filer.ReadDir(ctx, path)
}

func (s stateFiler) Stat(ctx context.Context, name string) (fs.FileInfo, error) {
	return s.filer.Stat(ctx, name)
}

func (s stateFiler) Write(ctx context.Context, path string, reader io.Reader, mode ...filer.WriteMode) error {
	return s.filer.Write(ctx, path, reader, mode...)
}

// StateFiler returns a filer.Filer that can be used to read/write state files.
// Reads use the streaming /workspace/export API, which is officially supported,
// scoped, and streams state files well beyond the 10 MB JSON export limit.
func StateFiler(ctx context.Context, b *bundle.Bundle) (filer.Filer, error) {
	f, err := filer.NewWorkspaceFilesClient(b.WorkspaceClient(ctx), b.Config.Workspace.StatePath)
	if err != nil {
		return nil, err
	}

	apiClient, err := client.New(b.WorkspaceClient(ctx).Config)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	return stateFiler{
		filer:     f,
		root:      filer.NewWorkspaceRootPath(b.Config.Workspace.StatePath),
		apiClient: apiClient,
	}, nil
}

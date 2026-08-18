package deploy

import (
	"context"
	"fmt"
	"io"
	"io/fs"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/databricks-sdk-go"
)

// FilerFactory is a function that returns a filer.Filer.
type FilerFactory func(ctx context.Context, b *bundle.Bundle) (filer.Filer, error)

type stateFiler struct {
	filer filer.Filer

	workspaceClient *databricks.WorkspaceClient
	root            filer.WorkspaceRootPath
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

	// Stream via the SDK's /workspace/export (direct_download=true); no 10 MB cap.
	return s.workspaceClient.Workspace.Download(ctx, absPath)
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
	w, err := stateWorkspaceClient(ctx, b)
	if err != nil {
		return nil, err
	}

	f, err := filer.NewWorkspaceFilesClient(w, b.Config.Workspace.StatePath)
	if err != nil {
		return nil, err
	}

	return stateFiler{
		filer:           f,
		root:            filer.NewWorkspaceRootPath(b.Config.Workspace.StatePath),
		workspaceClient: w,
	}, nil
}

// stateWorkspaceClient returns the bundle's workspace client, cloned with the CLI-only
// "none" workspace-id sentinel stripped so Workspace.Download does not forward it as the
// X-Databricks-Workspace-Id routing header.
func stateWorkspaceClient(ctx context.Context, b *bundle.Bundle) (*databricks.WorkspaceClient, error) {
	w := b.WorkspaceClient(ctx)
	if w.Config.WorkspaceID != auth.WorkspaceIDNone {
		return w, nil
	}

	// config.Config embeds a sync.Mutex, so use the SDK's copylocks-safe clone.
	cfg, err := w.Config.NewWithWorkspaceHost(w.Config.Host)
	if err != nil {
		return nil, err
	}
	cfg.WorkspaceID = ""
	return databricks.NewWorkspaceClient((*databricks.Config)(cfg))
}

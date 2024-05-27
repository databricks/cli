package filer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"strings"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/workspace"
)

// TODO: Ensure the root path is correctly configured in the workspace files client
// and the local client.
// TODO: Better documentation here regarding the boundary conditions and what
// exactly is this filer doing.
type workspaceFuseClient struct {
	workspaceFilesClient Filer
	localClient          Filer
}

type notebookExtension string

const (
	notebookExtensionPython notebookExtension = ".py"
	notebookExtensionR      notebookExtension = ".r"
	notebookExtensionScala  notebookExtension = ".scala"
	notebookExtensionSql    notebookExtension = ".sql"
)

func stripNotebookExtension(name string) (string, bool) {
	switch {
	case strings.HasSuffix(name, string(notebookExtensionPython)):
		return strings.TrimSuffix(name, string(notebookExtensionPython)), true
	case strings.HasSuffix(name, string(notebookExtensionR)):
		return strings.TrimSuffix(name, string(notebookExtensionR)), true
	case strings.HasSuffix(name, string(notebookExtensionScala)):
		return strings.TrimSuffix(name, string(notebookExtensionScala)), true
	case strings.HasSuffix(name, string(notebookExtensionSql)):
		return strings.TrimSuffix(name, string(notebookExtensionSql)), true
	default:
		return name, false
	}
}

func extensionForLanguage(l workspace.Language) notebookExtension {
	switch l {
	case workspace.LanguagePython:
		return ".py"
	case workspace.LanguageR:
		return ".r"
	case workspace.LanguageScala:
		return ".scala"
	case workspace.LanguageSql:
		return ".sql"
	default:
		return ""
	}
}

// TODO: Show a more informative error upstream, relaying that the project is
// not supported for DABs in the workspace.
type dupPathError struct {
	oi1 workspace.ObjectInfo
	oi2 workspace.ObjectInfo

	commonPath string
}

func (e dupPathError) Error() string {
	return fmt.Sprintf("duplicate paths. Both %s at %s and %s at %s have the same path %s on the local FUSE mount.", e.oi1.ObjectType, e.oi1.Path, e.oi2.ObjectType, e.oi2.Path, e.commonPath)
}

// TODO: Should we have this condition of /Workspace prefix here? Would it make
// more sense to move it outside the function and make it the responsibility of
// the callsite to validate this?
func NewWorkspaceFuseClient(w *databricks.WorkspaceClient, root string) (Filer, error) {
	if !strings.HasPrefix(root, "/Workspace") {
		return nil, errors.New("workspace fuse root path must be under /Workspace")
	}

	// TODO: Ensure this trim prefix works as expected. Edge cases around empty path strings maybe?
	// TODO: Does this work if we do not trim the prefix? That might be preferable.
	wc, err := NewWorkspaceFilesClient(w, strings.TrimPrefix(root, "/Workspace"))
	if err != nil {
		return nil, err
	}

	lc, err := NewLocalClient(root)
	if err != nil {
		return nil, err
	}

	return &workspaceFuseClient{
		workspaceFilesClient: wc,
		localClient:          lc,
	}, nil
}

// TODO: Notebooks do not have a mod time in the fuse mount. Would incremental sync work
// for DABs? Does DABs even use incremental sync today?
// TODO: Run the common filer tests on this client.

func (w *workspaceFuseClient) ReadDir(ctx context.Context, name string) ([]fs.DirEntry, error) {
	entries, err := w.workspaceFilesClient.ReadDir(ctx, name)
	if err != nil {
		return nil, err
	}

	seenPaths := make(map[string]workspace.ObjectInfo)
	for i, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}
		sysInfo := info.Sys().(workspace.ObjectInfo)

		// Skip if the object is not a notebook
		if sysInfo.ObjectType != workspace.ObjectTypeNotebook {
			continue
		}

		// Add extension to local file path if the file is a notebook
		newEntry := entry.(wsfsDirEntry)
		newPath := newEntry.wsfsFileInfo.oi.Path + string(extensionForLanguage(sysInfo.Language))

		if _, ok := seenPaths[newPath]; ok {
			return nil, dupPathError{
				oi1:        seenPaths[newPath],
				oi2:        sysInfo,
				commonPath: newPath,
			}
		}
		seenPaths[newPath] = sysInfo

		// Mutate the entry to have the new path
		entries[i] = newEntry
	}

	return entries, nil
}

func (w *workspaceFuseClient) Write(ctx context.Context, name string, reader io.Reader, mode ...WriteMode) error {
	// Note: Any objects (including notebooks) will be written and represented files
	// on the filesystem.
	// (future opportunity) There will be usecases where we want to write other objects
	// (e.g. notebooks or dashboards) to wsfs. A good usecase is templates directly
	// creating notebooks or dashboards. When we have such usecases, we can add a
	// new WriteMode to handle them.
	return w.localClient.Write(ctx, name, reader, mode...)
}

func (w *workspaceFuseClient) Read(ctx context.Context, name string) (io.ReadCloser, error) {
	r, err := w.localClient.Read(ctx, name)

	// If the file is not found in the local file system, it might be a notebook.
	// In that case, we should try to read the notebook from the workspace.
	if errors.As(err, &FileDoesNotExistError{}) {
		stripName, ok := stripNotebookExtension(name)
		if !ok {
			return nil, err
		}

		// Check if the file with it's extension stripped is a notebook. If it is not a notebook,
		// return the original error.
		stat, err2 := w.workspaceFilesClient.Stat(ctx, stripName)
		if err2 != nil {
			return nil, err
		}
		if stat.Sys().(workspace.ObjectInfo).ObjectType != workspace.ObjectTypeNotebook {
			return nil, err
		}

		// Since the file is a notebook, we should read the notebook from the workspace.
		return w.workspaceFilesClient.Read(ctx, stripName)
	}
	return r, err
}

// TODO: will local filer be enough for a recursive delete? Can os.RemoveAll delete directories
// with notebooks and files
// TODO: Iterate on the delete semantics a bit. Actually iterate on all the semantics.

func (w *workspaceFuseClient) Delete(ctx context.Context, name string, mode ...DeleteMode) error {
	err := w.localClient.Delete(ctx, name, mode...)

	// If the file is not found in the local file system, it might be a notebook.
	if errors.As(err, &FileDoesNotExistError{}) {
		// If the file is not found in the local file system, it might be a notebook.
		// In that case, we should try to delete the notebook from the workspace.
		stripName, ok := stripNotebookExtension(name)
		if !ok {
			return err
		}

		// Check if the file with it's extension stripped is a notebook. If it is not a notebook,
		// return the original error.
		stat, err2 := w.workspaceFilesClient.Stat(ctx, stripName)
		if err2 != nil {
			return err
		}
		if stat.Sys().(workspace.ObjectInfo).ObjectType != workspace.ObjectTypeNotebook {
			return err
		}

		// Since the file is a notebook, we should delete the notebook from the workspace.
		return w.workspaceFilesClient.Delete(ctx, stripName, mode...)
	}

	return err
}

func (w *workspaceFuseClient) Stat(ctx context.Context, name string) (fs.FileInfo, error) {
	info, err := w.localClient.Stat(ctx, name)

	// If the file is not found in the local file system, it might be a notebook.
	if errors.As(err, &FileDoesNotExistError{}) {
		stripName, ok := stripNotebookExtension(name)
		if !ok {
			return nil, err
		}

		// Check if the file with it's extension stripped is a notebook. If it is not a notebook,
		// return the original error.
		stat, err2 := w.workspaceFilesClient.Stat(ctx, stripName)
		if err2 != nil {
			return nil, err
		}
		if stat.Sys().(workspace.ObjectInfo).ObjectType != workspace.ObjectTypeNotebook {
			return nil, err
		}

		// Since the file is a notebook, we should stat the notebook from the workspace.
		return stat, nil
	}

	return info, err
}

// TODO: Does this work as expected? Does fuse mount have full fidelity
// with a local file system as far as directories are concerned?
func (w *workspaceFuseClient) Mkdir(ctx context.Context, name string) error {
	return w.localClient.Mkdir(ctx, name)
}



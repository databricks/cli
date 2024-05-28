package filer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"slices"
	"strings"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/workspace"
)

// TODO: Better documentation here regarding the boundary conditions and what
// exactly is this filer doing.
type workspaceFuseClient struct {
	workspaceFilesClient Filer
}

type notebookExtension string

const (
	notebookExtensionPython notebookExtension = ".py"
	notebookExtensionR      notebookExtension = ".r"
	notebookExtensionScala  notebookExtension = ".scala"
	notebookExtensionSql    notebookExtension = ".sql"
	notebookExtensionIpynb  notebookExtension = ".ipynb"
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
	case strings.HasSuffix(name, string(notebookExtensionIpynb)):
		return strings.TrimSuffix(name, string(notebookExtensionIpynb)), true
	default:
		return name, false
	}
}

func extensionForLanguage(l workspace.Language) notebookExtension {
	switch l {
	case workspace.LanguagePython:
		// Note: We cannot differentiate between python source notebooks and ipynb
		// notebook based on the workspace files API. We'll thus assign all python
		// notebooks the .py extension.
		//
		// One notable consequence is that a notebook called foo and a file
		// called foo.ipynb can live side by side in the same directory in wsfs
		// and will be supported by DABs. Export such a setup to a Git repo might
		// fail though.
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

type DuplicatePathError struct {
	oi1 workspace.ObjectInfo
	oi2 workspace.ObjectInfo

	commonName string
}

func (e DuplicatePathError) Error() string {
	// TODO: better error message here.
	return fmt.Sprintf("duplicate paths. Both %s at %s and %s at %s resolve to the same name %s", e.oi1.ObjectType, e.oi1.Path, e.oi2.ObjectType, e.oi2.Path, e.commonName)
}

// This is a wrapper over the workspace files client that is used to access files in
// the workspace file system. It fixes the notebook extension problem when directly using
// the workspace files client (or the API directly).
//
// With this client, you can read, write, delete, and stat notebooks in the workspace,
// using their file names with the extension included.
// The listing of files will also include the extension for notebooks.
//
// This makes the workspace file system resemble a traditional file system more closely,
// allowing DABs to work from a DBR runtime.
//
// Usage Conditions:
// The methods this filer implements assumes that there are no objects with duplicate
// paths (with extension) in the file tree. That is both a file foo.py and a python notebook
// foo do not exist in the same directory.
// The ReadDir method will return an error if such a case is detected. Thus using
// the ReadDir method before other methods makes them safe to use.
func NewWorkspaceFuseClient(w *databricks.WorkspaceClient, root string) (Filer, error) {
	// TODO: Ensure this trim prefix works as expected. Edge cases around empty path strings maybe?
	// TODO: Does this work if we do not trim the prefix? That might be preferable.
	wc, err := NewWorkspaceFilesClient(w, root)
	if err != nil {
		return nil, err
	}

	return &workspaceFuseClient{
		workspaceFilesClient: wc,
	}, nil
}

// TODO: Notebooks do not have a mod time in the fuse mount. Would incremental sync work
// for DABs? Does DABs even use incremental sync today?

func (w *workspaceFuseClient) ReadDir(ctx context.Context, name string) ([]fs.DirEntry, error) {
	entries, err := w.workspaceFilesClient.ReadDir(ctx, name)
	if err != nil {
		return nil, err
	}

	// Sort the entries by name to ensure that the order and any resultant
	// errors are deterministic.
	slices.SortFunc(entries, func(a, b fs.DirEntry) int {
		return strings.Compare(a.Name(), b.Name())
	})

	seenPaths := make(map[string]workspace.ObjectInfo)
	for i, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}
		sysInfo := info.Sys().(workspace.ObjectInfo)

		// Skip if the object is not a notebook
		if sysInfo.ObjectType != workspace.ObjectTypeNotebook {
			if _, ok := seenPaths[entries[i].Name()]; ok {
				return nil, DuplicatePathError{
					oi1:        seenPaths[entries[i].Name()],
					oi2:        sysInfo,
					commonName: path.Join(name, entries[i].Name()),
				}
			}
			seenPaths[entry.Name()] = sysInfo
			continue
		}

		// Add extension to local file path if the file is a notebook
		newEntry := entry.(wsfsDirEntry)
		newEntry.wsfsFileInfo.oi.Path = newEntry.wsfsFileInfo.oi.Path + string(extensionForLanguage(sysInfo.Language))
		entries[i] = newEntry

		if _, ok := seenPaths[newEntry.Name()]; ok {
			return nil, DuplicatePathError{
				oi1:        seenPaths[newEntry.Name()],
				oi2:        sysInfo,
				commonName: path.Join(name, newEntry.Name()),
			}
		}
		seenPaths[newEntry.Name()] = sysInfo

	}

	return entries, nil
}

func (w *workspaceFuseClient) Write(ctx context.Context, name string, reader io.Reader, mode ...WriteMode) error {
	return w.workspaceFilesClient.Write(ctx, name, reader, mode...)
}

func (w *workspaceFuseClient) Read(ctx context.Context, name string) (io.ReadCloser, error) {
	r, err := w.workspaceFilesClient.Read(ctx, name)

	// If the file is not found, it might be a notebook. Try to read the notebook
	// in that case.
	if errors.As(err, &FileDoesNotExistError{}) {
		stripName, ok := stripNotebookExtension(name)
		if !ok {
			return nil, err
		}

		stat, err2 := w.workspaceFilesClient.Stat(ctx, stripName)
		// If we run into an error trying to determine if the file is a notebook,
		// return the original error.
		if err2 != nil {
			return nil, err
		}
		// If the file is not a notebook, return the original error.
		if stat.Sys().(workspace.ObjectInfo).ObjectType != workspace.ObjectTypeNotebook {
			return nil, err
		}

		return w.workspaceFilesClient.Read(ctx, stripName)
	}
	return r, err
}

// TODO: Handle file already exists error everywhere? Atleast provide error messages with better context?

func (w *workspaceFuseClient) Delete(ctx context.Context, name string, mode ...DeleteMode) error {
	err := w.workspaceFilesClient.Delete(ctx, name, mode...)

	// If the file is not found, it might be a notebook.
	if errors.As(err, &FileDoesNotExistError{}) {
		// In that case, we should try to delete the notebook from the workspace.
		stripName, ok := stripNotebookExtension(name)
		if !ok {
			return err
		}

		stat, err2 := w.workspaceFilesClient.Stat(ctx, stripName)
		// If we run into an error trying to determine if the file is a notebook,
		// return the original error.
		if err2 != nil {
			return err
		}
		// If the file is not a notebook, return the original error.
		if stat.Sys().(workspace.ObjectInfo).ObjectType != workspace.ObjectTypeNotebook {
			return err
		}

		// Since the file is a notebook, we should delete the notebook from the workspace.
		return w.workspaceFilesClient.Delete(ctx, stripName, mode...)
	}

	return err
}

func (w *workspaceFuseClient) Stat(ctx context.Context, name string) (fs.FileInfo, error) {
	info, err := w.workspaceFilesClient.Stat(ctx, name)

	// If the file is not found in the local file system, it might be a notebook.
	if errors.As(err, &FileDoesNotExistError{}) {
		stripName, ok := stripNotebookExtension(name)
		if !ok {
			return nil, err
		}

		// Check if the file with it's extension stripped is a notebook. If it is not a notebook,
		// return the original error.
		stat, err2 := w.workspaceFilesClient.Stat(ctx, stripName)
		// If we run into an error trying to determine if the file is a notebook,
		// return the original error.
		if err2 != nil {
			return nil, err
		}
		// If the file is not a notebook, return the original error.
		if stat.Sys().(workspace.ObjectInfo).ObjectType != workspace.ObjectTypeNotebook {
			return nil, err
		}

		// Since the file is a notebook, we should return the stat of the notebook,
		// with the path modified to include the extension.
		newStat := stat.(wsfsFileInfo)
		newStat.oi.Path = newStat.oi.Path + string(extensionForLanguage(stat.Sys().(workspace.ObjectInfo).Language))
		return newStat, nil
	}

	return info, err
}

// TODO: Is incremental sync a problem? Does it need to be fixed for DABs in the
// workspace?
func (w *workspaceFuseClient) Mkdir(ctx context.Context, name string) error {
	return w.workspaceFilesClient.Mkdir(ctx, name)
}

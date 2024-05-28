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

	"github.com/databricks/cli/libs/notebook"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"golang.org/x/exp/maps"
)

// TODO: Better documentation here regarding the boundary conditions and what
// exactly is this filer doing.
type workspaceFuseClient struct {
	workspaceFilesClient Filer
}

// TODO: rename this file

func stripNotebookExtension(ctx context.Context, w Filer, name string) (stripName string, stat fs.FileInfo, ok bool) {
	ext := path.Ext(name)

	extensionsToLanguages := map[string]workspace.Language{
		".py":    workspace.LanguagePython,
		".r":     workspace.LanguageR,
		".scala": workspace.LanguageScala,
		".sql":   workspace.LanguageSql,
		".ipynb": workspace.LanguagePython,
	}

	stripName = strings.TrimSuffix(name, ext)

	// File name does not have an extension associated with Databricks notebooks, return early.
	if !slices.Contains(maps.Keys(extensionsToLanguages), ext) {
		return "", nil, false
	}

	// If the file could be a notebook, check it's a notebook and has the correct language.
	stat, err := w.Stat(ctx, stripName)
	if err != nil {
		return "", nil, false
	}
	info := stat.Sys().(workspace.ObjectInfo)

	// Not a notebook. Return early.
	if info.ObjectType != workspace.ObjectTypeNotebook {
		return "", nil, false
	}

	// Not the correct language. Return early.
	// TODO: Add a test for this.
	if info.Language != extensionsToLanguages[ext] {
		return "", nil, false
	}

	return stripName, stat, true
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
	wc, err := NewWorkspaceFilesClient(w, root)
	if err != nil {
		return nil, err
	}

	return &workspaceFuseClient{
		workspaceFilesClient: wc,
	}, nil
}

// TODO: Write note on read methods that it's unsafe in that it might not error
// on unsupported setups. Or maybe document functionality.

// TODO: Note the loss of information when writing a ipynb notebook.

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
		newEntry.wsfsFileInfo.oi.Path = newEntry.wsfsFileInfo.oi.Path + notebook.GetExtensionByLanguage(&sysInfo)
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

// Note: There is loss of information when writing a ipynb file. A notebook written
// with .Write(name = "foo.ipynb") will be written as "foo" in the workspace and
// will have to be read as .Read(name = "foo.py") instead of "foo.ipynb
func (w *workspaceFuseClient) Write(ctx context.Context, name string, reader io.Reader, mode ...WriteMode) error {
	return w.workspaceFilesClient.Write(ctx, name, reader, mode...)
}

func (w *workspaceFuseClient) Read(ctx context.Context, name string) (io.ReadCloser, error) {
	r, err := w.workspaceFilesClient.Read(ctx, name)

	// If the file is not found, it might be a notebook.
	if errors.As(err, &FileDoesNotExistError{}) {
		stripName, _, ok := stripNotebookExtension(ctx, w.workspaceFilesClient, name)
		if !ok {
			// Not a valid notebook. Return the original error.
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
		stripName, _, ok := stripNotebookExtension(ctx, w.workspaceFilesClient, name)
		if !ok {
			// Not a valid notebook. Return the original error.
			return err
		}
		return w.workspaceFilesClient.Delete(ctx, stripName, mode...)
	}

	return err
}

func (w *workspaceFuseClient) Stat(ctx context.Context, name string) (fs.FileInfo, error) {
	info, err := w.workspaceFilesClient.Stat(ctx, name)

	// If the file is not found in the local file system, it might be a notebook.
	if errors.As(err, &FileDoesNotExistError{}) {
		_, stat, ok := stripNotebookExtension(ctx, w.workspaceFilesClient, name)
		if !ok {
			return nil, err
		}

		// Since the file is a notebook, we should return the stat of the notebook,
		// with the path modified to include the extension.
		newStat := stat.(wsfsFileInfo)
		newStat.oi.Path = newStat.oi.Path + notebook.GetExtensionByLanguage(&newStat.oi)
		return newStat, nil
	}

	return info, err
}

func (w *workspaceFuseClient) Mkdir(ctx context.Context, name string) error {
	return w.workspaceFilesClient.Mkdir(ctx, name)
}

// TODO: Iterate on the comments for this filer.

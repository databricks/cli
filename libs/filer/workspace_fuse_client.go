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

// TODO: What if the resolved name of a notebook clashed with a directory.

// This is a filer for the workspace file system that allows you to pretend the
// workspace file system is a traditional file system. It allows you to list, read, write,
// delete, and stat notebooks (and files in general) in the workspace, using their paths
// with the extension included.
//
// The ReadDir method returns a DuplicatePathError if this traditional file system view is
// not possible. For example, a python notebook called foo and a python file called foo.py
// would resolve to the same path foo.py in a tradition file system.
//
// Users of this filer should be careful when using the Write and Mkdir methods.
// The underlying import API we use to upload notebooks and files returns opaque internal
// errors for namespace clashes (e.g. a file and a notebook or a directory and a notebook).
// Thus users of these methods should be careful to avoid such clashes.
func NewWorkspaceFuseClient(w *databricks.WorkspaceClient, root string) (Filer, error) {
	wc, err := NewWorkspaceFilesClient(w, root)
	if err != nil {
		return nil, err
	}

	return &workspaceFuseClient{
		workspaceFilesClient: wc,
	}, nil
}

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
//
// Note: The import API returns opaque internal errors for namespace clashes
// (e.g. a file and a notebook or a directory and a notebook). Thus users of this
// method should be careful to avoid such clashes.
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

// Note: The import API returns opaque internal errors for namespace clashes
// (e.g. a file and a notebook or a directory and a notebook). Thus users of this
// method should be careful to avoid such clashes.
func (w *workspaceFuseClient) Mkdir(ctx context.Context, name string) error {
	return w.workspaceFilesClient.Mkdir(ctx, name)
}

// TODO: Iterate on the comments for this filer.

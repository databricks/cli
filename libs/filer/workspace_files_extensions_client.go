package filer

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/notebook"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/workspace"
)

type workspaceFilesExtensionsClient struct {
	workspaceFilesClient Filer

	root      string
	apiClient *client.DatabricksClient
}

var extensionsToLanguages = map[string]workspace.Language{
	".py":    workspace.LanguagePython,
	".r":     workspace.LanguageR,
	".scala": workspace.LanguageScala,
	".sql":   workspace.LanguageSql,
	".ipynb": workspace.LanguagePython,
}

type workspaceFileStatus struct {
	*workspace.ObjectInfo
	ReposExportFormat workspace.ExportFormat `json:"repos_export_format,omitempty"`

	// Name of the file to be used in any API calls made using the workspace files
	// filer. For notebooks this path does not include the extension.
	nameForWorkspaceAPI string
}

// A custom unmarsaller for the workspaceFileStatus struct. This is needed because
// workspaceFileStatus embeds the workspace.ObjectInfo which itself has a custom
// unmarshaller.
// If a custom unmarshaller is not provided extra fields like ReposExportFormat
// will not have values set.
func (s *workspaceFileStatus) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s *workspaceFileStatus) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

// This function returns the stat for the provided notebook. The stat object itself
// contains the path with the extension since it is meant used in the context of a fs.FileInfo.
func (w *workspaceFilesExtensionsClient) removeNotebookExtension(ctx context.Context, name string) (stat *workspaceFileStatus, ok bool) {
	ext := path.Ext(name)

	nameWithoutExtension := strings.TrimSuffix(name, ext)

	// File name does not have an extension associated that is with Databricks
	// notebooks, return early.
	if _, ok := extensionsToLanguages[ext]; !ok {
		return nil, false
	}

	// If the file could be a notebook, check it's a notebook and has the correct language.
	// We need repos_export_format to determine if the file is a py or a ipynb notebook.
	// This is not exposed by the SDK so we need to make a direct API call.
	stat = &workspaceFileStatus{
		nameForWorkspaceAPI: nameWithoutExtension,
	}

	err := w.apiClient.Do(ctx, http.MethodGet, "/api/2.0/workspace/get-status", nil,
		map[string]string{"path": path.Join(w.root, nameWithoutExtension), "return_export_info": "true"}, stat)
	if err != nil {
		log.Debugf(ctx, "attempting to determine if %s could be a notebook. Failed to fetch the status of object at %s: %s", name, path.Join(w.root, nameWithoutExtension), err)
		return nil, false
	}

	// Not a notebook. Return early.
	if stat.ObjectType != workspace.ObjectTypeNotebook {
		log.Debugf(ctx, "attempting to determine if %s could be a notebook. Found an object at %s but it is not a notebook. It is a %s.", name, path.Join(w.root, nameWithoutExtension), stat.ObjectType)
		return nil, false
	}

	// Not the correct language. Return early.
	if stat.Language != extensionsToLanguages[ext] {
		log.Debugf(ctx, "attempting to determine if %s could be a notebook. Found a notebook at %s but it is not of the correct language. Expected %s but found %s.", name, path.Join(w.root, nameWithoutExtension), extensionsToLanguages[ext], stat.Language)
		return nil, false
	}

	// When the extension is .py we expect the export format to be source.
	// If it's not, return early.
	if ext == ".py" && stat.ReposExportFormat != workspace.ExportFormatSource {
		log.Debugf(ctx, "attempting to determine if %s could be a notebook. Found a notebook at %s but it is not exported as a source notebook. Its export format is %s.", name, path.Join(w.root, nameWithoutExtension), stat.ReposExportFormat)
		return nil, false
	}

	// When the extension is .ipynb we expect the export format to be jupyter.
	// If it's not, return early.
	if ext == ".ipynb" && stat.ReposExportFormat != workspace.ExportFormatJupyter {
		log.Debugf(ctx, "attempting to determine if %s could be a notebook. Found a notebook at %s but it is not exported as a jupyter notebook. Its export format is %s.", name, path.Join(w.root, nameWithoutExtension), stat.ReposExportFormat)
		return nil, false
	}

	// Modify the stat object path to include the extension. This stat object will be used
	// to return the fs.FileInfo object in the stat method.
	stat.Path = stat.Path + ext
	return stat, true
}

func (w *workspaceFilesExtensionsClient) addNotebookExtension(ctx context.Context, name string) (stat *workspaceFileStatus, err error) {
	// Get status of the file to determine it's extension.
	stat = &workspaceFileStatus{
		nameForWorkspaceAPI: name,
		ObjectInfo:          &workspace.ObjectInfo{},
	}
	err = w.apiClient.Do(ctx, http.MethodGet, "/api/2.0/workspace/get-status", nil,
		map[string]string{"path": path.Join(w.root, name), "return_export_info": "true"}, stat)
	if err != nil {
		return nil, err
	}

	// Not a notebook. Return early.
	if stat.ObjectType != workspace.ObjectTypeNotebook {
		return nil, nil
	}

	// Get the extension for the notebook.
	ext := notebook.GetExtensionByLanguage(stat.ObjectInfo)

	// If the notebook was exported as a jupyter notebook, the extension should be .ipynb.
	if stat.Language == workspace.LanguagePython && stat.ReposExportFormat == workspace.ExportFormatJupyter {
		ext = ".ipynb"
	}

	// Modify the stat object path to include the extension. This stat object will be used
	// to return the fs.DirEntry object in the ReadDir method.
	stat.Path = stat.Path + ext
	return stat, nil
}

type DuplicatePathError struct {
	oi1 workspace.ObjectInfo
	oi2 workspace.ObjectInfo

	commonName string
}

func (e DuplicatePathError) Error() string {
	return fmt.Sprintf("failed to read files from the workspace file system. Duplicate paths encountered. Both %s at %s and %s at %s resolve to the same name %s. Changing the name of one of these objects will resolve this issue", e.oi1.ObjectType, e.oi1.Path, e.oi2.ObjectType, e.oi2.Path, e.commonName)
}

// This is a filer for the workspace file system that allows you to pretend the
// workspace file system is a traditional file system. It allows you to list, read, write,
// delete, and stat notebooks (and files in general) in the workspace, using their paths
// with the extension included.
//
// The ReadDir method returns a DuplicatePathError if this traditional file system view is
// not possible. For example, a Python notebook called foo and a Python file called `foo.py`
// would resolve to the same path `foo.py` in a tradition file system.
//
// Users of this filer should be careful when using the Write and Mkdir methods.
// The underlying import API we use to upload notebooks and files returns opaque internal
// errors for namespace clashes (e.g. a file and a notebook or a directory and a notebook).
// Thus users of these methods should be careful to avoid such clashes.
func NewWorkspaceFilesExtensionsClient(w *databricks.WorkspaceClient, root string) (Filer, error) {
	wc, err := NewWorkspaceFilesClient(w, root)
	if err != nil {
		return nil, err
	}

	apiClient, err := client.New(w.Config)
	if err != nil {
		return nil, err
	}

	return &workspaceFilesExtensionsClient{
		root:                 root,
		workspaceFilesClient: wc,
		apiClient:            apiClient,
	}, nil
}

func (w *workspaceFilesExtensionsClient) ReadDir(ctx context.Context, name string) ([]fs.DirEntry, error) {
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
			// Error if we have seen this path before in the current directory.
			// If not seen before, add it to the seen paths.
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
		stat, err := w.addNotebookExtension(ctx, entry.Name())
		if err != nil {
			return nil, err
		}
		entries[i] = wsfsDirEntry{wsfsFileInfo{oi: *stat.ObjectInfo}}

		// Error if we have seen this path before in the current directory.
		// If not seen before, add it to the seen paths.
		if _, ok := seenPaths[entries[i].Name()]; ok {
			return nil, DuplicatePathError{
				oi1:        seenPaths[entries[i].Name()],
				oi2:        sysInfo,
				commonName: path.Join(name, entries[i].Name()),
			}
		}
		seenPaths[entries[i].Name()] = sysInfo

	}

	return entries, nil
}

// Note: The import API returns opaque internal errors for namespace clashes
// (e.g. a file and a notebook or a directory and a notebook). Thus users of this
// method should be careful to avoid such clashes.
func (w *workspaceFilesExtensionsClient) Write(ctx context.Context, name string, reader io.Reader, mode ...WriteMode) error {
	return w.workspaceFilesClient.Write(ctx, name, reader, mode...)
}

// Try to read the file as a regular file. If the file is not found, try to read it as a notebook.
func (w *workspaceFilesExtensionsClient) Read(ctx context.Context, name string) (io.ReadCloser, error) {
	r, err := w.workspaceFilesClient.Read(ctx, name)

	// If the file is not found, it might be a notebook.
	if errors.As(err, &FileDoesNotExistError{}) {
		stat, ok := w.removeNotebookExtension(ctx, name)
		if !ok {
			// Not a valid notebook. Return the original error.
			return nil, err
		}
		return w.workspaceFilesClient.Read(ctx, stat.nameForWorkspaceAPI)
	}
	return r, err
}

// Try to delete the file as a regular file. If the file is not found, try to delete it as a notebook.
func (w *workspaceFilesExtensionsClient) Delete(ctx context.Context, name string, mode ...DeleteMode) error {
	err := w.workspaceFilesClient.Delete(ctx, name, mode...)

	// If the file is not found, it might be a notebook.
	if errors.As(err, &FileDoesNotExistError{}) {
		stat, ok := w.removeNotebookExtension(ctx, name)
		if !ok {
			// Not a valid notebook. Return the original error.
			return err
		}
		return w.workspaceFilesClient.Delete(ctx, stat.nameForWorkspaceAPI, mode...)
	}

	return err
}

// Try to stat the file as a regular file. If the file is not found, try to stat it as a notebook.
func (w *workspaceFilesExtensionsClient) Stat(ctx context.Context, name string) (fs.FileInfo, error) {
	info, err := w.workspaceFilesClient.Stat(ctx, name)

	// If the file is not found in the local file system, it might be a notebook.
	if errors.As(err, &FileDoesNotExistError{}) {
		stat, ok := w.removeNotebookExtension(ctx, name)
		if !ok {
			return nil, err
		}

		return wsfsFileInfo{oi: *stat.ObjectInfo}, nil
	}

	return info, err
}

// Note: The import API returns opaque internal errors for namespace clashes
// (e.g. a file and a notebook or a directory and a notebook). Thus users of this
// method should be careful to avoid such clashes.
func (w *workspaceFilesExtensionsClient) Mkdir(ctx context.Context, name string) error {
	return w.workspaceFilesClient.Mkdir(ctx, name)
}

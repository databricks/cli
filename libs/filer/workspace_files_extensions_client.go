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
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/workspace"
)

type workspaceFilesExtensionsClient struct {
	workspaceClient *databricks.WorkspaceClient
	apiClient       *client.DatabricksClient

	wsfs Filer
	root string
}

var extensionsToLanguages = map[string]workspace.Language{
	".py":    workspace.LanguagePython,
	".r":     workspace.LanguageR,
	".scala": workspace.LanguageScala,
	".sql":   workspace.LanguageSql,
	".ipynb": workspace.LanguagePython,
}

// workspaceFileStatus defines a custom response body for the "/api/2.0/workspace/get-status" API.
// The "repos_export_format" field is not exposed by the SDK.
type workspaceFileStatus struct {
	*workspace.ObjectInfo

	// The export format of the notebook. This is not exposed by the SDK.
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

func (w *workspaceFilesExtensionsClient) stat(ctx context.Context, name string) (*workspaceFileStatus, error) {
	stat := &workspaceFileStatus{
		nameForWorkspaceAPI: name,
	}

	// Perform bespoke API call because "return_export_info" is not exposed by the SDK.
	// We need "repos_export_format" to determine if the file is a py or a ipynb notebook.
	// This is not exposed by the SDK so we need to make a direct API call.
	err := w.apiClient.Do(
		ctx,
		http.MethodGet,
		"/api/2.0/workspace/get-status",
		nil,
		map[string]string{
			"path":               path.Join(w.root, name),
			"return_export_info": "true",
		},
		stat,
	)
	if err != nil {
		// If we got an API error we deal with it below.
		var aerr *apierr.APIError
		if !errors.As(err, &aerr) {
			return nil, err
		}

		// This API returns a 404 if the specified path does not exist.
		if aerr.StatusCode == http.StatusNotFound {
			return nil, FileDoesNotExistError{path.Join(w.root, name)}
		}
	}
	return stat, err
}

// This function returns the stat for the provided notebook. The stat object itself contains the path
// with the extension since it is meant to be used in the context of a fs.FileInfo.
func (w *workspaceFilesExtensionsClient) getNotebookStatByNameWithExt(ctx context.Context, name string) (*workspaceFileStatus, error) {
	ext := path.Ext(name)
	nameWithoutExt := strings.TrimSuffix(name, ext)

	// File name does not have an extension associated with Databricks notebooks, return early.
	if _, ok := extensionsToLanguages[ext]; !ok {
		return nil, nil
	}

	// If the file could be a notebook, check if it is and has the correct language.
	stat, err := w.stat(ctx, nameWithoutExt)
	if err != nil {
		// If the file does not exist, return early.
		if errors.As(err, &FileDoesNotExistError{}) {
			return nil, nil
		}
		log.Debugf(ctx, "attempting to determine if %s could be a notebook. Failed to fetch the status of object at %s: %s", name, path.Join(w.root, nameWithoutExt), err)
		return nil, err
	}

	// Not a notebook. Return early.
	if stat.ObjectType != workspace.ObjectTypeNotebook {
		log.Debugf(ctx, "attempting to determine if %s could be a notebook. Found an object at %s but it is not a notebook. It is a %s.", name, path.Join(w.root, nameWithoutExt), stat.ObjectType)
		return nil, nil
	}

	// Not the correct language. Return early.
	if stat.Language != extensionsToLanguages[ext] {
		log.Debugf(ctx, "attempting to determine if %s could be a notebook. Found a notebook at %s but it is not of the correct language. Expected %s but found %s.", name, path.Join(w.root, nameWithoutExt), extensionsToLanguages[ext], stat.Language)
		return nil, nil
	}

	// When the extension is .py we expect the export format to be source.
	// If it's not, return early.
	if ext == ".py" && stat.ReposExportFormat != workspace.ExportFormatSource {
		log.Debugf(ctx, "attempting to determine if %s could be a notebook. Found a notebook at %s but it is not exported as a source notebook. Its export format is %s.", name, path.Join(w.root, nameWithoutExt), stat.ReposExportFormat)
		return nil, nil
	}

	// When the extension is .ipynb we expect the export format to be Jupyter.
	// If it's not, return early.
	if ext == ".ipynb" && stat.ReposExportFormat != workspace.ExportFormatJupyter {
		log.Debugf(ctx, "attempting to determine if %s could be a notebook. Found a notebook at %s but it is not exported as a Jupyter notebook. Its export format is %s.", name, path.Join(w.root, nameWithoutExt), stat.ReposExportFormat)
		return nil, nil
	}

	// Modify the stat object path to include the extension. This stat object will be used
	// to return the fs.FileInfo object in the stat method.
	stat.Path = stat.Path + ext
	return stat, nil
}

func (w *workspaceFilesExtensionsClient) getNotebookStatByNameWithoutExt(ctx context.Context, name string) (*workspaceFileStatus, error) {
	stat, err := w.stat(ctx, name)
	if err != nil {
		return nil, err
	}

	// We expect this internal function to only be called from [ReadDir] when we are sure
	// that the object is a notebook. Thus, this should never happen.
	if stat.ObjectType != workspace.ObjectTypeNotebook {
		return nil, fmt.Errorf("expected object at %s to be a notebook but it is a %s", path.Join(w.root, name), stat.ObjectType)
	}

	// Get the extension for the notebook.
	ext := notebook.GetExtensionByLanguage(stat.ObjectInfo)

	// If the notebook was exported as a Jupyter notebook, the extension should be .ipynb.
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
	apiClient, err := client.New(w.Config)
	if err != nil {
		return nil, err
	}

	filer, err := NewWorkspaceFilesClient(w, root)
	if err != nil {
		return nil, err
	}

	return &workspaceFilesExtensionsClient{
		workspaceClient: w,
		apiClient:       apiClient,

		wsfs: filer,
		root: root,
	}, nil
}

func (w *workspaceFilesExtensionsClient) ReadDir(ctx context.Context, name string) ([]fs.DirEntry, error) {
	entries, err := w.wsfs.ReadDir(ctx, name)
	if err != nil {
		return nil, err
	}

	seenPaths := make(map[string]workspace.ObjectInfo)
	for i := range entries {
		info, err := entries[i].Info()
		if err != nil {
			return nil, err
		}
		sysInfo := info.Sys().(workspace.ObjectInfo)

		// If the object is a notebook, include an extension in the entry.
		if sysInfo.ObjectType == workspace.ObjectTypeNotebook {
			stat, err := w.getNotebookStatByNameWithoutExt(ctx, path.Join(name, entries[i].Name()))
			if err != nil {
				return nil, err
			}
			// Replace the entry with the new entry that includes the extension.
			entries[i] = wsfsDirEntry{wsfsFileInfo{oi: *stat.ObjectInfo}}
		}

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
	return w.wsfs.Write(ctx, name, reader, mode...)
}

// Try to read the file as a regular file. If the file is not found, try to read it as a notebook.
func (w *workspaceFilesExtensionsClient) Read(ctx context.Context, name string) (io.ReadCloser, error) {
	r, err := w.wsfs.Read(ctx, name)

	// If the file is not found, it might be a notebook.
	if errors.As(err, &FileDoesNotExistError{}) {
		stat, serr := w.getNotebookStatByNameWithExt(ctx, name)
		if serr != nil {
			// Unable to stat. Return the stat error.
			return nil, serr
		}
		if stat == nil {
			// Not a notebook. Return the original error.
			return nil, err
		}

		// The workspace files filer performs an additional stat call to make sure
		// the path is not a directory. We can skip this step since we already have
		// the stat object and know that the path is a notebook.
		return w.workspaceClient.Workspace.Download(
			ctx,
			path.Join(w.root, stat.nameForWorkspaceAPI),
			workspace.DownloadFormat(stat.ReposExportFormat),
		)
	}
	return r, err
}

// Try to delete the file as a regular file. If the file is not found, try to delete it as a notebook.
func (w *workspaceFilesExtensionsClient) Delete(ctx context.Context, name string, mode ...DeleteMode) error {
	err := w.wsfs.Delete(ctx, name, mode...)

	// If the file is not found, it might be a notebook.
	if errors.As(err, &FileDoesNotExistError{}) {
		stat, serr := w.getNotebookStatByNameWithExt(ctx, name)
		if serr != nil {
			// Unable to stat. Return the stat error.
			return serr
		}
		if stat == nil {
			// Not a notebook. Return the original error.
			return err
		}

		return w.wsfs.Delete(ctx, stat.nameForWorkspaceAPI, mode...)
	}

	return err
}

// Try to stat the file as a regular file. If the file is not found, try to stat it as a notebook.
func (w *workspaceFilesExtensionsClient) Stat(ctx context.Context, name string) (fs.FileInfo, error) {
	info, err := w.wsfs.Stat(ctx, name)

	// If the file is not found, it might be a notebook.
	if errors.As(err, &FileDoesNotExistError{}) {
		stat, serr := w.getNotebookStatByNameWithExt(ctx, name)
		if serr != nil {
			// Unable to stat. Return the stat error.
			return nil, serr
		}
		if stat == nil {
			// Not a notebook. Return the original error.
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
	return w.wsfs.Mkdir(ctx, name)
}

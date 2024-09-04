package template

import (
	"context"
	"encoding/base64"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/notebook"
	"github.com/databricks/cli/libs/runtime"
	"github.com/databricks/databricks-sdk-go/service/workspace"
)

// Interface representing a file to be materialized from a template into a project
// instance
type file interface {
	// Destination path for file. This is where the file will be created when
	// PersistToDisk is called.
	DstPath() *destinationPath

	// Write file to disk at the destination path.
	PersistToDisk() error
}

type destinationPath struct {
	// Root path for the project instance. This path uses the system's default
	// file separator. For example /foo/bar on Unix and C:\foo\bar on windows
	root string

	// Unix like file path relative to the "root" of the instantiated project. Is used to
	// evaluate whether the file should be skipped by comparing it to a list of
	// skip glob patterns.
	relPath string
}

// Absolute path of the file, in the os native format. For example /foo/bar on
// Unix and C:\foo\bar on windows
func (f *destinationPath) absPath() string {
	return filepath.Join(f.root, filepath.FromSlash(f.relPath))
}

type copyFile struct {
	ctx context.Context

	// Permissions bits for the destination file
	perm fs.FileMode

	dstPath *destinationPath

	// Filer rooted at template root. Used to read srcPath.
	srcFiler filer.Filer

	// Relative path from template root for file to be copied.
	srcPath string
}

func (f *copyFile) DstPath() *destinationPath {
	return f.dstPath
}

func (f *copyFile) PersistToDisk() error {
	path := f.DstPath().absPath()
	err := os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		return err
	}
	srcFile, err := f.srcFiler.Read(f.ctx, f.srcPath)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	content, err := io.ReadAll(srcFile)
	if err != nil {
		return err
	}
	return writeFile(f.ctx, path, content, f.perm)
}

type inMemoryFile struct {
	ctx context.Context

	dstPath *destinationPath

	content []byte

	// Permissions bits for the destination file
	perm fs.FileMode
}

func (f *inMemoryFile) DstPath() *destinationPath {
	return f.dstPath
}

func (f *inMemoryFile) PersistToDisk() error {
	path := f.DstPath().absPath()

	err := os.MkdirAll(filepath.Dir(path), 0755)
	if err != nil {
		return err
	}

	return writeFile(f.ctx, path, f.content, f.perm)
}

func shouldUseImportNotebook(ctx context.Context, path string, content []byte) bool {
	if strings.HasPrefix(path, "/Workspace/") && runtime.RunsOnDatabricks(ctx) {
		isNotebook, _, err := notebook.DetectWithContent(path, content)
		if err != nil {
			log.Debugf(ctx, "Error detecting notebook: %v", err)
		}
		return isNotebook && err != nil
	}

	return false
}

func writeFile(ctx context.Context, path string, content []byte, perm fs.FileMode) error {
	if shouldUseImportNotebook(ctx, path, content) {
		return importNotebook(ctx, path, content)
	} else {
		return os.WriteFile(path, content, perm)
	}
}

func importNotebook(ctx context.Context, path string, content []byte) error {
	w := root.WorkspaceClient(ctx)

	return w.Workspace.Import(ctx, workspace.Import{
		Format:    "AUTO",
		Overwrite: false,
		Path:      path,
		Content:   base64.StdEncoding.EncodeToString(content),
	})
}

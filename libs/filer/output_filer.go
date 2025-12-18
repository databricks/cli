package filer

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/dbr"
)

// NewOutputFiler creates a filer for writing output files.
// When running on DBR and writing to the workspace filesystem, it uses the
// workspace files extensions client (import/export API) to support writing notebooks.
// Otherwise, it uses the local filesystem client.
//
// It is not possible to write notebooks through the workspace filesystem's FUSE mount for DBR versions less than 16.4.
// This function ensures the correct filer is used based on the runtime environment.
func NewOutputFiler(ctx context.Context, outputDir string) (Filer, error) {
	outputDir, err := filepath.Abs(outputDir)
	if err != nil {
		return nil, err
	}

	// If the CLI is running on DBR and we're writing to the workspace file system,
	// use the extension-aware workspace filesystem filer.
	if strings.HasPrefix(outputDir, "/Workspace/") && dbr.RunsOnRuntime(ctx) {
		w := cmdctx.WorkspaceClient(ctx)
		return NewWorkspaceFilesExtensionsClient(w, outputDir)
	}

	return NewLocalClient(outputDir)
}

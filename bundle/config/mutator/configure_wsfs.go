package mutator

import (
	"context"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/vfs"
)

type configureWsfs struct{}

func ConfigureWsfs() bundle.Mutator {
	return &configureWsfs{}
}

func (m *configureWsfs) Name() string {
	return "ConfigureWsfs"
}

func (m *configureWsfs) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	root := b.BundleRoot.Native()

	// The bundle root must be is located in /Workspace/
	if !strings.HasPrefix(root, "/Workspace/") {
		return nil
	}

	// The executable must be running on DBR.
	if _, ok := env.Lookup(ctx, "DATABRICKS_RUNTIME_VERSION"); !ok {
		return nil
	}

	// If so, swap out vfs.Path instance of the sync root with one that
	// makes all Workspace File System interactions extension aware.
	p, err := vfs.NewFilerPath(ctx, root, func(path string) (filer.Filer, error) {
		return filer.NewWorkspaceFilesExtensionsClient(b.WorkspaceClient(), root)
	})
	if err != nil {
		return diag.FromErr(err)
	}

	b.SyncRoot = p
	return nil
}

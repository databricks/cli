package mutator

import (
	"context"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/dbr"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/vfs"
)

type configureWSFS struct{}

func ConfigureWSFS() bundle.Mutator {
	return &configureWSFS{}
}

func (m *configureWSFS) Name() string {
	return "ConfigureWSFS"
}

func (m *configureWSFS) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	root := b.SyncRoot.Native()

	// The bundle root must be located in /Workspace/
	if !strings.HasPrefix(root, "/Workspace/") {
		return nil
	}

	// The executable must be running on DBR.
	if !dbr.RunsOnRuntime(ctx) {
		return nil
	}

	// On serverless (client version 2+), use the native sync root directly via FUSE.
	// The FUSE provides capabitilies for both reading and writing notebooks. It also
	// is much faster and enables running cloud tests on DBR, since otherwise the tests
	// fail with an AsyncFlushError because of the conflict between writing to FUSE
	// and via the workspace APIs simultaneously.
	v := dbr.GetVersion(ctx)
	if v.Type == dbr.ClusterTypeServerless && v.Major >= 2 {
		return nil
	}

	// If so, swap out vfs.Path instance of the sync root with one that
	// makes all Workspace File System interactions extension aware.
	p, err := vfs.NewFilerPath(ctx, root, func(path string) (filer.Filer, error) {
		return filer.NewReadOnlyWorkspaceFilesExtensionsClient(b.WorkspaceClient(), path)
	})
	if err != nil {
		return diag.FromErr(err)
	}

	b.SyncRoot = p
	return nil
}

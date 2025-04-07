package mutator

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/vfs"
)

type syncInferRoot struct{}

// SyncInferRoot is a mutator that infers the root path of all files to synchronize by looking at the
// paths in the sync configuration. The sync root may be different from the bundle root
// when the user intends to synchronize files outside the bundle root.
//
// The sync root can be equivalent to or an ancestor of the bundle root, but not a descendant.
// That is, the sync root must contain the bundle root.
//
// This mutator requires all sync-related paths and patterns to be relative to the bundle root path.
// This is done by the [RewriteSyncPaths] mutator, which must run before this mutator.
func SyncInferRoot() bundle.Mutator {
	return &syncInferRoot{}
}

func (m *syncInferRoot) Name() string {
	return "SyncInferRoot"
}

// computeRoot finds the innermost path that contains the specified path.
// It traverses up the root path until it finds the innermost path.
// If the path does not exist, it returns an empty string.
//
// See "sync_infer_root_internal_test.go" for examples.
func (m *syncInferRoot) computeRoot(path, root string) string {
	for !filepath.IsLocal(path) {
		// Break if we have reached the root of the filesystem.
		dir := filepath.Dir(root)
		if dir == root {
			return ""
		}

		// Update the sync path as we navigate up the directory tree.
		path = filepath.Join(filepath.Base(root), path)

		// Move up the directory tree.
		root = dir
	}

	return filepath.Clean(root)
}

func (m *syncInferRoot) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics

	// Use the bundle root path as the starting point for inferring the sync root path.
	bundleRootPath := filepath.Clean(b.BundleRootPath)

	// Infer the sync root path by looking at each one of the sync paths.
	// Every sync path must be a descendant of the final sync root path.
	syncRootPath := bundleRootPath
	for _, path := range b.Config.Sync.Paths {
		computedPath := m.computeRoot(path, bundleRootPath)
		if computedPath == "" {
			continue
		}

		// Update sync root path if the computed root path is an ancestor of the current sync root path.
		if len(computedPath) < len(syncRootPath) {
			syncRootPath = computedPath
		}
	}

	// The new sync root path can only be an ancestor of the previous root path.
	// Compute the relative path from the sync root to the bundle root.
	rel, err := filepath.Rel(syncRootPath, bundleRootPath)
	if err != nil {
		return diag.FromErr(err)
	}

	// If during computation of the sync root path we hit the root of the filesystem,
	// then one or more of the sync paths are outside the filesystem.
	// Check if this happened by verifying that none of the paths escape the root
	// when joined with the sync root path.
	for i, path := range b.Config.Sync.Paths {
		if filepath.IsLocal(filepath.Join(rel, path)) {
			continue
		}

		diags = append(diags, diag.Diagnostic{
			Severity:  diag.Error,
			Summary:   fmt.Sprintf("invalid sync path %q", path),
			Locations: b.Config.GetLocations(fmt.Sprintf("sync.paths[%d]", i)),
			Paths:     []dyn.Path{dyn.NewPath(dyn.Key("sync"), dyn.Key("paths"), dyn.Index(i))},
		})
	}

	if diags.HasError() {
		return diags
	}

	// Update all paths in the sync configuration to be relative to the sync root.
	for i, p := range b.Config.Sync.Paths {
		b.Config.Sync.Paths[i] = filepath.Join(rel, p)
	}

	// Convert include and exclude in the sync block to use Unix-style slashes.
	// This is required for the ignore.GitIgnore we use in libs/fileset to work correctly.
	for i, p := range b.Config.Sync.Include {
		b.Config.Sync.Include[i] = filepath.ToSlash(filepath.Join(rel, p))
	}
	for i, p := range b.Config.Sync.Exclude {
		b.Config.Sync.Exclude[i] = filepath.ToSlash(filepath.Join(rel, p))
	}

	// Configure the sync root path.
	b.SyncRoot = vfs.MustNew(syncRootPath)
	b.SyncRootPath = syncRootPath
	return nil
}

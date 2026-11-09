package run

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/notebook"
	"github.com/databricks/cli/libs/sync"
)

// ResolveRefreshSelectors resolves source files to workspace paths.
func (o *PipelineOptions) ResolveRefreshSelectors(ctx context.Context, b *bundle.Bundle) error {
	r := refreshSelectorResolver{bundle: b}
	var err error
	o.Refresh, o.RefreshByPath, err = r.resolve(ctx, o.Refresh)
	if err != nil {
		return fmt.Errorf("invalid --refresh: %w", err)
	}
	o.FullRefresh, o.FullRefreshByPath, err = r.resolve(ctx, o.FullRefresh)
	if err != nil {
		return fmt.Errorf("invalid --full-refresh: %w", err)
	}
	// Validate before Restart stops an active update.
	return o.Validate(nil)
}

type refreshSelectorResolver struct {
	bundle      *bundle.Bundle
	remoteNames map[string]string
}

func (r *refreshSelectorResolver) resolve(ctx context.Context, selectors []string) (tables, paths []string, err error) {
	b := r.bundle
	for _, selector := range selectors {
		// /Workspace/ distinguishes remote paths from local paths like /Users/... on macOS.
		if strings.HasPrefix(selector, "/Workspace/") {
			paths = append(paths, selector)
			continue
		}

		// Quoted identifiers remain table selectors even when they contain path separators.
		if strings.Contains(selector, "`") {
			tables = append(tables, selector)
			continue
		}

		info, statErr := os.Stat(selector)
		// A bare selector is a table unless it identifies an existing local file.
		looksLikeFile := strings.ContainsAny(selector, `/\`)
		if !looksLikeFile && (statErr != nil || info.IsDir()) {
			tables = append(tables, selector)
			continue
		}
		if errors.Is(statErr, fs.ErrNotExist) {
			return nil, nil, fmt.Errorf("source file %q not found; use a table name or an existing source file", selector)
		}
		if statErr != nil {
			return nil, nil, fmt.Errorf("cannot read refresh selector %q: %w", selector, statErr)
		}
		if info.IsDir() {
			return nil, nil, fmt.Errorf("refresh selector %q is a directory; select a source file", selector)
		}

		localPath, err := filepath.Abs(selector)
		if err != nil {
			return nil, nil, err
		}
		rel, err := filepath.Rel(b.SyncRootPath, localPath)
		if err != nil {
			return nil, nil, err
		}
		if !filepath.IsLocal(rel) {
			return nil, nil, fmt.Errorf("source file %q is outside the bundle sync root", selector)
		}
		if r.remoteNames == nil {
			// Refresh uses the deployed snapshot.
			if b.IsImmutableFolder() {
				if err := b.DeploymentBundle.ResolveConfigAgainstState(&b.Config, dyn.MustPathFromString("workspace.file_path")); err != nil {
					return nil, nil, err
				}
			}
			r.remoteNames, err = refreshSourcePaths(ctx, b)
			if err != nil {
				return nil, nil, err
			}
		}
		remoteName, ok := r.remoteNames[filepath.ToSlash(rel)]
		if !ok {
			if _, _, err := notebook.DetectWithFS(b.SyncRoot, filepath.ToSlash(rel)); err != nil {
				return nil, nil, fmt.Errorf("cannot identify source file %q: %w", selector, err)
			}
			return nil, nil, fmt.Errorf("source file %q is excluded from bundle sync; include it and deploy first", selector)
		}
		remoteRoot := b.RemoteFilePath()
		if strings.Contains(remoteRoot, "${") {
			return nil, nil, fmt.Errorf("cannot resolve the deployed path for %q; use its absolute /Workspace/ path", selector)
		}
		paths = append(paths, path.Join(remoteRoot, remoteName))
	}
	return tables, paths, nil
}

// refreshSourcePaths maps local files to their deployed names.
func refreshSourcePaths(ctx context.Context, b *bundle.Bundle) (map[string]string, error) {
	files, err := b.GetSyncFiles(ctx)
	if err != nil {
		return nil, err
	}
	state, err := sync.NewSnapshotState(files)
	if err != nil {
		return nil, err
	}
	return state.LocalToRemoteNames, nil
}

package pipelines

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
	"github.com/databricks/cli/libs/notebook"
	"github.com/databricks/cli/libs/sync"
)

// resolveRefreshSelectors separates table names from source files and translates local files to workspace paths.
func resolveRefreshSelectors(ctx context.Context, b *bundle.Bundle, selectors []string) (tables, paths []string, err error) {
	var remoteNames map[string]string
	for _, selector := range selectors {
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
		if statErr != nil && !errors.Is(statErr, fs.ErrNotExist) {
			return nil, nil, fmt.Errorf("cannot read refresh selector %q: %w", selector, statErr)
		}
		// Preserve table identifiers such as schema.sql when no local file exists.
		looksLikeFile := strings.ContainsAny(selector, `/\`)
		if errors.Is(statErr, fs.ErrNotExist) && !looksLikeFile {
			tables = append(tables, selector)
			continue
		}
		if statErr != nil {
			return nil, nil, fmt.Errorf("source file %q not found; use a table name or an existing source file", selector)
		}
		if info.IsDir() {
			if !looksLikeFile {
				tables = append(tables, selector)
				continue
			}
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
		if remoteNames == nil {
			remoteNames, err = refreshSourcePaths(ctx, b)
			if err != nil {
				return nil, nil, err
			}
		}
		remoteName, ok := remoteNames[filepath.ToSlash(rel)]
		if !ok {
			if _, _, err := notebook.DetectWithFS(b.SyncRoot, filepath.ToSlash(rel)); err != nil {
				return nil, nil, fmt.Errorf("cannot identify source file %q: %w", selector, err)
			}
			return nil, nil, fmt.Errorf("source file %q is excluded from bundle sync; include it and run 'databricks pipelines deploy' first", selector)
		}
		remoteRoot := b.Config.Workspace.FilePath
		if b.IsSourceLinkedDeployment() {
			remoteRoot = filepath.ToSlash(b.SyncRootPath)
		}
		if strings.Contains(remoteRoot, "${") {
			return nil, nil, fmt.Errorf("cannot resolve the deployed path for %q; use its absolute /Workspace/ path", selector)
		}
		paths = append(paths, path.Join(remoteRoot, remoteName))
	}
	return tables, paths, nil
}

// refreshSourcePaths uses the sync mapping so notebook names and sync exclusions match deployment.
func refreshSourcePaths(ctx context.Context, b *bundle.Bundle) (map[string]string, error) {
	includes, err := b.GetSyncIncludePatterns(ctx)
	if err != nil {
		return nil, err
	}
	list, err := sync.NewFileList(ctx, b.WorktreeRoot, b.SyncRoot, b.Config.Sync.Paths, includes, b.Config.Sync.Exclude)
	if err != nil {
		return nil, err
	}
	files, err := list.Files(ctx)
	if err != nil {
		return nil, err
	}
	state, err := sync.NewSnapshotState(files)
	if err != nil {
		return nil, err
	}
	return state.LocalToRemoteNames, nil
}

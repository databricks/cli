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
	"unicode"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/notebook"
	"github.com/databricks/cli/libs/sync"
)

type refreshSelectionFlag []string

func (f *refreshSelectionFlag) String() string {
	return strings.Join(*f, " ")
}

func (f *refreshSelectionFlag) Type() string {
	return "selectors"
}

func (f *refreshSelectionFlag) Set(value string) error {
	var selectors []string
	start := -1
	quoted := false
	for i, c := range value {
		if !quoted && unicode.IsSpace(c) {
			if start >= 0 {
				selectors = append(selectors, value[start:i])
				start = -1
			}
			continue
		}
		if start < 0 {
			start = i
		}
		switch c {
		case '`':
			quoted = !quoted
		case ',':
			if !quoted {
				return errors.New("comma-separated selectors are no longer supported; separate selectors with spaces inside quotes, or repeat the flag; use backticks for literal commas")
			}
		}
	}
	if quoted {
		return errors.New("unclosed backtick in refresh selector")
	}
	if start >= 0 {
		selectors = append(selectors, value[start:])
	}
	if len(selectors) == 0 {
		return errors.New("expected at least one table or file selector")
	}
	*f = append(*f, selectors...)
	return nil
}

func unquoteRefreshSelector(value string) string {
	var result strings.Builder
	quoted := false
	for i := 0; i < len(value); i++ {
		if value[i] != '`' {
			result.WriteByte(value[i])
			continue
		}
		if quoted && i+1 < len(value) && value[i+1] == '`' {
			result.WriteByte('`')
			i++
		} else {
			quoted = !quoted
		}
	}
	return result.String()
}

// ResolveRefreshSelectors separates table names and workspace file paths, then validates flag combinations.
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
		if selector == "``" {
			return nil, nil, errors.New("table selector requires a table name")
		}
		if table, ok := strings.CutPrefix(selector, "table:"); ok {
			if unquoteRefreshSelector(table) == "" {
				return nil, nil, errors.New("table: requires a table name")
			}
			tables = append(tables, table)
			continue
		}
		file, ok := strings.CutPrefix(selector, "file:")
		if !ok {
			if strings.HasPrefix(selector, "flow:") {
				return nil, nil, errors.New("flow selectors are not supported yet; select tables or files")
			}
			tables = append(tables, selector)
			continue
		}
		file = unquoteRefreshSelector(file)
		if file == "" {
			return nil, nil, errors.New("file: requires a source file path")
		}
		selector = file
		// /Workspace/ distinguishes remote paths from local paths like /Users/... on macOS.
		if strings.HasPrefix(selector, "/Workspace/") {
			paths = append(paths, selector)
			continue
		}

		info, statErr := os.Stat(selector)
		if errors.Is(statErr, fs.ErrNotExist) {
			return nil, nil, fmt.Errorf("source file %q not found; use an existing source file", selector)
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
		if b.IsSourceLinkedDeployment() {
			remoteName := filepath.ToSlash(rel)
			isNotebook, _, err := notebook.DetectWithFS(b.SyncRoot, remoteName)
			if err != nil {
				return nil, nil, fmt.Errorf("cannot identify source file %q: %w", selector, err)
			}
			if isNotebook {
				remoteName = notebook.StripExtension(remoteName)
			}
			paths = append(paths, path.Join(b.RemoteFilePath(), remoteName))
			continue
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

package run

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/libs/vfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRefreshSelectionFlag(t *testing.T) {
	var manyTables, manyFiles []string
	for i := range 50 {
		manyTables = append(manyTables, fmt.Sprintf("table_%d", i))
		manyFiles = append(manyFiles, fmt.Sprintf("file:src/pipeline_%d.py", i))
	}
	tests := []struct {
		name   string
		values []string
		want   []string
		err    string
	}{
		{name: "50 tables", values: []string{strings.Join(manyTables, " ")}, want: manyTables},
		{name: "50 files", values: []string{strings.Join(manyFiles, " ")}, want: manyFiles},
		{name: "single table", values: []string{"orders"}, want: []string{"orders"}},
		{name: "table list", values: []string{"orders payments"}, want: []string{"orders", "payments"}},
		{name: "repeated flag", values: []string{"orders payments", "table:customers"}, want: []string{"orders", "payments", "table:customers"}},
		{name: "file list", values: []string{"file:a.py file:b.py"}, want: []string{"file:a.py", "file:b.py"}},
		{name: "whitespace", values: []string{"  orders\tpayments\ncustomers\u2003returns  "}, want: []string{"orders", "payments", "customers", "returns"}},
		{name: "quoted table", values: []string{"`order details` payments"}, want: []string{"`order details`", "payments"}},
		{name: "qualified identifier", values: []string{"table:`catalog`.`schema`.`order details`"}, want: []string{"table:`catalog`.`schema`.`order details`"}},
		{name: "quoted comma", values: []string{"table:`a,b` orders"}, want: []string{"table:`a,b`", "orders"}},
		{name: "escaped backtick", values: []string{"table:`a`` b,c` orders"}, want: []string{"table:`a`` b,c`", "orders"}},
		{name: "quoted file", values: []string{"file:`./my pipeline.py` file:b.py"}, want: []string{"file:`./my pipeline.py`", "file:b.py"}},
		{name: "empty value", values: []string{""}, err: "expected at least one"},
		{name: "whitespace only", values: []string{" \t "}, err: "expected at least one"},
		{name: "old comma list", values: []string{"orders,payments"}, err: "comma-separated selectors are no longer supported"},
		{name: "comma after quoted name", values: []string{"`orders`,payments"}, err: "comma-separated selectors"},
		{name: "unclosed backtick", values: []string{"orders `payments"}, err: "unclosed backtick"},
		{name: "unclosed escaped backtick", values: []string{"file:`a``"}, err: "unclosed backtick"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var flag refreshSelectionFlag
			for _, value := range tt.values {
				err := flag.Set(value)
				if tt.err != "" {
					require.ErrorContains(t, err, tt.err)
					assert.Empty(t, flag)
					return
				}
				require.NoError(t, err)
			}
			assert.Equal(t, tt.want, []string(flag))
		})
	}
}

func TestResolveRefreshSelectors(t *testing.T) {
	longTable := strings.Repeat("c", 100) + "." + strings.Repeat("s", 100) + "." + strings.Repeat("t", 100)
	tests := []struct {
		name      string
		selectors []string
		tables    []string
		paths     []string
		err       string
	}{
		{name: "no selection"},
		{name: "long qualified table", selectors: []string{longTable}, tables: []string{longTable}},
		{
			name:      "table identifiers",
			selectors: []string{"table1", "schema.table2", "schema.sql", "catalog.schema.sql", "`schema`.`table/with/slashes`", "`schema`.`sql`"},
			tables:    []string{"table1", "schema.table2", "schema.sql", "catalog.schema.sql", "`schema`.`table/with/slashes`", "`schema`.`sql`"},
		},
		{
			name:      "bare table with the same name as a file",
			selectors: []string{"orders"},
			tables:    []string{"orders"},
		},
		{
			name:      "mixed table and file selectors",
			selectors: []string{"table:table1", "file:src/query.sql", "file:src/module.py"},
			tables:    []string{"table1"},
			paths:     []string{"/Workspace/Shared/project/files/src/query.sql", "/Workspace/Shared/project/files/src/module.py"},
		},
		{
			name:      "notebook extension removed",
			selectors: []string{"file:src/notebook.py", "file:src/jupyter.ipynb"},
			paths:     []string{"/Workspace/Shared/project/files/src/notebook", "/Workspace/Shared/project/files/src/jupyter"},
		},
		{
			name:      "workspace path",
			selectors: []string{"file:/Workspace/Shared/remote+part.sql"},
			paths:     []string{"/Workspace/Shared/remote+part.sql"},
		},
		{
			name:      "explicit table identifiers",
			selectors: []string{"table:orders", "table:catalog.schema.orders", "table:`file:orders`", "table:file:orders"},
			tables:    []string{"orders", "catalog.schema.orders", "`file:orders`", "file:orders"},
		},
		{name: "empty quoted table", selectors: []string{"table:``"}, err: "table: requires a table name"},
		{name: "empty bare table", selectors: []string{"``"}, err: "table selector requires a table name"},
		{name: "empty quoted file", selectors: []string{"file:``"}, err: "file: requires a source file path"},
		{name: "empty table", selectors: []string{"table:"}, err: "table: requires a table name"},
		{name: "unsupported flow", selectors: []string{"flow:ingest"}, err: "flow selectors are not supported"},
		{name: "quoted workspace path", selectors: []string{"file:`/Workspace/Shared/my pipeline.py`"}, paths: []string{"/Workspace/Shared/my pipeline.py"}},
		{name: "escaped backtick in workspace path", selectors: []string{"file:`/Workspace/Shared/a``b.py`"}, paths: []string{"/Workspace/Shared/a`b.py"}},
		{name: "local file with spaces", selectors: []string{"file:`src/my query.sql`"}, paths: []string{"/Workspace/Shared/project/files/src/my query.sql"}},
		{name: "empty path", selectors: []string{"file:"}, err: "file: requires a source file path"},
		{name: "missing relative file", selectors: []string{"file:src/missing.sql"}, err: `source file "src/missing.sql" not found`},
		{name: "explicit missing filename", selectors: []string{"file:./missing.sql"}, err: `source file "./missing.sql" not found`},
		{name: "table named like a directory", selectors: []string{"src"}, tables: []string{"src"}},
		{name: "explicit directory", selectors: []string{"file:./src"}, err: "is a directory"},
		{name: "excluded file", selectors: []string{"file:src/excluded.sql"}, err: "is excluded from bundle sync"},
		{name: "invalid notebook", selectors: []string{"file:src/invalid.ipynb"}, err: "invalid Jupyter notebook file"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(root, "orders"), []byte("SELECT 1;\n"), 0o600))
			require.NoError(t, os.MkdirAll(filepath.Join(root, "src"), 0o700))
			for name, content := range map[string]string{
				"query.sql":     "SELECT 1;\n",
				"my query.sql":  "SELECT 1;\n",
				"module.py":     "print('file')\n",
				"notebook.py":   "# Databricks notebook source\nprint('notebook')\n",
				"jupyter.ipynb": `{"cells":[],"metadata":{},"nbformat":4,"nbformat_minor":5}`,
				"excluded.sql":  "SELECT 2;\n",
				"invalid.ipynb": `{}`,
			} {
				require.NoError(t, os.WriteFile(filepath.Join(root, "src", name), []byte(content), 0o600))
			}
			t.Chdir(root)
			b := refreshTestBundle(root)
			b.Config.Sync.Exclude = []string{"src/excluded.sql"}
			tables, paths, err := (&refreshSelectorResolver{bundle: b}).resolve(t.Context(), tt.selectors)
			if tt.err != "" {
				require.ErrorContains(t, err, tt.err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.tables, tables)
			assert.Equal(t, tt.paths, paths)
		})
	}
}

func refreshTestBundle(root string) *bundle.Bundle {
	return &bundle.Bundle{
		BundleRootPath: root,
		SyncRootPath:   root,
		SyncRoot:       vfs.MustNew(root),
		WorktreeRoot:   vfs.MustNew(root),
		Config: config.Root{
			Bundle:    config.Bundle{Name: "test", Target: "default"},
			Workspace: config.Workspace{FilePath: "/Workspace/Shared/project/files"},
			Sync:      config.Sync{Paths: []string{"."}},
		},
	}
}

func TestResolveRefreshSelectorsFromSubdirectory(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "project", "src"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(root, "shared.sql"), []byte("SELECT 1;\n"), 0o600))
	t.Chdir(filepath.Join(root, "project", "src"))
	b := refreshTestBundle(root)
	b.BundleRootPath = filepath.Join(root, "project")
	for _, selector := range []string{"../../shared.sql", filepath.Join(root, "shared.sql")} {
		tables, paths, err := (&refreshSelectorResolver{bundle: b}).resolve(t.Context(), []string{"file:" + selector})
		require.NoError(t, err)
		assert.Empty(t, tables)
		assert.Equal(t, []string{"/Workspace/Shared/project/files/shared.sql"}, paths)
	}
}

func TestResolveRefreshSelectorsOutsideSyncRoot(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(root, "project"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(root, "outside.sql"), []byte("SELECT 1;\n"), 0o600))
	t.Chdir(filepath.Join(root, "project"))
	_, _, err := (&refreshSelectorResolver{bundle: refreshTestBundle(filepath.Join(root, "project"))}).resolve(t.Context(), []string{"file:../outside.sql"})
	require.ErrorContains(t, err, "is outside the bundle sync root")
}

func TestResolveRefreshSelectorsDeploymentMode(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "query.sql"), []byte("SELECT 1;\n"), 0o600))
	t.Chdir(root)
	enabled, disabled := true, false
	localPath := filepath.ToSlash(filepath.Join(root, "query.sql"))
	remotePath := "/Workspace/Shared/project/files/query.sql"
	tests := []struct {
		name         string
		mode         config.SyncMode
		sourceLinked *bool
		want         string
	}{
		{name: "default upload", want: remotePath},
		{name: "in-place", mode: config.SyncModeInPlace, want: localPath},
		{name: "legacy source-linked", sourceLinked: &enabled, want: localPath},
		{name: "overwrite overrides legacy setting", mode: config.SyncModeOverwrite, sourceLinked: &enabled, want: remotePath},
		{name: "in-place overrides legacy setting", mode: config.SyncModeInPlace, sourceLinked: &disabled, want: localPath},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := refreshTestBundle(root)
			b.Config.Workspace.SyncMode = tt.mode
			b.Config.Presets.SourceLinkedDeployment = tt.sourceLinked
			tables, paths, err := (&refreshSelectorResolver{bundle: b}).resolve(t.Context(), []string{"file:query.sql"})
			require.NoError(t, err)
			assert.Empty(t, tables)
			assert.Equal(t, []string{tt.want}, paths)
		})
	}
}

func TestResolveRefreshSelectorsUnresolvedWorkspacePath(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "query.sql"), []byte("SELECT 1;\n"), 0o600))
	t.Chdir(root)
	b := refreshTestBundle(root)
	b.Config.Workspace.FilePath = "${resources.internal_immutable_snapshots.immutable.full_path}/files"
	b.Config.Workspace.SyncMode = config.SyncModeImmutable
	enabled := true
	b.Config.Presets.SourceLinkedDeployment = &enabled
	b.DeploymentBundle.StateDB.OpenWithData("state", dstate.Database{})
	_, _, err := (&refreshSelectorResolver{bundle: b}).resolve(t.Context(), []string{"file:query.sql"})
	require.ErrorContains(t, err, "use its absolute /Workspace/ path")
}

func TestResolveRefreshSelectorsDeployedSnapshot(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"query.sql", "full_query.sql"} {
		require.NoError(t, os.WriteFile(filepath.Join(root, name), []byte("SELECT 2;\n"), 0o600))
	}
	t.Chdir(root)
	b := refreshTestBundle(root)
	b.Config.Workspace.SyncMode = config.SyncModeImmutable
	b.Config.Workspace.FilePath = resources.SnapshotFullPathRef + "/files"
	b.DeploymentBundle.StateDB.OpenWithData("state", dstate.Database{
		State: map[string]dstate.ResourceEntry{
			resources.SnapshotKey: {
				ID:    "deployed-snapshot",
				State: []byte(`{"full_path":"/Workspace/Shared/snapshots/deployed/snapshot"}`),
			},
		},
	})
	opts := PipelineOptions{Refresh: []string{"file:query.sql"}, FullRefresh: []string{"file:full_query.sql"}}
	require.NoError(t, opts.ResolveRefreshSelectors(t.Context(), b))
	assert.Empty(t, opts.Refresh)
	assert.Empty(t, opts.FullRefresh)
	assert.Equal(t, []string{"/Workspace/Shared/snapshots/deployed/snapshot/files/query.sql"}, opts.RefreshByPath)
	assert.Equal(t, []string{"/Workspace/Shared/snapshots/deployed/snapshot/files/full_query.sql"}, opts.FullRefreshByPath)
}

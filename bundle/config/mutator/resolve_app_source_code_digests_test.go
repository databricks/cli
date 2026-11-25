package mutator_test

import (
	"maps"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/vfs"
	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const appFilePath = "/Workspace/Users/someone@example.test/.bundle/test/default/files"

// appSourceDigests writes files (slash-separated paths relative to the bundle root, mapped to
// their content) into a new bundle root, applies the mutator and returns the digest of each app.
func appSourceDigests(t *testing.T, files map[string]string, sync config.Sync, appConfigs map[string]*resources.App) map[string]string {
	dir := t.TempDir()
	for name, content := range files {
		path := filepath.Join(dir, filepath.FromSlash(name))
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	}
	root := vfs.MustNew(dir)
	sync.Paths = []string{"."}
	b := &bundle.Bundle{
		BundleRootPath: dir,
		SyncRootPath:   dir,
		SyncRoot:       root,
		WorktreeRoot:   root,
		Config: config.Root{
			Bundle:    config.Bundle{Target: "default"},
			Workspace: config.Workspace{FilePath: appFilePath},
			Sync:      sync,
			Resources: config.Resources{Apps: appConfigs},
		},
	}

	diags := bundle.Apply(t.Context(), b, mutator.ResolveAppSourceCodeDigests())
	require.NoError(t, diags.Error())

	digests := make(map[string]string, len(appConfigs))
	for name, app := range b.Config.Resources.Apps {
		digests[name] = app.SourceCodeDigest
	}
	return digests
}

func startedApp(sourceCodePath string) *resources.App {
	started := true
	return &resources.App{
		SourceCodePath: sourceCodePath,
		Lifecycle:      &resources.LifecycleWithStarted{Started: &started},
	}
}

func TestResolveAppSourceCodeDigestsHashesFilesUnderSourceCodePath(t *testing.T) {
	files := map[string]string{
		"app/app.py":         "print(1)",
		"app/static/app.css": "body {}",
		"other/notes.txt":    "notes",
	}
	digests := appSourceDigests(t, files, config.Sync{}, map[string]*resources.App{
		"my_app": startedApp(appFilePath + "/app"),
	})

	// File names are relative to source_code_path and sorted.
	assert.Equal(t,
		"sha256:"+contentHash(
			"app.py\x00"+contentHash("print(1)")+"\x00"+
				"static/app.css\x00"+contentHash("body {}")+"\x00",
		),
		digests["my_app"],
	)
}

func TestResolveAppSourceCodeDigestsFollowsSourceContent(t *testing.T) {
	base := map[string]string{
		"app/app.py":        "print(1)",
		"app/data/seed.csv": "a,b",
		"other/notes.txt":   "notes",
	}
	sync := config.Sync{Include: []string{"app/build/*"}, Exclude: []string{"app/data/*"}}
	digest := func(files map[string]string) string {
		return appSourceDigests(t, files, sync, map[string]*resources.App{"my_app": startedApp(appFilePath + "/app")})["my_app"]
	}
	want := digest(base)

	tests := []struct {
		name    string
		files   map[string]string
		changed bool
	}{
		{name: "file content changed", files: map[string]string{"app/app.py": "print(2)"}, changed: true},
		{name: "file added", files: map[string]string{"app/util.py": "x = 1"}, changed: true},
		// An empty content removes the file.
		{name: "file renamed", files: map[string]string{"app/app.py": "", "app/main.py": "print(1)"}, changed: true},
		{name: "file outside source_code_path changed", files: map[string]string{"other/notes.txt": "more notes"}},
		{name: "file matching sync.exclude changed", files: map[string]string{"app/data/seed.csv": "c,d"}},
		{name: "git-ignored file added", files: map[string]string{".gitignore": "*.log\n", "app/debug.log": "x"}},
		{name: "git-ignored file matching sync.include added", files: map[string]string{".gitignore": "build/\n", "app/build/out.js": "x"}, changed: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			files := maps.Clone(base)
			for name, content := range tt.files {
				if content == "" {
					delete(files, name)
				} else {
					files[name] = content
				}
			}
			got := digest(files)
			if tt.changed {
				assert.NotEqual(t, want, got)
			} else {
				assert.Equal(t, want, got)
			}
		})
	}
}

func TestResolveAppSourceCodeDigestsSameContentInAnotherDirectory(t *testing.T) {
	digests := appSourceDigests(t,
		map[string]string{"one/app.py": "print(1)", "two/app.py": "print(1)", "app.py": "print(2)"},
		config.Sync{},
		map[string]*resources.App{
			"one": startedApp(appFilePath + "/one"),
			"two": startedApp(appFilePath + "/two"),
			// The sync root itself: every synced file is part of the source.
			"root": startedApp(appFilePath),
		},
	)

	assert.Equal(t, digests["one"], digests["two"])
	assert.NotEmpty(t, digests["root"])
	assert.NotEqual(t, digests["one"], digests["root"])
}

func TestResolveAppSourceCodeDigestsSkipsAppsTheBundleDoesNotDeploy(t *testing.T) {
	stopped := startedApp(appFilePath + "/app")
	*stopped.Lifecycle.Started = false

	gitSource := startedApp("")
	gitSource.GitSource = &apps.GitSource{Branch: "main"}

	digests := appSourceDigests(t,
		map[string]string{"app/app.py": "print(1)"},
		config.Sync{},
		map[string]*resources.App{
			"started_omitted":   {SourceCodePath: appFilePath + "/app"},
			"started_false":     stopped,
			"git_source":        gitSource,
			"workspace_path":    startedApp("/Workspace/Users/someone@example.test/app"),
			"sibling_of_files":  startedApp(appFilePath + "-other/app"),
			"started_with_code": startedApp(appFilePath + "/app"),
		},
	)

	assert.NotEmpty(t, digests["started_with_code"])
	delete(digests, "started_with_code")
	for name, digest := range digests {
		assert.Empty(t, digest, name)
	}
}

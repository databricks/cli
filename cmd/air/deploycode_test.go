package aircmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeCodeTree writes files (slash-separated path to content) into a new git
// repository and returns a deployment config whose code_source is its root.
func writeCodeTree(t *testing.T, files map[string]string) (*deployConfig, string) {
	t.Helper()
	return codeTreeConfig(t, newTestRepo(t), files)
}

// codeTreeConfig writes files under dir and returns a deployment config whose
// code_source is dir.
func codeTreeConfig(t *testing.T, dir string, files map[string]string) (*deployConfig, string) {
	t.Helper()
	for rel, content := range files {
		writeRepoFile(t, dir, filepath.FromSlash(rel), content)
	}
	entry, task, path := "agent.py", "agent/v1/responses", "/Volumes/main/models/weights/support"
	cfg := &deployConfig{
		Model:       &deployModelConfig{Type: modelTypeCheckpoint, Path: &path, Entry: &entry},
		Task:        &task,
		CodeSource:  &deployCodeSourceConfig{RootPath: dir},
		Environment: &deployEnvironmentConfig{Dependencies: []string{"mlflow==3.5.0"}},
		Compute:     &deployComputeConfig{WorkloadType: "GPU_SMALL", WorkloadSize: "Small"},
	}
	return cfg, filepath.Join(dir, "deploy.yaml")
}

func TestCollectModelCode(t *testing.T) {
	cfg, configPath := writeCodeTree(t, map[string]string{
		"agent.py":                       "import tools.search\n",
		"tools/__init__.py":              "",
		"tools/search.py":                "def search(): ...\n",
		"tools/__pycache__/search.pyc":   "\x00\xff",
		".env":                           "TOKEN=x\n",
		".venv/lib/site.py":              "x = 1\n",
		"notes/.draft.py":                "y = 2\n",
		"deploy.yaml":                    "endpoint: e\n",
		"configs/prompts/system.md":      "You are helpful.\n",
		"configs/prompts/__init__.py.md": "not hidden\n",
	})

	code, err := collectEntryCode(t.Context(), cfg, configPath)
	require.NoError(t, err)
	assert.Equal(t, "agent.py", code.EntryFile)
	assert.Equal(t, "import tools.search\n", code.EntrySource)

	var paths []string
	for _, f := range code.Files {
		paths = append(paths, f.Path)
	}
	assert.ElementsMatch(t, []string{
		"configs/prompts/__init__.py.md",
		"configs/prompts/system.md",
		"deploy.yaml",
		"tools/__init__.py",
		"tools/search.py",
	}, paths)
}

func TestCollectModelCodeIsDeterministic(t *testing.T) {
	cfg, configPath := writeCodeTree(t, map[string]string{
		"agent.py": "x = 1\n", "b.py": "b\n", "a/c.py": "c\n", "a.py": "a\n",
	})
	first, err := collectEntryCode(t.Context(), cfg, configPath)
	require.NoError(t, err)
	second, err := collectEntryCode(t.Context(), cfg, configPath)
	require.NoError(t, err)
	assert.Equal(t, first, second)
}

func TestCollectModelCodeIncludePaths(t *testing.T) {
	cfg, configPath := writeCodeTree(t, map[string]string{
		"agent.py": "x = 1\n", "tools/a.py": "a\n", "data/big.csv": "1,2\n",
	})
	cfg.CodeSource.IncludePaths = []string{"agent.py", "tools"}
	code, err := collectEntryCode(t.Context(), cfg, configPath)
	require.NoError(t, err)
	require.Len(t, code.Files, 1)
	assert.Equal(t, "tools/a.py", code.Files[0].Path)
}

func TestCollectModelCodeErrors(t *testing.T) {
	tests := []struct {
		name    string
		files   map[string]string
		include []string
		wantErr string
	}{
		{"missing entry", map[string]string{"main.py": "x\n"}, nil, "model.entry agent.py isn't among the files code_source sends"},
		{"entry left out", map[string]string{"agent.py": "x\n", "tools/a.py": "a\n"}, []string{"tools"}, "model.entry agent.py isn't among the files code_source sends"},
		{"binary file", map[string]string{"agent.py": "x\n", "model.bin": "\xff\xfe\x00"}, nil, "code_source file model.bin isn't UTF-8 text"},
		{"unplaceable name", map[string]string{"agent.py": "x\n", "my tools/a.py": "a\n"}, nil, "code_source file my tools/a.py can't be sent"},
		{"too deep", map[string]string{"agent.py": "x\n", strings.Repeat("d/", maxCodePathDepth) + "a.py": "a\n"}, nil, "can't be sent: a path must have at most 16 parts"},
		{"too large", map[string]string{"agent.py": "x\n", "blob.txt": strings.Repeat("a", maxCodePackageBytes)}, nil, "total more than 4194304 bytes"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, configPath := writeCodeTree(t, tt.files)
			cfg.CodeSource.IncludePaths = tt.include
			_, err := collectEntryCode(t.Context(), cfg, configPath)
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestCollectModelCodeLeavesOutIgnoredFiles(t *testing.T) {
	cfg, configPath := writeCodeTree(t, map[string]string{
		"agent.py": "x\n", "credentials.json": "{}\n", "tools/a.py": "a\n",
	})
	require.NoError(t, os.WriteFile(filepath.Join(cfg.CodeSource.RootPath, ".gitignore"), []byte("credentials.json\n"), 0o600))
	code, err := collectEntryCode(t.Context(), cfg, configPath)
	require.NoError(t, err)
	assert.Equal(t, []string{"agent.py", "tools/a.py"}, codeFilePaths(code))
}

func TestCollectModelCodeRefusesTrackedIgnoredFiles(t *testing.T) {
	cfg, configPath := writeCodeTree(t, map[string]string{"agent.py": "x\n", "creds.json": "{}\n"})
	repo := cfg.CodeSource.RootPath
	commitAll(t, repo, "init")
	writeRepoFile(t, repo, ".gitignore", "creds.json\n")
	cfg.CodeSource.IncludePaths = []string{"agent.py", "creds.json"}

	_, err := collectEntryCode(t.Context(), cfg, configPath)
	assert.ErrorContains(t, err, "code_source file creds.json matches a .gitignore rule but is tracked by git")
}

func TestCollectModelCodeOutsideGit(t *testing.T) {
	cfg, configPath := codeTreeConfig(t, t.TempDir(), map[string]string{"agent.py": "x\n", "tools/a.py": "a\n", "credentials.json": "{}\n"})
	cfg.CodeSource.IncludePaths = []string{"agent.py", "tools"}
	code, err := collectEntryCode(t.Context(), cfg, configPath)
	require.NoError(t, err)
	assert.Equal(t, []string{"agent.py", "tools/a.py"}, codeFilePaths(code))
}

func TestCollectModelCodeChecksSizesBeforeReading(t *testing.T) {
	// Read in full, this file would fail as binary; the size check must reject it
	// from its listing first.
	cfg, configPath := writeCodeTree(t, map[string]string{
		"agent.py": "x\n", "data.bin": strings.Repeat("\xff", maxCodePackageBytes+1),
	})
	_, err := collectEntryCode(t.Context(), cfg, configPath)
	assert.ErrorContains(t, err, "total more than 4194304 bytes")
}

func TestCollectModelCodeRejectsSymlink(t *testing.T) {
	cfg, configPath := writeCodeTree(t, map[string]string{"agent.py": "x\n", "real.py": "r\n"})
	dir := filepath.Dir(configPath)
	if err := os.Symlink("real.py", filepath.Join(dir, "link.py")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	_, err := collectEntryCode(t.Context(), cfg, configPath)
	assert.ErrorContains(t, err, "code_source path link.py isn't a regular file")
}

func TestCollectModelCodeTooManyFiles(t *testing.T) {
	files := map[string]string{"agent.py": "x\n"}
	for i := range maxCodeFiles + 1 {
		files[fmt.Sprintf("pkg/m%d.py", i)] = ""
	}
	cfg, configPath := writeCodeTree(t, files)
	_, err := collectEntryCode(t.Context(), cfg, configPath)
	assert.ErrorContains(t, err, "code_source sends 1002 files, but model.entry can bring at most 1000 other files")
}

func TestSentCodePath(t *testing.T) {
	for rel, want := range map[string]bool{
		"agent.py":              true,
		"tools/search.py":       true,
		".env":                  false,
		"tools/.cache/x.py":     false,
		"__pycache__/a.pyc":     false,
		"tools/__pycache__/a.p": false,
		"__pycache__":           true,
	} {
		assert.Equal(t, want, sentCodePath(rel), rel)
	}
}

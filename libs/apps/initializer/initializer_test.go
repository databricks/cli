package initializer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/apps/pkgmanager"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetProjectInitializer(t *testing.T) {
	tests := []struct {
		name     string
		files    map[string]string
		wantType string
	}{
		{
			name:     "nodejs project with package.json",
			files:    map[string]string{"package.json": `{"name": "test"}`},
			wantType: "*initializer.InitializerNodeJs",
		},
		{
			name:     "python project with pyproject.toml",
			files:    map[string]string{"pyproject.toml": "[project]\nname = \"test\""},
			wantType: "*initializer.InitializerPythonUv",
		},
		{
			name:     "python project with requirements.txt",
			files:    map[string]string{"requirements.txt": "flask==2.0.0"},
			wantType: "*initializer.InitializerPythonPip",
		},
		{
			name:     "no recognizable project type",
			files:    map[string]string{"README.md": "# Test"},
			wantType: "",
		},
		{
			name: "nodejs takes precedence over python",
			files: map[string]string{
				"package.json":     `{"name": "test"}`,
				"requirements.txt": "flask==2.0.0",
			},
			wantType: "*initializer.InitializerNodeJs",
		},
		{
			name: "pyproject.toml takes precedence over requirements.txt",
			files: map[string]string{
				"pyproject.toml":   "[project]\nname = \"test\"",
				"requirements.txt": "flask==2.0.0",
			},
			wantType: "*initializer.InitializerPythonUv",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			// Create test files
			for name, content := range tt.files {
				err := os.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0o644)
				require.NoError(t, err)
			}

			// Get initializer with default PM
			init := GetProjectInitializer(tmpDir, pkgmanager.Default())

			if tt.wantType == "" {
				assert.Nil(t, init)
			} else {
				require.NotNil(t, init)
				assert.Equal(t, tt.wantType, getTypeName(init))
			}
		})
	}
}

func TestNextSteps(t *testing.T) {
	tests := []struct {
		name      string
		init      Initializer
		wantSteps string
	}{
		{
			name:      "nodejs with pnpm",
			init:      &InitializerNodeJs{manager: pkgmanager.Manager{Name: "pnpm"}},
			wantSteps: "pnpm run dev",
		},
		{
			name:      "nodejs with npm",
			init:      &InitializerNodeJs{manager: pkgmanager.Manager{Name: "npm"}},
			wantSteps: "npm run dev",
		},
		{
			name:      "python uv",
			init:      &InitializerPythonUv{},
			wantSteps: "uv run", // contains check below
		},
		{
			name:      "python pip",
			init:      &InitializerPythonPip{},
			wantSteps: ".venv", // contains check below
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.init.NextSteps()
			assert.Contains(t, got, tt.wantSteps)
		})
	}
}

func TestInstallCommand(t *testing.T) {
	tests := []struct {
		name       string
		init       Initializer
		wantCmd    string
		wantSubstr []string // substrings that must be present
		notContain string   // substring that must NOT be present
	}{
		{
			name:    "nodejs with pnpm",
			init:    &InitializerNodeJs{manager: pkgmanager.Manager{InstallCommand: "pnpm install --frozen-lockfile"}},
			wantCmd: "pnpm install --frozen-lockfile",
		},
		{
			name:    "nodejs with npm",
			init:    &InitializerNodeJs{manager: pkgmanager.Manager{InstallCommand: "npm ci"}},
			wantCmd: "npm ci",
		},
		{
			name:    "python uv",
			init:    &InitializerPythonUv{},
			wantCmd: "uv sync",
		},
		{
			name:       "python pip",
			init:       &InitializerPythonPip{},
			wantSubstr: []string{"venv .venv", "pip install -r requirements.txt"},
			notContain: "activate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.init.InstallCommand()
			if tt.wantCmd != "" {
				assert.Equal(t, tt.wantCmd, got)
			}
			for _, substr := range tt.wantSubstr {
				assert.Contains(t, got, substr)
			}
			if tt.notContain != "" {
				assert.NotContains(t, got, tt.notContain, "should not contain: "+tt.notContain)
			}
		})
	}
}

func TestSupportsDevRemote(t *testing.T) {
	// Node.js without appkit
	nodejs := &InitializerNodeJs{workDir: "", manager: pkgmanager.Default()}
	assert.False(t, nodejs.SupportsDevRemote())

	// Python initializers never support dev-remote
	pythonUv := &InitializerPythonUv{}
	assert.False(t, pythonUv.SupportsDevRemote())

	pythonPip := &InitializerPythonPip{}
	assert.False(t, pythonPip.SupportsDevRemote())
}

func TestDetectPythonCommand(t *testing.T) {
	tests := []struct {
		name    string
		files   map[string]string
		wantCmd []string
	}{
		{
			name: "command from app.yaml",
			files: map[string]string{
				"app.yaml":         "command: [\"streamlit\", \"run\", \"app.py\"]",
				"requirements.txt": "flask==2.0.0",
			},
			wantCmd: []string{"streamlit", "run", "app.py"},
		},
		{
			name: "detect streamlit from requirements.txt",
			files: map[string]string{
				"requirements.txt": "streamlit==1.0.0\npandas",
			},
			wantCmd: []string{"streamlit", "run", "app.py"},
		},
		{
			name: "default to python app.py",
			files: map[string]string{
				"requirements.txt": "flask==2.0.0",
			},
			wantCmd: []string{"python", "app.py"},
		},
		{
			name:    "empty directory defaults to python",
			files:   map[string]string{},
			wantCmd: []string{"python", "app.py"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			for name, content := range tt.files {
				err := os.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0o644)
				require.NoError(t, err)
			}

			cmd := detectPythonCommand(tmpDir)
			assert.Equal(t, tt.wantCmd, cmd)
		})
	}
}

func getTypeName(i Initializer) string {
	switch i.(type) {
	case *InitializerNodeJs:
		return "*initializer.InitializerNodeJs"
	case *InitializerPythonUv:
		return "*initializer.InitializerPythonUv"
	case *InitializerPythonPip:
		return "*initializer.InitializerPythonPip"
	default:
		return ""
	}
}

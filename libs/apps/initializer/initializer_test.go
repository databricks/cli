package initializer

import (
	"os"
	"path/filepath"
	"testing"

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
			// Create temp directory
			tmpDir, err := os.MkdirTemp("", "initializer-test-*")
			require.NoError(t, err)
			defer os.RemoveAll(tmpDir)

			// Create test files
			for name, content := range tt.files {
				err := os.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0o644)
				require.NoError(t, err)
			}

			// Get initializer
			init := GetProjectInitializer(tmpDir)

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
	nodejs := &InitializerNodeJs{}
	assert.Equal(t, "npm run dev", nodejs.NextSteps())

	pythonUv := &InitializerPythonUv{}
	assert.Contains(t, pythonUv.NextSteps(), "uv run")

	pythonPip := &InitializerPythonPip{}
	assert.Contains(t, pythonPip.NextSteps(), "uv run")
}

func TestSupportsDevRemote(t *testing.T) {
	// Node.js without appkit
	nodejs := &InitializerNodeJs{workDir: ""}
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
			tmpDir, err := os.MkdirTemp("", "python-cmd-test-*")
			require.NoError(t, err)
			defer os.RemoveAll(tmpDir)

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

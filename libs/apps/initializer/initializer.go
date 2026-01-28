package initializer

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"
)

// InitResult contains the outcome of an initialization operation.
type InitResult struct {
	Success bool
	Message string
	Error   error
}

// Initializer defines the interface for project initialization strategies.
type Initializer interface {
	// Initialize runs the setup steps for the project type.
	Initialize(ctx context.Context, workDir string) *InitResult

	// NextSteps returns the next steps message for this project type.
	NextSteps() string

	// RunDev starts the local development server.
	RunDev(ctx context.Context, workDir string) error

	// SupportsDevRemote returns true if dev-remote mode is supported.
	SupportsDevRemote() bool
}

// GetProjectInitializer returns the appropriate initializer based on project type.
// Detection order: package.json (Node.js), pyproject.toml (Python/uv), requirements.txt (Python/pip).
// Returns nil if no initializer is applicable.
func GetProjectInitializer(workDir string) Initializer {
	// Check for Node.js project (package.json exists)
	if fileExists(filepath.Join(workDir, "package.json")) {
		return &InitializerNodeJs{workDir: workDir}
	}

	// Check for Python project with pyproject.toml (use uv)
	if fileExists(filepath.Join(workDir, "pyproject.toml")) {
		return &InitializerPythonUv{}
	}

	// Check for Python project with requirements.txt (use pip + venv)
	if fileExists(filepath.Join(workDir, "requirements.txt")) {
		return &InitializerPythonPip{}
	}

	return nil
}

// fileExists checks if a file exists at the given path.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// appYaml represents the structure of app.yaml for parsing the command.
type appYaml struct {
	Command []string `yaml:"command"`
}

// getAppCommand reads the command from app.yaml if it exists.
// Returns the command as a slice of strings, or nil if not found.
func getAppCommand(workDir string) []string {
	appYamlPath := filepath.Join(workDir, "app.yaml")
	data, err := os.ReadFile(appYamlPath)
	if err != nil {
		return nil
	}

	var config appYaml
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil
	}

	return config.Command
}

// detectPythonCommand determines the command to run a Python app.
// Priority: app.yaml command > detect streamlit > default to python app.py
func detectPythonCommand(workDir string) []string {
	// First, check app.yaml
	if cmd := getAppCommand(workDir); len(cmd) > 0 {
		return cmd
	}

	// Check if streamlit is in requirements.txt or pyproject.toml
	if hasStreamlit(workDir) {
		return []string{"streamlit", "run", "app.py"}
	}

	// Default to python app.py
	return []string{"python", "app.py"}
}

// hasStreamlit checks if streamlit is a dependency.
func hasStreamlit(workDir string) bool {
	// Check requirements.txt
	reqPath := filepath.Join(workDir, "requirements.txt")
	if data, err := os.ReadFile(reqPath); err == nil {
		if strings.Contains(strings.ToLower(string(data)), "streamlit") {
			return true
		}
	}

	// Check pyproject.toml (simple check)
	pyprojectPath := filepath.Join(workDir, "pyproject.toml")
	if data, err := os.ReadFile(pyprojectPath); err == nil {
		if strings.Contains(strings.ToLower(string(data)), "streamlit") {
			return true
		}
	}

	return false
}

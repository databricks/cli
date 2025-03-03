package patchwheel

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// minimalPythonProject returns a map of file paths to their contents for a minimal Python project.
func minimalPythonProject() map[string]string {
	return map[string]string{
		// pyproject.toml using modern Python packaging
		"pyproject.toml": `[project]
name = "myproj"
version = "0.1.0"

[build-system]
requires = ["setuptools>=61.0.0", "wheel"]
build-backend = "setuptools.build_meta"

[tool.setuptools.packages.find]
where = ["."]
`,
		// A simple module with a __version__.
		"myproj/__init__.py": `__version__ = "0.1.0"
def hello():
    return "Hello, world!"
`,
	}
}

func writeProjectFiles(baseDir string, files map[string]string) error {
	for path, content := range files {
		fullPath := filepath.Join(baseDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			return err
		}
		if err := ioutil.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			return err
		}
	}
	return nil
}

// runCmd runs a command in the given directory and returns its combined output.
func runCmd(dir, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return out.String(), err
}

// TestPatchWheel tests PatchWheel with several Python versions.
func TestPatchWheel(t *testing.T) {
	pythonVersions := []string{"python3.9", "python3.10", "python3.11", "python3.12"}
	for _, py := range pythonVersions {
		t.Run(py, func(t *testing.T) {
			// Create a temporary directory for the project
			tempDir := t.TempDir()

			// Write minimal Python project files.
			projFiles := minimalPythonProject()
			if err := writeProjectFiles(tempDir, projFiles); err != nil {
				t.Fatal(err)
			}

			// Create a virtual environment using uv
			if out, err := runCmd(tempDir, "uv", "venv", "--python", py, "venv"); err != nil {
				t.Fatalf("uv venv creation failed: %v, output: %s", err, out)
			}

			// Determine the pip and python paths inside the venv.
			venvBin := filepath.Join(venvDir, "bin")
			pyExec := filepath.Join(venvBin, "python")
			pipExec := filepath.Join(venvBin, "pip")

			// Install build using uv
			if out, err := runCmd(tempDir, pipExec, "install", "build"); err != nil {
				t.Fatalf("uv pip install failed: %v, output: %s", err, out)
			}

			// Build the wheel.
			if out, err := runCmd(tempDir, pyExec, "-m", "build", "--wheel"); err != nil {
				t.Fatalf("wheel build failed: %v, output: %s", err, out)
			}
			distDir := filepath.Join(tempDir, "dist")
			entries, err := ioutil.ReadDir(distDir)
			if err != nil || len(entries) == 0 {
				t.Fatalf("no wheel built: %v", err)
			}
			// Assume the first wheel is our package.
			origWheel := filepath.Join(distDir, entries[0].Name())

			// Call our PatchWheel function.
			outputDir := filepath.Join(tempDir, "patched")
			if err := os.Mkdir(outputDir, 0o755); err != nil {
				t.Fatal(err)
			}
			patchedWheel, err := PatchWheel(context.Background(), origWheel, outputDir)
			if err != nil {
				t.Fatalf("PatchWheel failed: %v", err)
			}
			t.Logf("origWheel=%s patchedWheel=%s", origWheel, patchedWheel)

			// Install the patched wheel using uv
			if out, err := runCmd(tempDir, pipExec, "install", "--reinstall", patchedWheel); err != nil {
				t.Fatalf("failed to install patched wheel with uv: %v, output: %s", err, out)
			}

			// Run a small command to import the package and print its version.
			cmdOut, err := runCmd(tempDir, pyExec, "-c", "import myproj; print(myproj.__version__)")
			if err != nil {
				t.Fatalf("importing patched package failed: %v, output: %s", err, cmdOut)
			}
			version := strings.TrimSpace(cmdOut)
			if !strings.HasPrefix(version, "0.1.0+") {
				t.Fatalf("expected version to start with 0.1.0+, got %s", version)
			}
			t.Logf("Tested %s: patched version = %s", py, version)
		})
	}
}

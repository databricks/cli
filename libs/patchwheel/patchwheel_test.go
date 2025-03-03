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

// runCmd runs a command in the given directory and fails the test if it fails
func runCmd(t *testing.T, dir, name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err != nil {
		t.Logf("Command failed: %s %s", name, strings.Join(args, " "))
		t.Logf("Output:\n%s", out.String())
		t.Fatal(err)
	}
}

// captureOutput runs a command and returns its output
func captureOutput(t *testing.T, dir, name string, args ...string) string {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err != nil {
		t.Logf("Command failed: %s %s", name, strings.Join(args, " "))
		t.Logf("Output:\n%s", out.String())
		t.Fatal(err)
	}
	return out.String()
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
			runCmd(t, tempDir, "uv", "venv", "--python", py, "venv")

			// Determine the pip and python paths inside the venv.
			venvBin := filepath.Join(tempDir, "venv", "bin")
			pyExec := filepath.Join(venvBin, "python")
			pipExec := filepath.Join(venvBin, "pip")

			// Install build using uv
			runCmd(t, tempDir, pipExec, "install", "build")

			// Build the wheel.
			runCmd(t, tempDir, pyExec, "-m", "build", "--wheel")
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
			runCmd(t, tempDir, pipExec, "install", "--reinstall", patchedWheel)

			// Run a small command to import the package and print its version.
			cmdOut := captureOutput(t, tempDir, pyExec, "-c", "import myproj; print(myproj.__version__)")
			version := strings.TrimSpace(cmdOut)
			if !strings.HasPrefix(version, "0.1.0+") {
				t.Fatalf("expected version to start with 0.1.0+, got %s", version)
			}
			t.Logf("Tested %s: patched version = %s", py, version)
		})
	}
}

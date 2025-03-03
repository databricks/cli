package patchwheel

import (
	"bytes"
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
		// A simple setup.py that uses setuptools.
		"setup.py": `from setuptools import setup, find_packages
setup(
    name="myproj",
    version="0.1.0",
    packages=find_packages(),
)`,
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
			// Create a temporary directory for the project.
			tempDir, err := ioutil.TempDir("", "myproj")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tempDir)

			// Write minimal Python project files.
			projFiles := minimalPythonProject()
			if err := writeProjectFiles(tempDir, projFiles); err != nil {
				t.Fatal(err)
			}

			// Create a virtual environment.
			venvDir := filepath.Join(tempDir, "venv")
			if out, err := runCmd(tempDir, py, "-m", "venv", "venv"); err != nil {
				t.Fatalf("venv creation failed: %v, output: %s", err, out)
			}

			// Determine the pip and python paths inside the venv.
			venvBin := filepath.Join(venvDir, "bin")
			pyExec := filepath.Join(venvBin, "python")
			pipExec := filepath.Join(venvBin, "pip")

			// Upgrade pip and install build.
			if out, err := runCmd(tempDir, pipExec, "install", "--upgrade", "pip", "build"); err != nil {
				t.Fatalf("pip install failed: %v, output: %s", err, out)
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

			// Install the patched wheel.
			if out, err := runCmd(tempDir, pipExec, "install", "--force-reinstall", patchedWheel); err != nil {
				t.Fatalf("failed to install patched wheel: %v, output: %s", err, out)
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

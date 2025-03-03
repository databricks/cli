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

	"github.com/stretchr/testify/require"
)

// Variants -- existing env
// Clean install
// Install unpatched first
// Install patched then another patched

// Variants -- source setup.py vs pyproject
//    Different build backends? setuptools vs hatchling vs flit?

// Different tools? e.g. test poetry? test pdm? test regular pip?

// Variants -- python versions

// Variants --

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
where = ["src"]
`,
		"src/myproj/__init__.py": `
def hello():
    return "Hello, world!"

def print_version():
	from importlib.metadata import version
	print(version("myproj"))
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

func getWheel(t *testing.T, dir string) string {
	pattern := filepath.Join(dir, "*.whl")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatalf("Error matching pattern %s: %v", pattern, err)
	}

	if len(matches) == 0 {
		t.Fatalf("No files found matching %s", pattern)
		return ""
	}

	if len(matches) != 1 {
		t.Fatalf("Too many matches %s: %v", pattern, matches)
		return ""
	}

	return matches[0]
}

// TestPatchWheel tests PatchWheel with several Python versions.
func TestPatchWheel(t *testing.T) {
	pythonVersions := []string{"python3.9", "python3.10", "python3.11", "python3.12"}
	for _, py := range pythonVersions {
		t.Run(py, func(t *testing.T) {
			tempDir := t.TempDir()
			//tempDir, err := os.MkdirTemp("", "pythontestdir")
			//t.Logf("tempDir=%s", tempDir)

			// Write minimal Python project files.
			projFiles := minimalPythonProject()
			if err := writeProjectFiles(tempDir, projFiles); err != nil {
				t.Fatal(err)
			}

			runCmd(t, tempDir, "uv", "venv", "--python", py)

			runCmd(t, tempDir, "uv", "build", "--wheel")
			distDir := filepath.Join(tempDir, "dist")
			origWheel := getWheel(t, distDir)
			//t.Logf("Found origWheel: %s", origWheel)

			patchedWheel, err := PatchWheel(context.Background(), origWheel, distDir)
			require.NoError(t, err)
			//t.Logf("origWheel=%s patchedWheel=%s", origWheel, patchedWheel)

			runCmd(t, tempDir, "uv", "pip", "install", patchedWheel)

			pyExec := filepath.Join(tempDir, ".venv", "bin", "python") // XXX Windows
			cmdOut := captureOutput(t, tempDir, pyExec, "-c", "import myproj; myproj.print_version()")
			version := strings.TrimSpace(cmdOut)
			if !strings.HasPrefix(version, "0.1.0+20") {
				t.Fatalf("expected version to start with 0.1.0+20, got %s", version)
			}
			//t.Logf("Tested %s: patched version = %s", py, version)
		})
	}
}

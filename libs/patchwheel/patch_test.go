package patchwheel

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
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

// verifyVersion checks that the installed package version matches the expected version
// from the patched wheel filename.
func verifyVersion(t *testing.T, tempDir, wheelPath string) {
	// Extract the expected version from the wheel filename
	wheelInfo, err := ParseWheelFilename(wheelPath)
	require.NoError(t, err)
	expectedVersion := wheelInfo.Version

	pyExec := filepath.Join(tempDir, ".venv", "bin", "python") // Handle Windows paths appropriately
	cmdOut := captureOutput(t, tempDir, pyExec, "-c", "import myproj; myproj.print_version()")
	actualVersion := strings.TrimSpace(cmdOut)
	t.Logf("Verified installed version: %s", actualVersion)
	assert.True(t, strings.HasPrefix(actualVersion, "0.1.0+20"), "Version should start with 0.1.0+20, got %s", actualVersion)
	assert.Equal(t, expectedVersion, actualVersion, "Installed version doesn't match expected version from wheel filename")
}

// minimalPythonProject returns a map of file paths to their contents for a minimal Python project.
func minimalPythonProject() map[string]string {
	return map[string]string{
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

func writeProjectFiles(t *testing.T, baseDir string, files map[string]string) {
	for path, content := range files {
		fullPath := filepath.Join(baseDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", filepath.Dir(fullPath), err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatalf("Failed to write file %s: %v", fullPath, err)
		}
	}
}

func runCmd(t *testing.T, dir, name string, args ...string) {
	out := captureOutput(t, dir, name, args...)
	if len(out) > 0 {
		t.Errorf("Output from %s %s:\n%s", name, args, out)
	}
}

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

func TestPatchWheel(t *testing.T) {
	pythonVersions := []string{"python3.9", "python3.10", "python3.11", "python3.12"}
	for _, py := range pythonVersions {
		t.Run(py, func(t *testing.T) {
			tempDir := t.TempDir()

			projFiles := minimalPythonProject()
			writeProjectFiles(t, tempDir, projFiles)

			runCmd(t, tempDir, "uv", "venv", "-q", "--python", py)

			runCmd(t, tempDir, "uv", "build", "-q", "--wheel")
			distDir := filepath.Join(tempDir, "dist")
			origWheel := getWheel(t, distDir)
			// t.Logf("Found origWheel: %s", origWheel)

			// First patch
			patchedWheel, err := PatchWheel(context.Background(), origWheel, distDir)
			require.NoError(t, err)
			// t.Logf("origWheel=%s patchedWheel=%s", origWheel, patchedWheel)

			// Get file info of the patched wheel
			patchedInfo, err := os.Stat(patchedWheel)
			require.NoError(t, err)
			patchedTime := patchedInfo.ModTime()

			// Test idempotency - patching the same wheel again should produce the same result
			// and should not recreate the file (file modification time should remain the same)
			patchedWheel2, err := PatchWheel(context.Background(), origWheel, distDir)
			require.NoError(t, err)
			require.Equal(t, patchedWheel, patchedWheel2, "PatchWheel is not idempotent")

			// Check that the file wasn't recreated
			patchedInfo2, err := os.Stat(patchedWheel2)
			require.NoError(t, err)
			require.Equal(t, patchedTime, patchedInfo2.ModTime(), "File was recreated when it shouldn't have been")

			runCmd(t, tempDir, "uv", "pip", "install", "-q", patchedWheel)

			// Verify the installed version matches what we expect
			verifyVersion(t, tempDir, patchedWheel)

			// TODO: install one more patched wheel (add an option to PatchWheel to add extra to timestamp)
		})
	}
}

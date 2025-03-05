package patchwheel

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	scriptsDir    = getPythonScriptsDir()
	prebuiltWheel = "testdata/my_test_code-0.0.1-py3-none-any.whl"
	emptyZip      = "testdata/empty.zip"
)

func getPythonScriptsDir() string {
	if runtime.GOOS == "windows" {
		return "Scripts"
	}
	return "bin"
}

func getPythonVersions() []string {
	return []string{
		"python3.9",
		"python3.10",
		"python3.11",
		"python3.12",
		"python3.13",
	}
}

func verifyVersion(t *testing.T, tempDir, wheelPath string) {
	wheelInfo, err := ParseWheelFilename(wheelPath)
	require.NoError(t, err)
	expectedVersion := wheelInfo.Version

	pyExec := filepath.Join(tempDir, ".venv", scriptsDir, "python")
	cmdOut := captureOutput(t, tempDir, pyExec, "-c", "import myproj; myproj.print_version()")
	actualVersion := strings.TrimSpace(cmdOut)
	t.Logf("Verified installed version: %s", actualVersion)
	assert.True(t, strings.HasPrefix(actualVersion, "0.1.0+20"), "Version should start with 0.1.0+20, got %s", actualVersion)
	assert.Equal(t, expectedVersion, actualVersion, "Installed version doesn't match expected version from wheel filename")
}

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
	pythonVersions := getPythonVersions()
	for _, py := range pythonVersions {
		t.Run(py, func(t *testing.T) {
			t.Parallel()
			tempDir := t.TempDir()

			projFiles := minimalPythonProject()
			writeProjectFiles(t, tempDir, projFiles)

			runCmd(t, tempDir, "uv", "venv", "-q", "--python", py)

			runCmd(t, tempDir, "uv", "build", "-q", "--wheel")
			distDir := filepath.Join(tempDir, "dist")
			origWheel := getWheel(t, distDir)

			patchedWheel, err := PatchWheel(context.Background(), origWheel, distDir)
			require.NoError(t, err)

			patchedInfo, err := os.Stat(patchedWheel)
			require.NoError(t, err)
			patchedTime := patchedInfo.ModTime()

			// Test idempotency - patching the same wheel again should produce the same result
			// and should not recreate the file (file modification time should remain the same)
			patchedWheel2, err := PatchWheel(context.Background(), origWheel, distDir)
			require.NoError(t, err)
			require.Equal(t, patchedWheel, patchedWheel2, "PatchWheel is not idempotent")

			patchedInfo2, err := os.Stat(patchedWheel2)
			require.NoError(t, err)
			require.Equal(t, patchedTime, patchedInfo2.ModTime(), "File was recreated when it shouldn't have been")

			runCmd(t, tempDir, "uv", "pip", "install", "-q", patchedWheel)
			verifyVersion(t, tempDir, patchedWheel)

			newTime := patchedInfo.ModTime().Add(10 * time.Millisecond)

			err = os.Chtimes(origWheel, newTime, newTime)
			require.NoError(t, err)

			patchedWheel3, err := PatchWheel(context.Background(), origWheel, distDir)
			require.NoError(t, err)
			require.Greater(t, patchedWheel3, patchedWheel)

			runCmd(t, tempDir, "uv", "pip", "install", "-q", patchedWheel3)
			verifyVersion(t, tempDir, patchedWheel3)
		})
	}
}

func TestPrebuilt(t *testing.T) {
	tempDir := t.TempDir()
	ctx := context.Background()

	// Set fixed modification time for deterministic testing
	fixedTime := time.Date(2025, 3, 5, 14, 15, 55, 123456789, time.UTC)
	err := os.Chtimes(prebuiltWheel, fixedTime, fixedTime)
	require.NoError(t, err)

	// With the fixed time, we know exactly what the output filename will be
	expectedVersion := "0.0.1+20250305141555.12"
	expectedFilename := "my_test_code-" + expectedVersion + "-py3-none-any.whl"
	expectedPath := filepath.Join(tempDir, expectedFilename)

	outname, err := PatchWheel(ctx, prebuiltWheel, tempDir)
	require.NoError(t, err)
	require.Equal(t, expectedPath, outname)

	_, err = os.Stat(outname)
	require.NoError(t, err)

	// Verify the contents of the patched wheel
	archive, err := zip.OpenReader(outname)
	require.NoError(t, err)
	defer archive.Close()

	// With fixed time, we know the exact dist-info directory name
	distInfoPrefix := "my_test_code-" + expectedVersion + ".dist-info/"

	// Find METADATA and RECORD files
	var metadataContent, recordContent []byte
	for _, f := range archive.File {
		if f.Name == distInfoPrefix+"METADATA" {
			rc, err := f.Open()
			require.NoError(t, err)
			metadataContent, err = io.ReadAll(rc)
			rc.Close()
			require.NoError(t, err)
		} else if f.Name == distInfoPrefix+"RECORD" {
			rc, err := f.Open()
			require.NoError(t, err)
			recordContent, err = io.ReadAll(rc)
			rc.Close()
			require.NoError(t, err)
		}
	}

	// Verify METADATA contains the expected version
	require.NotNil(t, metadataContent, "METADATA file not found in wheel")
	assert.Contains(t, string(metadataContent), "Version: "+expectedVersion)

	// Verify RECORD contains entries with the correct dist-info prefix
	require.NotNil(t, recordContent, "RECORD file not found in wheel")
	assert.Contains(t, string(recordContent), distInfoPrefix+"METADATA")
	assert.Contains(t, string(recordContent), distInfoPrefix+"RECORD")
}

func errPatchWheel(t *testing.T, name, out string) {
	ctx := context.Background()
	outname, err := PatchWheel(ctx, name, out)
	assert.Error(t, err, "PatchWheel(%s, %s) expected to error", name, out)
	assert.Empty(t, outname)
}

func TestError(t *testing.T) {
	// empty name and dir
	errPatchWheel(t, "", "")

	// empty name
	errPatchWheel(t, "", ".")

	// file not found
	errPatchWheel(t, "not-found.txt", ".")

	// output directory not found
	errPatchWheel(t, prebuiltWheel, "not-found/a/b/c")
}

func TestEmptyZip(t *testing.T) {
	tempDir := t.TempDir()
	errPatchWheel(t, emptyZip, tempDir)
}

func TestNonZip(t *testing.T) {
	tempDir := t.TempDir()

	_, err := os.Stat("patch.go")
	require.NoError(t, err, "file must exist for this test")
	errPatchWheel(t, "patch.go", tempDir)
}

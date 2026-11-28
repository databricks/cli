package patchwheel

import (
	"bytes"
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
	scriptsDir       = getPythonScriptsDir()
	prebuiltWheel    = "testdata/my_test_code-0.0.1-py3-none-any.whl"
	emptyZip         = "testdata/empty.zip"
	vendoredPackages = mustAbs("../vendored_py_packages")
)

func mustAbs(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		panic(err)
	}
	return abs
}

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
	t.Helper()
	wheelInfo, err := ParseWheelFilename(wheelPath)
	require.NoError(t, err)
	expectedVersion := wheelInfo.Version

	pyExec := filepath.Join(tempDir, ".venv", scriptsDir, "python")
	cmdOut := captureOutput(t, tempDir, pyExec, "-c", "import myproj; myproj.print_version()")
	actualVersion := strings.TrimSpace(cmdOut)
	t.Logf("Verified installed version: %s", actualVersion)
	assert.Equal(t, expectedVersion, actualVersion, "Installed version doesn't match expected version from wheel filename")
}

func minimalPythonProject() map[string]string {
	return map[string]string{
		"pyproject.toml": `[project]
name = "myproj"
version = "0.1.0"

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
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err != nil {
		t.Errorf("Command failed: %s %s", name, strings.Join(args, " "))
		t.Errorf("Output:\n%s", out.String())
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

	// Unset any existing virtualenv so that "uv pip install" below is not confused
	// (it prefers virtual env from the environment and fallsback to .venv in current directory)
	t.Setenv("VIRTUAL_ENV", "")
	t.Setenv("UV_OFFLINE", "true")

	for _, py := range pythonVersions {
		t.Run(py, func(t *testing.T) {
			t.Parallel()
			tempDir := t.TempDir()

			projFiles := minimalPythonProject()
			writeProjectFiles(t, tempDir, projFiles)

			runCmd(t, tempDir, "uv", "venv", "-q", "--python", py)

			runCmd(t, tempDir, "uv", "build", "-q", "--wheel", "--no-index", "--find-links", vendoredPackages)
			distDir := filepath.Join(tempDir, "dist")
			origWheel := getWheel(t, distDir)

			patchedWheel, isBuilt, err := PatchWheel(origWheel, distDir)
			require.NoError(t, err)
			require.True(t, isBuilt)

			patchedInfo, err := os.Stat(patchedWheel)
			require.NoError(t, err)
			patchedTime := patchedInfo.ModTime()

			// Test idempotency - patching the same wheel again should produce the same result
			// and should not recreate the file (file modification time should remain the same)
			patchedWheel2, isBuilt2, err := PatchWheel(origWheel, distDir)
			require.NoError(t, err)
			require.False(t, isBuilt2)
			require.Equal(t, patchedWheel, patchedWheel2, "PatchWheel is not idempotent")

			patchedInfo2, err := os.Stat(patchedWheel2)
			require.NoError(t, err)
			require.Equal(t, patchedTime, patchedInfo2.ModTime(), "File was recreated when it shouldn't have been")

			runCmd(t, tempDir, "uv", "pip", "install", "-q", patchedWheel)
			verifyVersion(t, tempDir, patchedWheel)

			newTime := patchedInfo.ModTime().Add(10 * time.Millisecond)

			err = os.Chtimes(origWheel, newTime, newTime)
			require.NoError(t, err)

			patchedWheel3, isBuilt3, err := PatchWheel(origWheel, distDir)
			require.NoError(t, err)
			require.True(t, isBuilt3)
			require.Greater(t, patchedWheel3, patchedWheel)

			// Now use regular pip to re-install the wheel. Install pip from vendored packages (no network needed).
			runCmd(t, tempDir, "uv", "pip", "install", "-q", "--no-index", "--find-links", vendoredPackages, "pip")

			pippath := filepath.Join(tempDir, ".venv", getPythonScriptsDir(), "pip")
			// Stay offline: pip's version check queries the index, and any warning it prints fails runCmd.
			runCmd(t, tempDir, pippath, "install", "-q", "--no-index", "--disable-pip-version-check", patchedWheel3)
			verifyVersion(t, tempDir, patchedWheel3)
		})
	}
}

func errPatchWheel(t *testing.T, name, out string) {
	outname, isBuilt, err := PatchWheel(name, out)
	assert.Error(t, err, "PatchWheel(%s, %s) expected to error", name, out)
	assert.Empty(t, outname)
	assert.False(t, isBuilt)
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

func TestPatchMetadata(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		err      string
	}{
		{
			name:     "headers only",
			input:    "Metadata-Version: 2.1\nName: foo\nVersion: 1.0\n",
			expected: "Metadata-Version: 2.1\nName: foo\nVersion: 1.0+1\n",
		},
		{
			name:     "Version line in description is kept",
			input:    "Name: foo\nVersion: 1.0\n\nVersion: 0.9\nVersion: 1.0\n",
			expected: "Name: foo\nVersion: 1.0+1\n\nVersion: 0.9\nVersion: 1.0\n",
		},
		{
			// bufio.ScanLines drops the \r, so the blank separator line is still empty.
			name:     "CRLF line endings",
			input:    "Name: foo\r\nVersion: 1.0\r\n\r\nVersion: 0.9\r\n",
			expected: "Name: foo\nVersion: 1.0+1\n\nVersion: 0.9\n",
		},
		{
			name:  "different version",
			input: "Name: foo\nVersion: 2.0\n",
			err:   "unexpected version in METADATA: Version: 2.0 (expected 1.0)",
		},
		{
			name:  "no Version header",
			input: "Name: foo\n\nVersion: 1.0\n",
			err:   "expected one Version header in METADATA, found 0",
		},
		{
			name:  "duplicate Version header",
			input: "Name: foo\nVersion: 1.0\nVersion: 1.0\n",
			err:   "expected one Version header in METADATA, found 2",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out, err := patchMetadata(strings.NewReader(tc.input), "1.0", "1.0+1")
			if tc.err != "" {
				assert.EqualError(t, err, tc.err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expected, string(out))
		})
	}
}

func TestPatchRecord(t *testing.T) {
	renamePath := func(name string) string {
		if suffix, ok := strings.CutPrefix(name, "mypkg-1.0.dist-info/"); ok {
			return "mypkg-1.0+1.dist-info/" + suffix
		}
		if suffix, ok := strings.CutPrefix(name, "mypkg-1.0.data/"); ok {
			return "mypkg-1.0+1.data/" + suffix
		}
		return name
	}
	input := strings.Join([]string{
		"mypkg/__init__.py,sha256=abc,0",
		`"mypkg-1.0.data/data/a,b.txt",sha256=def,10`,
		"  \t",
		"",
		"mypkg-1.0.dist-info/METADATA,sha256=old,5",
		"short-row",
		"mypkg-1.0.dist-info/RECORD,,",
	}, "\r\n") + "\r\n"
	expected := strings.Join([]string{
		"mypkg/__init__.py,sha256=abc,0",
		`"mypkg-1.0+1.data/data/a,b.txt",sha256=def,10`,
		"mypkg-1.0+1.dist-info/METADATA,sha256=newhash,42",
		"short-row",
		"mypkg-1.0+1.dist-info/RECORD,,",
	}, "\n") + "\n"

	out, err := patchRecord(strings.NewReader(input), renamePath, "mypkg-1.0+1.dist-info/METADATA", "newhash", 42)
	require.NoError(t, err)
	assert.Equal(t, expected, string(out))
}

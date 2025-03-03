package patchwheel

import (
	"bytes"
	"context"
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
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			return err
		}
	}
	return nil
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

// TestParseWheelFilename tests the ParseWheelFilename function.
func TestParseWheelFilename(t *testing.T) {
	tests := []struct {
		filename         string
		wantDistribution string
		wantVersion      string
		wantTags         []string
		wantErr          bool
	}{
		{
			filename:         "myproj-0.1.0-py3-none-any.whl",
			wantDistribution: "myproj",
			wantVersion:      "0.1.0",
			wantTags:         []string{"py3", "none", "any"},
			wantErr:          false,
		},
		{
			filename:         "myproj-0.1.0+20240303123456-py3-none-any.whl",
			wantDistribution: "myproj",
			wantVersion:      "0.1.0+20240303123456",
			wantTags:         []string{"py3", "none", "any"},
			wantErr:          false,
		},
		{
			filename:         "my-proj-with-hyphens-0.1.0-py3-none-any.whl",
			wantDistribution: "my-proj-with-hyphens",
			wantVersion:      "0.1.0",
			wantTags:         []string{"py3", "none", "any"},
			wantErr:          false,
		},
		{
			filename:         "invalid-filename.txt",
			wantDistribution: "",
			wantVersion:      "",
			wantTags:         nil,
			wantErr:          true,
		},
		{
			filename:         "not-enough-parts-py3.whl",
			wantDistribution: "",
			wantVersion:      "",
			wantTags:         nil,
			wantErr:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			info, err := ParseWheelFilename(tt.filename)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.wantDistribution, info.Distribution)
				require.Equal(t, tt.wantVersion, info.Version)
				require.Equal(t, tt.wantTags, info.Tags)
			}
		})
	}
}

func TestPatchWheel(t *testing.T) {
	pythonVersions := []string{"python3.9", "python3.10", "python3.11", "python3.12"}
	for _, py := range pythonVersions {
		t.Run(py, func(t *testing.T) {
			tempDir := t.TempDir()

			projFiles := minimalPythonProject()
			// AI TODO: modify writeProjectFiles to accept t as first parameter and fail test on errors in internally
			if err := writeProjectFiles(tempDir, projFiles); err != nil {
				t.Fatal(err)
			}

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

			pyExec := filepath.Join(tempDir, ".venv", "bin", "python") // XXX Windows
			cmdOut := captureOutput(t, tempDir, pyExec, "-c", "import myproj; myproj.print_version()")
			version := strings.TrimSpace(cmdOut)
			if !strings.HasPrefix(version, "0.1.0+20") {
				t.Fatalf("expected version to start with 0.1.0+20, got %s", version)
			}
			// t.Logf("Tested %s: patched version = %s", py, version)

			// TODO: install one more patched wheel (add an option to PatchWheel to add extra to timestamp)
		})
	}
}

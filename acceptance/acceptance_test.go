package acceptance_test

import (
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/testdiff"
	"github.com/stretchr/testify/require"
)

var KeepTmp = os.Getenv("KEEP_TMP") != ""

func TestAll(t *testing.T) {
	execPath := BuildCLI(t)
	t.Setenv("CLI", execPath)

	server := StartServer(t)
	AddHandlers(server)
	t.Setenv("DATABRICKS_HOST", fmt.Sprintf("http://127.0.0.1:%d", server.Port))
	t.Setenv("DATABRICKS_TOKEN", "dapi1234")

	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	testDirs := getTests(t)
	require.NotEmpty(t, testDirs)
	for _, dir := range testDirs {
		t.Run(dir, func(t *testing.T) {
			t.Parallel()
			runTest(t, dir)
		})
	}
}

func getTests(t *testing.T) []string {
	testDirs := make(map[string]bool, 128)

	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		name := filepath.Base(path)
		if name == "script" {
			// Presence of 'script' marks a test case in this directory
			testDirs[filepath.Dir(path)] = true
		}
		return nil
	})
	require.NoError(t, err)

	keys := slices.Collect(maps.Keys(testDirs))
	sort.Strings(keys)
	return keys
}

func runTest(t *testing.T, dir string) {
	var tmpDir string
	var err error
	if KeepTmp {
		tempDirBase := filepath.Join(os.TempDir(), "acceptance")
		_ = os.Mkdir(tempDirBase, 0o755)
		tmpDir, err = os.MkdirTemp(tempDirBase, "")
		require.NoError(t, err)
		t.Logf("Created directory: %s", tmpDir)
	} else {
		tmpDir = t.TempDir()
	}

	scriptContents := readMergedScriptContents(t, dir)
	testutil.WriteFile(t, filepath.Join(tmpDir, "script"), scriptContents)

	inputs := make(map[string]bool, 2)
	outputs := make(map[string]bool, 2)
	err = CopyDir(dir, tmpDir, inputs, outputs)
	require.NoError(t, err)

	args := []string{"bash", "-euo", "pipefail", "script"}
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = tmpDir
	outB, err := cmd.CombinedOutput()

	out := formatOutput(string(outB), err)
	out = strings.ReplaceAll(out, os.Getenv("CLI"), "$CLI")
	doComparison(t, filepath.Join(dir, "output.txt"), "script output", out)

	for key := range outputs {
		if key == "output.txt" {
			// handled above
			continue
		}
		pathNew := filepath.Join(tmpDir, key)
		newValBytes, err := os.ReadFile(pathNew)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				t.Errorf("%s: expected to find this file but could not (%s)", key, tmpDir)
			} else {
				t.Errorf("%s: could not read: %s", key, err)
			}
			continue
		}
		pathExpected := filepath.Join(dir, key)
		doComparison(t, pathExpected, pathNew, string(newValBytes))
	}

	// Make sure there are not unaccounted for new files
	files, err := os.ReadDir(tmpDir)
	require.NoError(t, err)

	for _, f := range files {
		name := f.Name()
		if _, ok := inputs[name]; ok {
			continue
		}
		if _, ok := outputs[name]; ok {
			continue
		}
		t.Errorf("Unexpected output: %s", f)
		if strings.HasPrefix(name, "out") {
			// We have a new file starting with "out"
			// Show the contents & support overwrite mode for it:
			pathNew := filepath.Join(tmpDir, name)
			newVal := testutil.ReadFile(t, pathNew)
			doComparison(t, filepath.Join(dir, name), filepath.Join(tmpDir, name), newVal)
		}
	}
}

func doComparison(t *testing.T, pathExpected, pathNew, valueNew string) {
	valueExpected := string(readIfExists(t, pathExpected))
	testdiff.AssertEqualTexts(t, pathExpected, pathNew, valueExpected, valueNew)
	if testdiff.OverwriteMode {
		if valueNew != "" {
			t.Logf("Overwriting: %s", pathExpected)
			testutil.WriteFile(t, pathExpected, valueNew)
		} else {
			t.Logf("Removing: %s", pathExpected)
			_ = os.Remove(pathExpected)
		}
	}
}

// Returns combined script.prepare (root) + script.prepare (parent) + ... + script + ... + script.cleanup (parent) + ...
// Note, cleanups are not executed if main script fails; that's not a huge issue, since it runs it temp dir.
func readMergedScriptContents(t *testing.T, dir string) string {
	scriptContents := testutil.ReadFile(t, filepath.Join(dir, "script"))
	prepares := []string{}
	cleanups := []string{}

	for {
		x := readIfExists(t, filepath.Join(dir, "script.cleanup"))
		if len(x) > 0 {
			cleanups = append(cleanups, string(x))
		}

		x = readIfExists(t, filepath.Join(dir, "script.prepare"))
		if len(x) > 0 {
			prepares = append(prepares, string(x))
		}

		if dir == "" || dir == "." {
			break
		}

		dir = filepath.Dir(dir)
	}

	slices.Reverse(prepares)
	prepares = append(prepares, scriptContents)
	prepares = append(prepares, cleanups...)
	return strings.Join(prepares, "\n")
}

// Note, because "go build" always touches the final binary, even if unchanged,
// this acts as cache breaker for "go test".
func BuildCLI(t *testing.T) string {
	cwd, err := os.Getwd()
	require.NoError(t, err)
	execPath := filepath.Join(cwd, "build", "databricks-cli")
	if runtime.GOOS == "windows" {
		execPath += ".exe"
	}

	start := time.Now()
	args := []string{"go", "build", "-mod", "vendor", "-o", execPath}
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = ".."
	out, err := cmd.CombinedOutput()
	elapsed := time.Since(start)
	t.Logf("%s took %s", args, elapsed)
	require.NoError(t, err, "go build failed: %s: %s\n%s", args, err, out)
	if len(out) > 0 {
		t.Logf("go build output: %s: %s", args, out)
	}

	// Quick check + warm up cache:
	cmd = exec.Command(execPath, "--version")
	out, err = cmd.CombinedOutput()
	require.NoError(t, err, "%s --version failed: %s\n%s", execPath, err, out)
	return execPath
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func formatOutput(out string, err error) string {
	if err == nil {
		return out
	}
	if exiterr, ok := err.(*exec.ExitError); ok {
		exitCode := exiterr.ExitCode()
		out += fmt.Sprintf("\nExit code: %d\n", exitCode)
	} else {
		out += fmt.Sprintf("\nError: %s\n", err)
	}
	return out
}

func readIfExists(t *testing.T, path string) []byte {
	data, err := os.ReadFile(path)
	if err == nil {
		return data
	}

	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("%s: %s", path, err)
	}
	return []byte{}
}

func CopyDir(src, dst string, inputs, outputs map[string]bool) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		name := info.Name()

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		if strings.HasPrefix(name, "out") {
			outputs[relPath] = true
			return nil
		} else {
			inputs[relPath] = true
		}

		if name == "script" {
			return nil
		}

		destPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}

		return copyFile(path, destPath)
	})
}

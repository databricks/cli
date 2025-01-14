package acceptance_test

import (
	"context"
	"errors"
	"fmt"
	"io"
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
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/testdiff"
	"github.com/databricks/databricks-sdk-go"
	"github.com/stretchr/testify/require"
)

var KeepTmp = os.Getenv("KEEP_TMP") != ""

const (
	EntryPointScript = "script"
	CleanupScript    = "script.cleanup"
	PrepareScript    = "script.prepare"
)

var Scripts = map[string]bool{
	EntryPointScript: true,
	CleanupScript:    true,
	PrepareScript:    true,
}

func TestAccept(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)

	coverDir := os.Getenv("CLI_GOCOVERDIR")

	if coverDir != "" {
		require.NoError(t, os.MkdirAll(coverDir, os.ModePerm))
		coverDir, err = filepath.Abs(coverDir)
		require.NoError(t, err)
		t.Logf("Writing coverage to %s", coverDir)
	}

	execPath := BuildCLI(t, cwd, coverDir)
	// $CLI is what test scripts are using
	t.Setenv("CLI", execPath)

	// Make helper scripts available
	t.Setenv("PATH", fmt.Sprintf("%s%c%s", filepath.Join(cwd, "bin"), os.PathListSeparator, os.Getenv("PATH")))

	repls := testdiff.ReplacementsContext{}
	repls.Set(execPath, "$CLI")

	ctx := context.Background()
	cloudEnv := os.Getenv("CLOUD_ENV")

	if cloudEnv == "" {
		server := StartServer(t)
		AddHandlers(server)
		// Redirect API access to local server:
		t.Setenv("DATABRICKS_HOST", server.URL)
		t.Setenv("DATABRICKS_TOKEN", "dapi1234")

		homeDir := t.TempDir()
		// Do not read user's ~/.databrickscfg
		t.Setenv(env.HomeEnvVar(), homeDir)
	}

	workspaceClient, err := databricks.NewWorkspaceClient()
	require.NoError(t, err)

	user, err := workspaceClient.CurrentUser.Me(ctx)
	require.NoError(t, err)
	require.NotNil(t, user)
	testdiff.PrepareReplacementsUser(t, &repls, *user)
	testdiff.PrepareReplacements(t, &repls, workspaceClient)

	testDirs := getTests(t)
	require.NotEmpty(t, testDirs)

	for _, dir := range testDirs {
		t.Run(dir, func(t *testing.T) {
			t.Parallel()
			runTest(t, dir, coverDir, repls)
		})
	}
}

func getTests(t *testing.T) []string {
	testDirs := make([]string, 0, 128)

	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		name := filepath.Base(path)
		if name == EntryPointScript {
			// Presence of 'script' marks a test case in this directory
			testDirs = append(testDirs, filepath.Dir(path))
		}
		return nil
	})
	require.NoError(t, err)

	sort.Strings(testDirs)
	return testDirs
}

func runTest(t *testing.T, dir, coverDir string, repls testdiff.ReplacementsContext) {
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
	testutil.WriteFile(t, filepath.Join(tmpDir, EntryPointScript), scriptContents)

	inputs := make(map[string]bool, 2)
	outputs := make(map[string]bool, 2)
	err = CopyDir(dir, tmpDir, inputs, outputs)
	require.NoError(t, err)

	args := []string{"bash", "-euo", "pipefail", EntryPointScript}
	cmd := exec.Command(args[0], args[1:]...)
	if coverDir != "" {
		// Creating individual coverage directory for each test, because writing to the same one
		// results in sporadic failures like this one (only if tests are running in parallel):
		// +error: coverage meta-data emit failed: writing ... rename .../tmp.covmeta.b3f... .../covmeta.b3f2c...: no such file or directory
		coverDir = filepath.Join(coverDir, strings.ReplaceAll(dir, string(os.PathSeparator), "--"))
		err := os.MkdirAll(coverDir, os.ModePerm)
		require.NoError(t, err)
		cmd.Env = append(os.Environ(), "GOCOVERDIR="+coverDir)
	}
	cmd.Dir = tmpDir
	outB, err := cmd.CombinedOutput()

	out := formatOutput(string(outB), err)
	out = repls.Replace(out)
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
		newVal := repls.Replace(string(newValBytes))
		doComparison(t, pathExpected, pathNew, newVal)
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
			newVal = repls.Replace(newVal)
			doComparison(t, filepath.Join(dir, name), filepath.Join(tmpDir, name), newVal)
		}
	}
}

func doComparison(t *testing.T, pathExpected, pathNew, valueNew string) {
	valueNew = testdiff.NormalizeNewlines(valueNew)
	valueExpected := string(readIfExists(t, pathExpected))
	valueExpected = testdiff.NormalizeNewlines(valueExpected)
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
	scriptContents := testutil.ReadFile(t, filepath.Join(dir, EntryPointScript))

	// Wrap script contents in a subshell such that changing the working
	// directory only affects the main script and not cleanup.
	scriptContents = "(\n" + scriptContents + ")\n"

	prepares := []string{}
	cleanups := []string{}

	for {
		x := readIfExists(t, filepath.Join(dir, CleanupScript))
		if len(x) > 0 {
			cleanups = append(cleanups, string(x))
		}

		x = readIfExists(t, filepath.Join(dir, PrepareScript))
		if len(x) > 0 {
			prepares = append(prepares, string(x))
		}

		if dir == "" || dir == "." {
			break
		}

		dir = filepath.Dir(dir)
		require.True(t, filepath.IsLocal(dir))
	}

	slices.Reverse(prepares)
	prepares = append(prepares, scriptContents)
	prepares = append(prepares, cleanups...)
	return strings.Join(prepares, "\n")
}

func BuildCLI(t *testing.T, cwd, coverDir string) string {
	execPath := filepath.Join(cwd, "build", "databricks")
	if runtime.GOOS == "windows" {
		execPath += ".exe"
	}

	start := time.Now()
	args := []string{
		"go", "build",
		"-mod", "vendor",
		"-o", execPath,
	}

	if coverDir != "" {
		args = append(args, "-cover")
	}

	if runtime.GOOS == "windows" {
		// Get this error on my local Windows:
		// error obtaining VCS status: exit status 128
		// Use -buildvcs=false to disable VCS stamping.
		args = append(args, "-buildvcs=false")
	}

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
	t.Logf("%s --version: %s", execPath, out)
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

		if _, ok := Scripts[name]; ok {
			return nil
		}

		destPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}

		return copyFile(path, destPath)
	})
}

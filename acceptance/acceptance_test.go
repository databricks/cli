package acceptance_test

import (
	"context"
	"errors"
	"flag"
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

var KeepTmp bool

// In order to debug CLI running under acceptance test, set this to full subtest name, e.g. "bundle/variables/empty"
// Then install your breakpoints and click "debug test" near TestAccept in VSCODE.
// example: var SingleTest = "bundle/variables/empty"
var SingleTest = ""

// If enabled, instead of compiling and running CLI externally, we'll start in-process server that accepts and runs
// CLI commands. The $CLI in test scripts is a helper that just forwards command-line arguments to this server (see bin/callserver.py).
// Also disables parallelism in tests.
var InprocessMode bool

func init() {
	flag.BoolVar(&InprocessMode, "inprocess", SingleTest != "", "Run CLI in the same process as test (for debugging)")
	flag.BoolVar(&KeepTmp, "keeptmp", false, "Do not delete TMP directory after run")
}

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
	testAccept(t, InprocessMode, SingleTest)
}

func TestInprocessMode(t *testing.T) {
	if InprocessMode {
		t.Skip("Already tested by TestAccept")
	}
	require.Equal(t, 1, testAccept(t, true, "selftest"))
}

func testAccept(t *testing.T, InprocessMode bool, singleTest string) int {
	repls := testdiff.ReplacementsContext{}
	cwd, err := os.Getwd()
	require.NoError(t, err)

	coverDir := os.Getenv("CLI_GOCOVERDIR")

	if coverDir != "" {
		require.NoError(t, os.MkdirAll(coverDir, os.ModePerm))
		coverDir, err = filepath.Abs(coverDir)
		require.NoError(t, err)
		t.Logf("Writing coverage to %s", coverDir)
	}

	execPath := ""

	if InprocessMode {
		cmdServer := StartCmdServer(t)
		t.Setenv("CMD_SERVER_URL", cmdServer.URL)
		execPath = filepath.Join(cwd, "bin", "callserver.py")
	} else {
		execPath = BuildCLI(t, cwd, coverDir)
	}

	t.Setenv("CLI", execPath)
	repls.SetPath(execPath, "$CLI")

	// Make helper scripts available
	t.Setenv("PATH", fmt.Sprintf("%s%c%s", filepath.Join(cwd, "bin"), os.PathListSeparator, os.Getenv("PATH")))

	tempHomeDir := t.TempDir()
	repls.SetPath(tempHomeDir, "$TMPHOME")
	t.Logf("$TMPHOME=%v", tempHomeDir)

	// Prevent CLI from downloading terraform in each test:
	t.Setenv("DATABRICKS_TF_EXEC_PATH", tempHomeDir)

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
	testdiff.PrepareReplacementsWorkspaceClient(t, &repls, workspaceClient)
	testdiff.PrepareReplacementsUUID(t, &repls)

	testDirs := getTests(t)
	require.NotEmpty(t, testDirs)

	if singleTest != "" {
		testDirs = slices.DeleteFunc(testDirs, func(n string) bool {
			return n != singleTest
		})
		require.NotEmpty(t, testDirs, "singleTest=%#v did not match any tests\n%#v", singleTest, testDirs)
	}

	for _, dir := range testDirs {
		testName := strings.ReplaceAll(dir, "\\", "/")
		t.Run(testName, func(t *testing.T) {
			if !InprocessMode {
				t.Parallel()
			}

			runTest(t, dir, coverDir, repls.Clone())
		})
	}

	return len(testDirs)
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
	config, configPath := LoadConfig(t, dir)

	isEnabled, isPresent := config.GOOS[runtime.GOOS]
	if isPresent && !isEnabled {
		t.Skipf("Disabled via GOOS.%s setting in %s", runtime.GOOS, configPath)
	}

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

	repls.SetPathWithParents(tmpDir, "$TMPDIR")
	repls.Repls = append(repls.Repls, config.Repls...)

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

	// Write combined output to a file
	out, err := os.Create(filepath.Join(tmpDir, "output.txt"))
	require.NoError(t, err)
	cmd.Stdout = out
	cmd.Stderr = out
	cmd.Dir = tmpDir
	err = cmd.Run()

	// Include exit code in output (if non-zero)
	formatOutput(out, err)
	require.NoError(t, out.Close())

	printedRepls := false

	// Compare expected outputs
	for relPath := range outputs {
		doComparison(t, repls, dir, tmpDir, relPath, &printedRepls)
	}

	// Make sure there are not unaccounted for new files
	files := ListDir(t, tmpDir)
	for _, relPath := range files {
		if _, ok := inputs[relPath]; ok {
			continue
		}
		if _, ok := outputs[relPath]; ok {
			continue
		}
		if strings.HasPrefix(relPath, "out") {
			// We have a new file starting with "out"
			// Show the contents & support overwrite mode for it:
			doComparison(t, repls, dir, tmpDir, relPath, &printedRepls)
		}
	}
}

func doComparison(t *testing.T, repls testdiff.ReplacementsContext, dirRef, dirNew, relPath string, printedRepls *bool) {
	pathRef := filepath.Join(dirRef, relPath)
	pathNew := filepath.Join(dirNew, relPath)
	bufRef, okRef := readIfExists(t, pathRef)
	bufNew, okNew := readIfExists(t, pathNew)
	if !okRef && !okNew {
		t.Errorf("Both files are missing: %s, %s", pathRef, pathNew)
		return
	}

	valueRef := testdiff.NormalizeNewlines(string(bufRef))
	valueNew := testdiff.NormalizeNewlines(string(bufNew))

	// Apply replacements to the new value only.
	// The reference value is stored after applying replacements.
	valueNew = repls.Replace(valueNew)

	// The test did not produce an expected output file.
	if okRef && !okNew {
		t.Errorf("Missing output file: %s", relPath)
		testdiff.AssertEqualTexts(t, pathRef, pathNew, valueRef, valueNew)
		if testdiff.OverwriteMode {
			t.Logf("Removing output file: %s", relPath)
			require.NoError(t, os.Remove(pathRef))
		}
		return
	}

	// The test produced an unexpected output file.
	if !okRef && okNew {
		t.Errorf("Unexpected output file: %s", relPath)
		testdiff.AssertEqualTexts(t, pathRef, pathNew, valueRef, valueNew)
		if testdiff.OverwriteMode {
			t.Logf("Writing output file: %s", relPath)
			testutil.WriteFile(t, pathRef, valueNew)
		}
		return
	}

	// Compare the reference and new values.
	equal := testdiff.AssertEqualTexts(t, pathRef, pathNew, valueRef, valueNew)
	if !equal && testdiff.OverwriteMode {
		t.Logf("Overwriting existing output file: %s", relPath)
		testutil.WriteFile(t, pathRef, valueNew)
	}

	if !equal && printedRepls != nil && !*printedRepls {
		*printedRepls = true
		var items []string
		for _, item := range repls.Repls {
			items = append(items, fmt.Sprintf("REPL %s => %s", item.Old, item.New))
		}
		t.Log("Available replacements:\n" + strings.Join(items, "\n"))
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
		x, ok := readIfExists(t, filepath.Join(dir, CleanupScript))
		if ok {
			cleanups = append(cleanups, string(x))
		}

		x, ok = readIfExists(t, filepath.Join(dir, PrepareScript))
		if ok {
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

func formatOutput(w io.Writer, err error) {
	if err == nil {
		return
	}
	if exiterr, ok := err.(*exec.ExitError); ok {
		exitCode := exiterr.ExitCode()
		fmt.Fprintf(w, "\nExit code: %d\n", exitCode)
	} else {
		fmt.Fprintf(w, "\nError: %s\n", err)
	}
}

func readIfExists(t *testing.T, path string) ([]byte, bool) {
	data, err := os.ReadFile(path)
	if err == nil {
		return data, true
	}

	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("%s: %s", path, err)
	}
	return []byte{}, false
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

		if strings.HasPrefix(relPath, "out") {
			if !info.IsDir() {
				outputs[relPath] = true
			}
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

func ListDir(t *testing.T, src string) []string {
	var files []string
	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Do not FailNow here.
			// The output comparison is happening after this call which includes output.txt which
			// includes errors printed by commands which include explanation why a given file cannot be read.
			t.Errorf("Error when listing %s: path=%s: %s", src, path, err)
			return nil
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		files = append(files, relPath)
		return nil
	})
	if err != nil {
		t.Errorf("Failed to list %s: %s", src, err)
	}
	return files
}

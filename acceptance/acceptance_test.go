package acceptance_test

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/testdiff"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/stretchr/testify/require"
)

var (
	KeepTmp     bool
	NoRepl      bool
	VerboseTest bool = os.Getenv("VERBOSE_TEST") != ""
)

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
	flag.BoolVar(&NoRepl, "norepl", false, "Do not apply any replacements (for debugging)")
}

const (
	EntryPointScript = "script"
	CleanupScript    = "script.cleanup"
	PrepareScript    = "script.prepare"
	MaxFileSize      = 100_000
	// Filename to save replacements to (used by diff.py)
	ReplsFile = "repls.json"
)

var Scripts = map[string]bool{
	EntryPointScript: true,
	CleanupScript:    true,
	PrepareScript:    true,
}

var Ignored = map[string]bool{
	ReplsFile: true,
}

func TestAccept(t *testing.T) {
	testAccept(t, InprocessMode, SingleTest)
}

func TestInprocessMode(t *testing.T) {
	if InprocessMode {
		t.Skip("Already tested by TestAccept")
	}
	require.Equal(t, 1, testAccept(t, true, "selftest/basic"))
	require.Equal(t, 1, testAccept(t, true, "selftest/server"))
}

func testAccept(t *testing.T, InprocessMode bool, singleTest string) int {
	repls := testdiff.ReplacementsContext{}
	cwd, err := os.Getwd()
	require.NoError(t, err)

	buildDir := filepath.Join(cwd, "build", fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH))

	// Download terraform and provider and create config; this also creates build directory.
	RunCommand(t, []string{"python3", filepath.Join(cwd, "install_terraform.py"), "--targetdir", buildDir}, ".")

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
		execPath = BuildCLI(t, buildDir, coverDir)
	}

	t.Setenv("CLI", execPath)
	repls.SetPath(execPath, "[CLI]")

	// Make helper scripts available
	t.Setenv("PATH", fmt.Sprintf("%s%c%s", filepath.Join(cwd, "bin"), os.PathListSeparator, os.Getenv("PATH")))

	tempHomeDir := t.TempDir()
	repls.SetPath(tempHomeDir, "[TMPHOME]")
	t.Logf("$TMPHOME=%v", tempHomeDir)

	// Make use of uv cache; since we set HomeEnvVar to temporary directory, it is not picked up automatically
	uvCache := getUVDefaultCacheDir(t)
	t.Setenv("UV_CACHE_DIR", uvCache)

	cloudEnv := os.Getenv("CLOUD_ENV")

	if cloudEnv == "" {
		defaultServer := testserver.New(t)
		AddHandlers(defaultServer)
		t.Setenv("DATABRICKS_DEFAULT_HOST", defaultServer.URL)

		homeDir := t.TempDir()
		// Do not read user's ~/.databrickscfg
		t.Setenv(env.HomeEnvVar(), homeDir)
	}

	terraformrcPath := filepath.Join(buildDir, ".terraformrc")
	t.Setenv("TF_CLI_CONFIG_FILE", terraformrcPath)
	t.Setenv("DATABRICKS_TF_CLI_CONFIG_FILE", terraformrcPath)
	repls.SetPath(terraformrcPath, "[DATABRICKS_TF_CLI_CONFIG_FILE]")

	terraformExecPath := filepath.Join(buildDir, "terraform")
	if runtime.GOOS == "windows" {
		terraformExecPath += ".exe"
	}
	t.Setenv("DATABRICKS_TF_EXEC_PATH", terraformExecPath)
	t.Setenv("TERRAFORM", terraformExecPath)
	repls.SetPath(terraformExecPath, "[TERRAFORM]")

	// do it last so that full paths match first:
	repls.SetPath(buildDir, "[BUILD_DIR]")

	testdiff.PrepareReplacementsDevVersion(t, &repls)
	testdiff.PrepareReplacementSdkVersion(t, &repls)
	testdiff.PrepareReplacementsGoVersion(t, &repls)

	repls.SetPath(cwd, "[TESTROOT]")

	repls.Repls = append(repls.Repls, testdiff.Replacement{Old: regexp.MustCompile("dbapi[0-9a-f]+"), New: "[DATABRICKS_TOKEN]"})

	testDirs := getTests(t)
	require.NotEmpty(t, testDirs)

	if singleTest != "" {
		testDirs = slices.DeleteFunc(testDirs, func(n string) bool {
			return n != singleTest
		})
		require.NotEmpty(t, testDirs, "singleTest=%#v did not match any tests\n%#v", singleTest, testDirs)
	}

	for _, dir := range testDirs {
		t.Run(dir, func(t *testing.T) {
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
			testName := filepath.ToSlash(filepath.Dir(path))
			testDirs = append(testDirs, testName)
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

	cloudEnv := os.Getenv("CLOUD_ENV")
	if config.LocalOnly && cloudEnv != "" {
		t.Skipf("Disabled via LocalOnly setting in %s (CLOUD_ENV=%s)", configPath, cloudEnv)
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

	repls.SetPathWithParents(tmpDir, "[TMPDIR]")

	scriptContents := readMergedScriptContents(t, dir)
	testutil.WriteFile(t, filepath.Join(tmpDir, EntryPointScript), scriptContents)

	inputs := make(map[string]bool, 2)
	outputs := make(map[string]bool, 2)
	err = CopyDir(dir, tmpDir, inputs, outputs)
	require.NoError(t, err)

	args := []string{"bash", "-euo", "pipefail", EntryPointScript}
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Env = os.Environ()

	var workspaceClient *databricks.WorkspaceClient
	var user iam.User

	// Start a new server with a custom configuration if the acceptance test
	// specifies a custom server stubs.
	var server *testserver.Server

	if cloudEnv == "" {
		// Start a new server for this test if either:
		// 1. A custom server spec is defined in the test configuration.
		// 2. The test is configured to record requests and assert on them. We need
		//    a duplicate of the default server to record requests because the default
		//    server otherwise is a shared resource.

		databricksLocalHost := os.Getenv("DATABRICKS_DEFAULT_HOST")

		if len(config.Server) > 0 || config.RecordRequests {
			server = testserver.New(t)
			server.RecordRequests = config.RecordRequests
			server.IncludeRequestHeaders = config.IncludeRequestHeaders

			// We want later stubs takes precedence, because then leaf configs take precedence over parent directory configs
			// In gorilla/mux earlier handlers take precedence, so we need to reverse the order
			slices.Reverse(config.Server)

			for _, stub := range config.Server {
				require.NotEmpty(t, stub.Pattern)
				items := strings.Split(stub.Pattern, " ")
				require.Len(t, items, 2)
				server.Handle(items[0], items[1], func(req testserver.Request) any {
					return stub.Response
				})
			}

			// The earliest handlers take precedence, add default handlers last
			AddHandlers(server)
			databricksLocalHost = server.URL
		}

		// Each local test should use a new token that will result into a new fake workspace,
		// so that test don't interfere with each other.
		tokenSuffix := strings.ReplaceAll(uuid.NewString(), "-", "")
		config := databricks.Config{
			Host:  databricksLocalHost,
			Token: "dbapi" + tokenSuffix,
		}
		workspaceClient, err = databricks.NewWorkspaceClient(&config)
		require.NoError(t, err)

		cmd.Env = append(cmd.Env, "DATABRICKS_HOST="+config.Host)
		cmd.Env = append(cmd.Env, "DATABRICKS_TOKEN="+config.Token)

		// For the purposes of replacements, use testUser.
		// Note, users might have overriden /api/2.0/preview/scim/v2/Me but that should not affect the replacement:
		user = testUser
	} else {
		// Use whatever authentication mechanism is configured by the test runner.
		workspaceClient, err = databricks.NewWorkspaceClient(&databricks.Config{})
		require.NoError(t, err)
		pUser, err := workspaceClient.CurrentUser.Me(context.Background())
		require.NoError(t, err, "Failed to get current user")
		user = *pUser
	}

	testdiff.PrepareReplacementsUser(t, &repls, user)
	testdiff.PrepareReplacementsWorkspaceClient(t, &repls, workspaceClient)

	// Must be added PrepareReplacementsUser, otherwise conflicts with [USERNAME]
	testdiff.PrepareReplacementsUUID(t, &repls)

	// User replacements come last:
	repls.Repls = append(repls.Repls, config.Repls...)

	// Save replacements to temp test directory so that it can be read by diff.py
	replsJson, err := json.MarshalIndent(repls.Repls, "", "  ")
	require.NoError(t, err)
	testutil.WriteFile(t, filepath.Join(tmpDir, ReplsFile), string(replsJson))

	if coverDir != "" {
		// Creating individual coverage directory for each test, because writing to the same one
		// results in sporadic failures like this one (only if tests are running in parallel):
		// +error: coverage meta-data emit failed: writing ... rename .../tmp.covmeta.b3f... .../covmeta.b3f2c...: no such file or directory
		coverDir = filepath.Join(coverDir, strings.ReplaceAll(dir, string(os.PathSeparator), "--"))
		err := os.MkdirAll(coverDir, os.ModePerm)
		require.NoError(t, err)
		cmd.Env = append(cmd.Env, "GOCOVERDIR="+coverDir)
	}

	absDir, err := filepath.Abs(dir)
	require.NoError(t, err)
	cmd.Env = append(cmd.Env, "TESTDIR="+absDir)

	// Write combined output to a file
	out, err := os.Create(filepath.Join(tmpDir, "output.txt"))
	require.NoError(t, err)
	cmd.Stdout = out
	cmd.Stderr = out
	cmd.Dir = tmpDir
	err = cmd.Run()

	// Write the requests made to the server to a output file if the test is
	// configured to record requests.
	if config.RecordRequests {
		f, err := os.OpenFile(filepath.Join(tmpDir, "out.requests.txt"), os.O_CREATE|os.O_WRONLY, 0o644)
		require.NoError(t, err)

		for _, req := range server.Requests {
			reqJson, err := json.MarshalIndent(req, "", "  ")
			require.NoErrorf(t, err, "Failed to indent: %#v", req)

			reqJsonWithRepls := repls.Replace(string(reqJson))
			_, err = f.WriteString(reqJsonWithRepls + "\n")
			require.NoError(t, err)
		}

		err = f.Close()
		require.NoError(t, err)
	}

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
	unexpected := []string{}
	for _, relPath := range files {
		if _, ok := inputs[relPath]; ok {
			continue
		}
		if _, ok := outputs[relPath]; ok {
			continue
		}
		if _, ok := Ignored[relPath]; ok {
			continue
		}
		unexpected = append(unexpected, relPath)
		if strings.HasPrefix(relPath, "out") {
			// We have a new file starting with "out"
			// Show the contents & support overwrite mode for it:
			doComparison(t, repls, dir, tmpDir, relPath, &printedRepls)
		}
	}

	if len(unexpected) > 0 {
		t.Error("Test produced unexpected files:\n" + strings.Join(unexpected, "\n"))
	}
}

func doComparison(t *testing.T, repls testdiff.ReplacementsContext, dirRef, dirNew, relPath string, printedRepls *bool) {
	pathRef := filepath.Join(dirRef, relPath)
	pathNew := filepath.Join(dirNew, relPath)
	bufRef, okRef := tryReading(t, pathRef)
	bufNew, okNew := tryReading(t, pathNew)
	if !okRef && !okNew {
		t.Errorf("Both files are missing or have errors: %s\npathRef: %s\npathNew: %s", relPath, pathRef, pathNew)
		return
	}

	valueRef := testdiff.NormalizeNewlines(bufRef)
	valueNew := testdiff.NormalizeNewlines(bufNew)

	// Apply replacements to the new value only.
	// The reference value is stored after applying replacements.
	if !NoRepl {
		valueNew = repls.Replace(valueNew)
	}

	// The test did not produce an expected output file.
	if okRef && !okNew {
		t.Errorf("Missing output file: %s", relPath)
		if testdiff.OverwriteMode {
			t.Logf("Removing output file: %s", relPath)
			require.NoError(t, os.Remove(pathRef))
		}
		return
	}

	// The test produced an unexpected output file.
	if !okRef && okNew {
		t.Errorf("Unexpected output file: %s\npathRef: %s\npathNew: %s", relPath, pathRef, pathNew)
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

	if VerboseTest && !equal && printedRepls != nil && !*printedRepls {
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
		x, ok := tryReading(t, filepath.Join(dir, CleanupScript))
		if ok {
			cleanups = append(cleanups, x)
		}

		x, ok = tryReading(t, filepath.Join(dir, PrepareScript))
		if ok {
			prepares = append(prepares, x)
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

func BuildCLI(t *testing.T, buildDir, coverDir string) string {
	execPath := filepath.Join(buildDir, "databricks")
	if runtime.GOOS == "windows" {
		execPath += ".exe"
	}

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

	RunCommand(t, args, "..")
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

func tryReading(t *testing.T, path string) (string, bool) {
	info, err := os.Stat(path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			t.Errorf("%s: %s", path, err)
		}
		return "", false
	}

	if info.Size() > MaxFileSize {
		t.Errorf("%s: ignoring, too large: %d", path, info.Size())
		return "", false
	}

	data, err := os.ReadFile(path)
	if err != nil {
		// already checked ErrNotExist above
		t.Errorf("%s: %s", path, err)
		return "", false
	}

	if !utf8.Valid(data) {
		t.Errorf("%s: not valid utf-8", path)
		return "", false
	}

	return string(data), true
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

func getUVDefaultCacheDir(t *testing.T) string {
	// According to uv docs https://docs.astral.sh/uv/concepts/cache/#caching-in-continuous-integration
	// the default cache directory is
	// "A system-appropriate cache directory, e.g., $XDG_CACHE_HOME/uv or $HOME/.cache/uv on Unix and %LOCALAPPDATA%\uv\cache on Windows"
	cacheDir, err := os.UserCacheDir()
	require.NoError(t, err)
	if runtime.GOOS == "windows" {
		return cacheDir + "\\uv\\cache"
	} else {
		return cacheDir + "/uv"
	}
}

func RunCommand(t *testing.T, args []string, dir string) {
	start := time.Now()
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	elapsed := time.Since(start)
	t.Logf("%s took %s", args, elapsed)

	require.NoError(t, err, "%s failed: %s\n%s", args, err, out)
	if len(out) > 0 {
		t.Logf("%s output: %s", args, out)
	}
}

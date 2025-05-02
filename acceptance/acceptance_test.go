package acceptance_test

import (
	"bufio"
	"context"
	"encoding/base32"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
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

	"github.com/databricks/cli/acceptance/internal"
	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/testdiff"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/stretchr/testify/require"
)

var (
	KeepTmp     bool
	NoRepl      bool
	VerboseTest bool = os.Getenv("VERBOSE_TEST") != ""
	Tail        bool
	Forcerun    bool
	LogRequests bool
	LogConfig   bool
)

// In order to debug CLI running under acceptance test, search for TestInprocessMode and update
// the test name there, e.g. "bundle/variables/empty".
// Then install your breakpoints and click "debug test" near TestInprocessMode in VSCODE.
//
// To debug integration tests you can run the "deco env flip workspace" command to configure a workspace
// and then click on "debug test" near TestInprocessMode.

// If enabled, instead of compiling and running CLI externally, we'll start in-process server that accepts and runs
// CLI commands. The $CLI in test scripts is a helper that just forwards command-line arguments to this server (see bin/callserver.py).
// Also disables parallelism in tests.
var InprocessMode bool

func init() {
	flag.BoolVar(&InprocessMode, "inprocess", false, "Run CLI in the same process as test (for debugging)")
	flag.BoolVar(&KeepTmp, "keeptmp", false, "Do not delete TMP directory after run")
	flag.BoolVar(&NoRepl, "norepl", false, "Do not apply any replacements (for debugging)")
	flag.BoolVar(&Tail, "tail", false, "Log output of script in real time. Use with -v to see the logs: -tail -v")
	flag.BoolVar(&Forcerun, "forcerun", false, "Force running the specified tests, ignore all reasons to skip")
	flag.BoolVar(&LogRequests, "logrequests", false, "Log request and responses from testserver")
	flag.BoolVar(&LogConfig, "logconfig", false, "Log merged for each test case")
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
	testAccept(t, InprocessMode, "")
}

func TestInprocessMode(t *testing.T) {
	if InprocessMode && !Forcerun {
		t.Skip("Already tested by TestAccept")
	}
	require.Equal(t, 1, testAccept(t, true, "selftest/basic"))
	require.Equal(t, 1, testAccept(t, true, "selftest/server"))
}

func testAccept(t *testing.T, InprocessMode bool, singleTest string) int {
	// Load debug environment when debugging a single test run from an IDE.
	if singleTest != "" && InprocessMode {
		testutil.LoadDebugEnvIfRunFromIDE(t, "workspace")
	}

	repls := testdiff.ReplacementsContext{}
	cwd, err := os.Getwd()
	require.NoError(t, err)

	// Consistent behavior of locale-dependent tools, such as 'sort'
	t.Setenv("LC_ALL", "C")

	buildDir := filepath.Join(cwd, "build", fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH))

	// Download terraform and provider and create config; this also creates build directory.
	RunCommand(t, []string{"python3", filepath.Join(cwd, "install_terraform.py"), "--targetdir", buildDir}, ".")

	wheelPath := buildDatabricksBundlesWheel(t, buildDir)
	if wheelPath != "" {
		t.Setenv("DATABRICKS_BUNDLES_WHEEL", wheelPath)
		repls.SetPath(wheelPath, "[DATABRICKS_BUNDLES_WHEEL]")
	}

	coverDir := os.Getenv("CLI_GOCOVERDIR")

	if coverDir != "" {
		require.NoError(t, os.MkdirAll(coverDir, os.ModePerm))
		coverDir, err = filepath.Abs(coverDir)
		require.NoError(t, err)
		t.Logf("Writing coverage to %s", coverDir)
	}

	execPath := ""

	if InprocessMode {
		cmdServer := internal.StartCmdServer(t)
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
		internal.StartDefaultServer(t)
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

	repls.Set(os.Getenv("TEST_INSTANCE_POOL_ID"), "[TEST_INSTANCE_POOL_ID]")

	testdiff.PrepareReplacementsDevVersion(t, &repls)
	testdiff.PrepareReplacementSdkVersion(t, &repls)
	testdiff.PrepareReplacementsGoVersion(t, &repls)

	repls.SetPath(cwd, "[TESTROOT]")

	repls.Repls = append(repls.Repls, testdiff.Replacement{Old: regexp.MustCompile("dbapi[0-9a-f]+"), New: "[DATABRICKS_TOKEN]"})

	// Matches defaultSparkVersion in ../integration/bundle/helpers_test.go
	t.Setenv("DEFAULT_SPARK_VERSION", "13.3.x-snapshot-scala2.12")

	nodeTypeID := getNodeTypeID(cloudEnv)
	t.Setenv("NODE_TYPE_ID", nodeTypeID)
	repls.Set(nodeTypeID, "[NODE_TYPE_ID]")

	testDirs := getTests(t)
	require.NotEmpty(t, testDirs)

	if singleTest != "" {
		testDirs = slices.DeleteFunc(testDirs, func(n string) bool {
			return n != singleTest
		})
		require.NotEmpty(t, testDirs, "singleTest=%#v did not match any tests\n%#v", singleTest, testDirs)
	}

	skippedDirs := 0
	totalDirs := 0
	selectedDirs := 0

	for _, dir := range testDirs {
		totalDirs += 1

		t.Run(dir, func(t *testing.T) {
			selectedDirs += 1

			config, configPath := internal.LoadConfig(t, dir)
			skipReason := getSkipReason(&config, configPath)

			if skipReason != "" {
				skippedDirs += 1
				t.Skip(skipReason)
			}

			if !InprocessMode {
				t.Parallel()
			}

			expanded := internal.ExpandEnvMatrix(config.EnvMatrix)

			if testdiff.OverwriteMode && len(expanded) > 1 {
				// All variants of the test are producing the same output,
				// there is no need to run the concurrently when updating.
				expanded = expanded[0:1]
			}

			if len(expanded) == 1 {
				// env vars aren't part of the test case name, so log them for debugging
				t.Logf("Running test with env %v", expanded[0])
				runTest(t, dir, coverDir, repls.Clone(), config, configPath, expanded[0])
			} else {
				for _, envset := range expanded {
					envname := strings.Join(envset, "/")
					t.Run(envname, func(t *testing.T) {
						if !InprocessMode {
							t.Parallel()
						}
						runTest(t, dir, coverDir, repls.Clone(), config, configPath, envset)
					})
				}
			}
		})
	}

	t.Logf("Summary: %d/%d/%d run/selected/total, %d skipped", selectedDirs-skippedDirs, selectedDirs, totalDirs, skippedDirs)

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

// Return a reason to skip the test. Empty string means "don't skip".
func getSkipReason(config *internal.TestConfig, configPath string) string {
	if Forcerun {
		return ""
	}

	isEnabled, isPresent := config.GOOS[runtime.GOOS]
	if isPresent && !isEnabled {
		return fmt.Sprintf("Disabled via GOOS.%s setting in %s", runtime.GOOS, configPath)
	}

	cloudEnv := os.Getenv("CLOUD_ENV")
	isRunningOnCloud := cloudEnv != ""

	if isRunningOnCloud {
		cloudEnvBase := getCloudEnvBase(cloudEnv)
		isEnabled, isPresent := config.CloudEnvs[cloudEnvBase]
		if isPresent && !isEnabled {
			return fmt.Sprintf("Disabled via CloudEnvs.%s setting in %s (CLOUD_ENV=%s)", cloudEnvBase, configPath, cloudEnv)
		}

		if isTruePtr(config.CloudSlow) {
			if testing.Short() {
				return fmt.Sprintf("Disabled via CloudSlow setting in %s (CLOUD_ENV=%s, Short=%v)", configPath, cloudEnv, testing.Short())
			}
		}

		isCloudEnabled := isTruePtr(config.Cloud) || isTruePtr(config.CloudSlow)
		if !isCloudEnabled {
			return fmt.Sprintf("Disabled via Cloud/CloudSlow setting in %s (CLOUD_ENV=%s, Cloud=%v, CloudSlow=%v)",
				configPath,
				cloudEnv,
				isTruePtr(config.Cloud),
				isTruePtr(config.CloudSlow),
			)
		}

		if isTruePtr(config.RequiresUnityCatalog) && os.Getenv("TEST_METASTORE_ID") == "" {
			return fmt.Sprintf("Disabled via RequiresUnityCatalog setting in %s (TEST_METASTORE_ID is empty)", configPath)
		}

		if isTruePtr(config.RequiresWarehouse) && os.Getenv("TEST_DEFAULT_WAREHOUSE_ID") == "" {
			return fmt.Sprintf("Disabled via RequiresWarehouse setting in %s (TEST_DEFAULT_WAREHOUSE_ID is empty)", configPath)
		}

		if isTruePtr(config.RequiresCluster) && os.Getenv("TEST_DEFAULT_CLUSTER_ID") == "" {
			return fmt.Sprintf("Disabled via RequiresCluster setting in %s (TEST_DEFAULT_CLUSTER_ID is empty)", configPath)
		}

	} else {
		// Local run
		if !isTruePtr(config.Local) {
			return fmt.Sprintf("Disabled via Local setting in %s (CLOUD_ENV=%s)", configPath, cloudEnv)
		}
	}

	return ""
}

func runTest(t *testing.T, dir, coverDir string, repls testdiff.ReplacementsContext, config internal.TestConfig, configPath string, customEnv []string) {
	if LogConfig {
		configBytes, err := json.MarshalIndent(config, "", "  ")
		require.NoError(t, err)
		t.Log(string(configBytes))
	}

	tailOutput := Tail
	cloudEnv := os.Getenv("CLOUD_ENV")
	isRunningOnCloud := cloudEnv != ""

	if isRunningOnCloud && isTruePtr(config.CloudSlow) && testing.Verbose() {
		// Combination of CloudSlow and -v auto-enables -tail
		tailOutput = true
	}

	id := uuid.New()
	uniqueName := strings.ToLower(strings.Trim(base32.StdEncoding.EncodeToString(id[:]), "="))
	repls.Set(uniqueName, "[UNIQUE_NAME]")

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

	repls.SetPathWithParents(tmpDir, "[TEST_TMP_DIR]")

	scriptContents := readMergedScriptContents(t, dir)
	testutil.WriteFile(t, filepath.Join(tmpDir, EntryPointScript), scriptContents)

	inputs := make(map[string]bool, 2)
	outputs := make(map[string]bool, 2)
	err = CopyDir(dir, tmpDir, inputs, outputs)
	require.NoError(t, err)

	timeout := config.Timeout

	if runtime.GOOS == "windows" {
		if isRunningOnCloud {
			timeout = max(timeout, config.TimeoutWindows, config.TimeoutCloud)
		} else {
			timeout = max(timeout, config.TimeoutWindows)
		}
	} else if isRunningOnCloud {
		timeout = max(timeout, config.TimeoutCloud)
	}

	ctx, cancelFunc := context.WithTimeout(context.Background(), timeout)
	defer cancelFunc()
	args := []string{"bash", "-euo", "pipefail", EntryPointScript}
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "UNIQUE_NAME="+uniqueName)
	cmd.Env = append(cmd.Env, "TEST_TMP_DIR="+tmpDir)

	workspaceClient := internal.PrepareServerAndClient(t, config, LogRequests, tmpDir)

	// Configure resolved credentials in the environment.
	cmd.Env = append(cmd.Env, "DATABRICKS_HOST="+workspaceClient.Config.Host)
	if workspaceClient.Config.Token != "" {
		cmd.Env = append(cmd.Env, "DATABRICKS_TOKEN="+workspaceClient.Config.Token)
	}

	var user iam.User
	if isRunningOnCloud {
		pUser, err := workspaceClient.CurrentUser.Me(context.Background())
		require.NoError(t, err, "Failed to get current user")
		user = *pUser
	} else {
		// For the purposes of replacements, use testUser for local runs.
		// Note, users might have overriden /api/2.0/preview/scim/v2/Me but that should not affect the replacement:
		user = internal.TestUser
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

	for _, key := range utils.SortedKeys(config.Env) {
		if hasKey(customEnv, key) {
			// We want EnvMatrix to take precedence.
			// Skip rather than relying on cmd.Env order, because this might interfere with replacements and substitutions.
			continue
		}
		cmd.Env = addEnvVar(t, cmd.Env, &repls, key, config.Env[key], config.EnvRepl)
	}

	for _, keyvalue := range customEnv {
		items := strings.SplitN(keyvalue, "=", 2)
		require.Len(t, items, 2)
		cmd.Env = addEnvVar(t, cmd.Env, &repls, items[0], items[1], config.EnvRepl)
	}

	absDir, err := filepath.Abs(dir)
	require.NoError(t, err)
	cmd.Env = append(cmd.Env, "TESTDIR="+absDir)
	cmd.Env = append(cmd.Env, "CLOUD_ENV="+cloudEnv)
	cmd.Env = append(cmd.Env, "CURRENT_USER_NAME="+user.UserName)
	cmd.Dir = tmpDir

	outputPath := filepath.Join(tmpDir, "output.txt")
	out, err := os.Create(outputPath)
	require.NoError(t, err)
	defer out.Close()

	err = runWithLog(t, cmd, out, tailOutput)

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
	var unexpected []string
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
		if config.CompiledIgnoreObject.MatchesPath(relPath) {
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

func hasKey(env []string, key string) bool {
	for _, keyvalue := range env {
		items := strings.SplitN(keyvalue, "=", 2)
		if len(items) == 2 && items[0] == key {
			return true
		}
	}
	return false
}

func addEnvVar(t *testing.T, env []string, repls *testdiff.ReplacementsContext, key, value string, envRepl map[string]bool) []string {
	newValue, newValueWithPlaceholders := internal.SubstituteEnv(value, env)
	if value != newValue {
		t.Logf("Substituted %s %#v -> %#v (%#v)", key, value, newValue, newValueWithPlaceholders)
	}

	shouldRepl, ok := envRepl[key]
	if !ok {
		shouldRepl = true
	}

	if shouldRepl {
		repls.Set(newValue, "["+key+"]")
		// newValue won't match because parts of it were already replaced; we adding it anyway just in case but we need newValueWithPlaceholders:
		repls.Set(newValueWithPlaceholders, "["+key+"]")
	}

	return append(env, key+"="+newValue)
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
		if shouldShowDiff(pathNew, valueNew) {
			testdiff.AssertEqualTexts(t, pathRef, pathNew, valueRef, valueNew)
		}
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

func shouldShowDiff(pathNew, valueNew string) bool {
	if strings.Contains(pathNew, "site-packages") {
		return false
	}
	if strings.Contains(pathNew, ".venv") {
		return false
	}
	if len(valueNew) > 10_000 {
		return false
	}
	// if file itself starts with "out" then it's likely to be intended to be recorded
	return strings.HasPrefix(filepath.Base(pathNew), "out")
}

// Returns combined script.prepare (root) + script.prepare (parent) + ... + script + ... + script.cleanup (parent) + ...
// Note, cleanups are not executed if main script fails; that's not a huge issue, since it runs it temp dir.
func readMergedScriptContents(t *testing.T, dir string) string {
	scriptContents := testutil.ReadFile(t, filepath.Join(dir, EntryPointScript))

	// Wrap script contents in a subshell such that changing the working
	// directory only affects the main script and not cleanup.
	scriptContents = "(\n" + scriptContents + ")\n"

	var prepares []string
	var cleanups []string

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

	info, err := in.Stat()
	if err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode())
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

	// Do not check output.txt for UTF8 validity, because 'deploy --debug' logs binary request/responses
	doUTF8Check := filepath.Base(path) != "output.txt"

	if doUTF8Check && !utf8.Valid(data) {
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

type LoggedRequest struct {
	Headers http.Header `json:"headers,omitempty"`
	Method  string      `json:"method"`
	Path    string      `json:"path"`
	Body    any         `json:"body,omitempty"`
	RawBody string      `json:"raw_body,omitempty"`
}

func isTruePtr(value *bool) bool {
	return value != nil && *value
}

func runWithLog(t *testing.T, cmd *exec.Cmd, out *os.File, tail bool) error {
	r, w := io.Pipe()
	cmd.Stdout = w
	cmd.Stderr = w
	processErrCh := make(chan error, 1)

	cmd.Cancel = func() error {
		processErrCh <- errors.New("Test script killed due to a timeout")
		_ = cmd.Process.Kill()
		_ = w.Close()
		return nil
	}

	start := time.Now()
	err := cmd.Start()
	if err != nil {
		return err
	}

	go func() {
		processErrCh <- cmd.Wait()
		_ = w.Close()
	}()

	reader := bufio.NewReader(r)
	for {
		line, err := reader.ReadString('\n')
		if tail {
			msg := strings.TrimRight(line, "\n")
			if len(msg) > 0 {
				d := time.Since(start)
				t.Logf("%2d.%03d %s", d/time.Second, (d%time.Second)/time.Millisecond, msg)
			}
		}
		if len(line) > 0 {
			_, err = out.WriteString(line)
			require.NoError(t, err)
		}
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
	}

	return <-processErrCh
}

func getCloudEnvBase(cloudEnv string) string {
	switch cloudEnv {
	// no idea why, but
	// aws-prod-ucws sets CLOUD_ENV to "ucws"
	// gcp-prod-ucws sets CLOUD_ENV to "gcp-ucws"
	// azure-prod-ucws sets CLOUD_ENV to "azure"
	case "aws", "ucws":
		return "aws"
	case "azure":
		return "azure"
	case "gcp", "gcp-ucws":
		return "gcp"
	case "":
		return ""
	default:
		return "unknown-cloudEnv-" + cloudEnv
	}
}

func getNodeTypeID(cloudEnv string) string {
	base := getCloudEnvBase(cloudEnv)
	switch base {
	case "aws":
		return "i3.xlarge"
	case "azure":
		return "Standard_DS4_v2"
	case "gcp":
		return "n1-standard-4"
	case "":
		return "local-fake-node"
	default:
		return "nodetype-" + cloudEnv
	}
}

// buildDatabricksBundlesWheel builds the databricks-bundles wheel and returns the path to the wheel.
func buildDatabricksBundlesWheel(t *testing.T, buildDir string) string {
	// Clean up directory, remove all but the latest wheel
	// Doing this avoids ambiguity if the build command below does not touch any whl files,
	// because it considers it already good. However, we would not know which one it considered good,
	// so we prepare here by keeping only one.
	_ = prepareWheelBuildDirectory(t, buildDir)

	RunCommand(t, []string{"uv", "build", "--no-cache", "-q", "--wheel", "--out-dir", buildDir}, "../experimental/python")

	latestWheel := prepareWheelBuildDirectory(t, buildDir)
	if latestWheel == "" {
		// Many tests don't need the wheel, so continue there rather than hard fail
		t.Errorf("databricks-bundles wheel not found in %s", buildDir)
	}

	return latestWheel
}

// Find all possible whl files in 'dir' and clean up all but the one with most recent mtime
// Return that full path to the wheel with most recent mtime (that was not cleaned up)
func prepareWheelBuildDirectory(t *testing.T, dir string) string {
	var wheels []string

	files, err := os.ReadDir(dir)
	require.NoError(t, err)

	var latestWheel string
	var latestTime time.Time

	// First pass: find the latest wheel
	for _, file := range files {
		name := file.Name()
		if strings.HasPrefix(name, "databricks_bundles-") && strings.HasSuffix(name, ".whl") {
			info, err := file.Info()
			require.NoError(t, err)
			name = filepath.Join(dir, name)
			wheels = append(wheels, name)
			if info.ModTime().After(latestTime) {
				latestWheel = name
				latestTime = info.ModTime()
			}
		}
	}

	// Second pass: delete all wheels except the latest
	for _, wheel := range wheels {
		if wheel != latestWheel {
			err := os.Remove(wheel)
			if err == nil {
				t.Logf("Cleaning up %s", wheel)
			} else {
				t.Errorf("Cleaning up %s failed: %s", wheel, err)
			}
		}
	}

	return latestWheel
}

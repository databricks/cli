package acceptance_test

import (
	"archive/zip"
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
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/databricks/cli/acceptance/internal"
	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/testdiff"
	"github.com/databricks/cli/libs/utils"
	"github.com/stretchr/testify/require"
)

var (
	KeepTmp         bool
	NoRepl          bool
	VerboseTest     bool = os.Getenv("VERBOSE_TEST") != ""
	Tail            bool
	Forcerun        bool
	LogRequests     bool
	LogConfig       bool
	SkipLocal       bool
	UseVersion      string
	WorkspaceTmpDir bool
	TerraformDir    string
	OnlyOutTestToml bool
)

// In order to debug CLI running under acceptance test, search for TestInprocessMode and update
// the test name there, e.g. "bundle/variables/empty".
// Then install your breakpoints and click "debug test" near TestInprocessMode in VSCODE.
//
// If enabled, instead of compiling and running CLI externally, we'll start in-process server that accepts and runs
// CLI commands. The $CLI in test scripts is a helper that just forwards command-line arguments to this server (see bin/callserver.py).
// Also disables parallelism in tests.
var InprocessMode bool

// lines with this prefix are not recorded in output.txt but logged instead
const TestLogPrefix = "TESTLOG: "

func init() {
	flag.BoolVar(&InprocessMode, "inprocess", false, "Run CLI in the same process as test (for debugging)")
	flag.BoolVar(&KeepTmp, "keeptmp", false, "Do not delete TMP directory after run")
	flag.BoolVar(&NoRepl, "norepl", false, "Do not apply any replacements (for debugging)")
	flag.BoolVar(&Tail, "tail", false, "Log output of script in real time. Use with -v to see the logs: -tail -v")
	flag.BoolVar(&Forcerun, "forcerun", false, "Force running the specified tests, ignore all reasons to skip")
	flag.BoolVar(&LogRequests, "logrequests", false, "Log request and responses from testserver")
	flag.BoolVar(&LogConfig, "logconfig", false, "Log merged for each test case")
	flag.BoolVar(&SkipLocal, "skiplocal", false, "Skip tests that are enabled to run on Local")
	flag.StringVar(&UseVersion, "useversion", "", "Download previously released version of CLI and use it to run the tests")

	// DABs in the workspace runs on the workspace file system. This flags does the same for acceptance tests
	// to simulate an identical environment.
	flag.BoolVar(&WorkspaceTmpDir, "workspace-tmp-dir", false, "Run tests on the workspace file system (For DBR testing).")

	// Symlinks from workspace file system to local file mount are not supported on DBR. Terraform implicitly
	// creates these symlinks when a file_mirror is used for a provider (in .terraformrc). This flag
	// allows us to download the provider to the workspace file system on DBR enabling DBR integration testing.
	flag.StringVar(&TerraformDir, "terraform-dir", "", "Directory to download the terraform provider to")
	flag.BoolVar(&OnlyOutTestToml, "only-out-test-toml", false, "Only regenerate out.test.toml files without running tests")
}

const (
	EntryPointScript = "script"
	CleanupScript    = "script.cleanup"
	PrepareScript    = "script.prepare"
	MaxFileSize      = 1_000_000
	// Filename to save replacements to (used by diff.py)
	ReplsFile = "repls.json"
	// Filename for materialized config (used as golden file)
	MaterializedConfigFile = "out.test.toml"

	// ENVFILTER allows filtering subtests matching certain env var.
	// e.g. ENVFILTER=SERVERLESS=yes will run all tests that run SERVERLESS to "yes"
	// The tests the don't set SERVERLESS variable or set to empty string will also be run.
	EnvFilterVar = "ENVFILTER"

	// File where scripts can output custom replacements
	// export $job_id=100200300
	// $ echo "$job_id:MY_JOB" >> ACC_REPLS  # This will replace 100200300 with [MY_JOB] in the output
	// TODO: this should be merged with repls.json functionality, currently these replacements are not parsed by diff.py
	userReplacementsFilename = "ACC_REPLS"
)

// On CI, we want to increase timeout, to account for slower environment
const CITimeoutMultiplier = 2

var ApplyCITimeoutMultipler = os.Getenv("GITHUB_WORKFLOW") != ""

var exeSuffix = func() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}()

var Scripts = map[string]bool{
	EntryPointScript: true,
	CleanupScript:    true,
	PrepareScript:    true,
}

var Ignored = map[string]bool{
	ReplsFile:                true,
	userReplacementsFilename: true,
}

func TestAccept(t *testing.T) {
	testAccept(t, InprocessMode, "")
}

func TestInprocessMode(t *testing.T) {
	if InprocessMode && !Forcerun {
		t.Skip("Already tested by TestAccept")
	}
	if os.Getenv("CLOUD_ENV") != "" {
		t.Skip("No need to run this as integration test.")
	}

	// Uncomment to load  ~/.databricks/debug-env.json to debug integration tests
	// testutil.LoadDebugEnvIfRunFromIDE(t, "workspace")
	// Run the "deco env flip workspace" command to configure a workspace.

	require.Equal(t, 1, testAccept(t, true, "selftest/basic"))
	require.Equal(t, 1, testAccept(t, true, "selftest/server"))
}

// Configure replacements for environment variables we read from test environments.
func setReplsForTestEnvVars(t *testing.T, repls *testdiff.ReplacementsContext) {
	envVars := []string{
		"TEST_DEFAULT_WAREHOUSE_ID",
		"TEST_INSTANCE_POOL_ID",
	}
	for _, envVar := range envVars {
		if value := os.Getenv(envVar); value != "" {
			repls.Set(value, "["+envVar+"]")
		}
	}
}

func testAccept(t *testing.T, inprocessMode bool, singleTest string) int {
	repls := testdiff.ReplacementsContext{}
	cwd, err := os.Getwd()
	require.NoError(t, err)

	// Consistent behavior of locale-dependent tools, such as 'sort'
	t.Setenv("LC_ALL", "C")

	buildDir := getBuildDir(t, cwd, runtime.GOOS, runtime.GOARCH)

	terraformDir := TerraformDir
	if terraformDir == "" {
		terraformDir = buildDir
	}

	// Download terraform and provider and create config.
	RunCommand(t, []string{"python3", filepath.Join(cwd, "install_terraform.py"), "--targetdir", terraformDir}, ".", []string{})

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

	if inprocessMode {
		cmdServer := internal.StartCmdServer(t)
		t.Setenv("CMD_SERVER_URL", cmdServer.URL)
		execPath = filepath.Join(cwd, "bin", "callserver.py")
	} else {
		if UseVersion != "" {
			execPath = DownloadCLI(t, buildDir, UseVersion)
		} else {
			execPath = BuildCLI(t, buildDir, coverDir, runtime.GOOS, runtime.GOARCH)
		}
	}

	BuildYamlfmt(t)

	t.Setenv("CLI", execPath)
	repls.SetPath(execPath, "[CLI]")

	pipelinesPath := filepath.Join(buildDir, "pipelines") + exeSuffix
	err = copyFile(execPath, pipelinesPath)
	require.NoError(t, err)
	t.Setenv("PIPELINES", pipelinesPath)
	repls.SetPath(pipelinesPath, "[PIPELINES]")

	paths := []string{
		// Make helper scripts available
		filepath.Join(cwd, "bin"),

		// Make <ROOT>/tools/ available (e.g. yamlfmt)
		filepath.Join(cwd, "..", "tools"),

		os.Getenv("PATH"),
	}
	t.Setenv("PATH", strings.Join(paths, string(os.PathListSeparator)))

	// Make use of uv cache; since we set HomeEnvVar to temporary directory, it is not picked up automatically
	uvCache := getUVDefaultCacheDir(t)
	t.Setenv("UV_CACHE_DIR", uvCache)

	// UV_CACHE_DIR only applies to packages but not Python installations.
	// UV_PYTHON_INSTALL_DIR ensures we cache Python downloads as well
	uvInstall := filepath.Join(uvCache, "python_installs")
	t.Setenv("UV_PYTHON_INSTALL_DIR", uvInstall)

	cloudEnv := os.Getenv("CLOUD_ENV")

	if cloudEnv == "" {
		internal.StartDefaultServer(t, LogRequests)
		if os.Getenv("TEST_DEFAULT_WAREHOUSE_ID") == "" {
			t.Setenv("TEST_DEFAULT_WAREHOUSE_ID", "8ec9edc1-db0c-40df-af8d-7580020fe61e")
		}
	}

	setReplsForTestEnvVars(t, &repls)

	if cloudEnv != "" && UseVersion == "" {
		// Create linux release artifacts, to be used by the cloud-only ssh tunnel tests
		releasesDir := CreateReleaseArtifacts(t, cwd, coverDir, "linux")
		t.Setenv("CLI_RELEASES_DIR", releasesDir)
	}

	terraformrcPath := filepath.Join(terraformDir, ".terraformrc")
	t.Setenv("TF_CLI_CONFIG_FILE", terraformrcPath)
	t.Setenv("DATABRICKS_TF_CLI_CONFIG_FILE", terraformrcPath)
	repls.SetPath(terraformrcPath, "[DATABRICKS_TF_CLI_CONFIG_FILE]")

	terraformExecPath := filepath.Join(terraformDir, "terraform") + exeSuffix
	t.Setenv("DATABRICKS_TF_EXEC_PATH", terraformExecPath)
	t.Setenv("TERRAFORM", terraformExecPath)
	repls.SetPath(terraformExecPath, "[TERRAFORM]")

	// do it last so that full paths match first:
	repls.SetPath(buildDir, "[BUILD_DIR]")

	testdiff.PrepareReplacementsDevVersion(t, &repls)
	testdiff.PrepareReplacementSdkVersion(t, &repls)
	testdiff.PrepareReplacementsGoVersion(t, &repls)

	t.Setenv("TESTROOT", cwd)
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

	envFilters := getEnvFilters(t)

	for _, dir := range testDirs {
		totalDirs += 1

		t.Run(dir, func(t *testing.T) {
			selectedDirs += 1

			config, configPath := internal.LoadConfig(t, dir)
			skipReason := getSkipReason(&config, configPath)

			if testdiff.OverwriteMode || OnlyOutTestToml {
				// Generate materialized config for this test
				// We do this before skipping the test, so the configs are generated for all tests.
				materializedConfig, err := internal.GenerateMaterializedConfig(config)
				require.NoError(t, err)
				testutil.WriteFile(t, filepath.Join(dir, internal.MaterializedConfigFile), materializedConfig)
			}

			// If only regenerating out.test.toml, skip the actual test execution
			if OnlyOutTestToml {
				t.Skip("Skipping test execution (only regenerating out.test.toml)")
			}

			if skipReason != "" {
				skippedDirs += 1
				t.Skip(skipReason)
			}

			if !inprocessMode {
				t.Parallel()
			}

			expanded := internal.ExpandEnvMatrix(config.EnvMatrix, config.EnvMatrixExclude)

			if len(expanded) == 1 {
				// env vars aren't part of the test case name, so log them for debugging
				if len(expanded[0]) > 0 {
					t.Logf("Running test with env %v", expanded[0])
				}
				runTest(t, dir, 0, coverDir, repls.Clone(), config, expanded[0], envFilters)
			} else {
				for ind, envset := range expanded {
					envname := strings.Join(envset, "/")
					t.Run(envname, func(t *testing.T) {
						if !inprocessMode {
							t.Parallel()
						}
						runTest(t, dir, ind, coverDir, repls.Clone(), config, envset, envFilters)
					})
				}
			}
		})
	}

	t.Logf("Summary (dirs): %d/%d/%d run/selected/total, %d skipped", selectedDirs-skippedDirs, selectedDirs, totalDirs, skippedDirs)

	return selectedDirs - skippedDirs
}

func getEnvFilters(t *testing.T) []string {
	envFilterValue := os.Getenv(EnvFilterVar)
	if envFilterValue == "" {
		return nil
	}

	filters := strings.Split(envFilterValue, ",")
	outFilters := make([]string, len(filters))

	for _, filter := range filters {
		items := strings.Split(filter, "=")
		if len(items) != 2 || len(items[0]) == 0 {
			t.Fatalf("Invalid filter %q in %s=%q", filter, EnvFilterVar, envFilterValue)
		}
		key := items[0]
		// Clear it just to be sure, since it's going to be part of os.Environ() and we're going to add different value based on settings.
		os.Unsetenv(key)
		outFilters = append(outFilters, key+"="+items[1])
	}

	return outFilters
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
	// Apply default first, so that it's visible in out.test.toml
	if isTruePtr(config.CloudSlow) {
		config.Cloud = config.CloudSlow
	}

	if SkipLocal && isTruePtr(config.Local) {
		return "Disabled via SkipLocal setting in " + configPath
	}

	if Forcerun {
		return ""
	}

	if isTruePtr(config.SkipOnDbr) && WorkspaceTmpDir {
		return "Disabled via SkipOnDbr setting in " + configPath
	}

	if isTruePtr(config.Slow) && testing.Short() {
		return "Disabled via Slow setting in " + configPath
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

func runTest(t *testing.T,
	dir string,
	variant int,
	coverDir string,
	repls testdiff.ReplacementsContext,
	config internal.TestConfig,
	customEnv []string,
	envFilters []string,
) {
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
	} else if WorkspaceTmpDir {
		// If the test is being run on DBR, auth is already configured
		// by the dbr_runner notebook by reading a token from the notebook context and
		// setting DATABRICKS_TOKEN and DATABRICKS_HOST environment variables.
		_, _, tmpDir = workspaceTmpDir(t.Context(), t)

		// Run DBR tests on the workspace file system to mimic usage from
		// DABs in the workspace.
		t.Logf("Running DBR tests on %s", tmpDir)
	} else {
		tmpDir = t.TempDir()
	}

	repls.SetPathWithParents(tmpDir, "[TEST_TMP_DIR]")

	scriptContents := readMergedScriptContents(t, dir)
	testutil.WriteFile(t, filepath.Join(tmpDir, EntryPointScript), scriptContents)

	// Generate materialized config for this test
	materializedConfig, err := internal.GenerateMaterializedConfig(config)
	require.NoError(t, err)
	testutil.WriteFile(t, filepath.Join(tmpDir, internal.MaterializedConfigFile), materializedConfig)

	inputs := make(map[string]bool, 2)
	outputs := make(map[string]bool, 2)
	err = CopyDir(dir, tmpDir, inputs, outputs)
	require.NoError(t, err)

	// Add materialized config to outputs for comparison
	outputs[internal.MaterializedConfigFile] = true

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

	if ApplyCITimeoutMultipler {
		timeout *= CITimeoutMultiplier
	}

	ctx, cancelFunc := context.WithTimeout(context.Background(), timeout)
	defer cancelFunc()
	args := []string{"bash", "-euo", "pipefail", EntryPointScript}
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)

	cfg, user := internal.PrepareServerAndClient(t, config, LogRequests, tmpDir)
	testdiff.PrepareReplacementsUser(t, &repls, user)
	testdiff.PrepareReplacementsWorkspaceConfig(t, &repls, cfg)

	cmd.Env = auth.ProcessEnv(cfg)

	rateLimit := os.Getenv("DATABRICKS_RATE_LIMIT")
	if rateLimit == "" {
		if isRunningOnCloud {
			rateLimit = "100"
		} else {
			rateLimit = "1000000000"
		}
	}
	cmd.Env = append(cmd.Env, "DATABRICKS_RATE_LIMIT="+rateLimit)
	cmd.Env = append(cmd.Env, "UNIQUE_NAME="+uniqueName)
	cmd.Env = append(cmd.Env, "TEST_TMP_DIR="+tmpDir)

	// populate CLOUD_ENV_BASE
	envBase := getCloudEnvBase(cloudEnv)
	cmd.Env = append(cmd.Env, "CLOUD_ENV_BASE="+envBase)

	// Must be added PrepareReplacementsUser, otherwise conflicts with [USERNAME]
	testdiff.PrepareReplacementsUUID(t, &repls)

	// User replacements:
	repls.Repls = append(repls.Repls, config.Repls...)

	// Save replacements to temp test directory so that it can be read by diff.py
	replsJson, err := json.MarshalIndent(repls.Repls, "", "  ")
	require.NoError(t, err)
	testutil.WriteFile(t, filepath.Join(tmpDir, ReplsFile), string(replsJson))

	if coverDir != "" {
		// Creating individual coverage directory for each test, because writing to the same one
		// results in sporadic failures like this one (only if tests are running in parallel):
		// +error: coverage meta-data emit failed: writing ... rename .../tmp.covmeta.b3f... .../covmeta.b3f2c...: no such file or directory
		// Note: should not use dir, because single dir can generate multiple tests via EnvMatrix
		coverDir = filepath.Join(coverDir, strings.ReplaceAll(dir, string(os.PathSeparator), "--"))
		if variant != 0 {
			coverDir += strconv.Itoa(variant)
		}
		err := os.MkdirAll(coverDir, os.ModePerm)
		require.NoError(t, err)
		cmd.Env = append(cmd.Env, "GOCOVERDIR="+coverDir)
	}

	// Set unique cache folder for this test to avoid race conditions between parallel tests
	// Use test temp directory to avoid polluting user's cache
	uniqueCacheDir := filepath.Join(t.TempDir(), ".cache")
	cmd.Env = append(cmd.Env, "DATABRICKS_CACHE_DIR="+uniqueCacheDir)

	for _, key := range utils.SortedKeys(config.Env) {
		if hasKey(customEnv, key) {
			// We want EnvMatrix to take precedence.
			// Skip rather than relying on cmd.Env order, because this might interfere with replacements and substitutions.
			continue
		}
		cmd.Env = addEnvVar(t, cmd.Env, &repls, key, config.Env[key], config.EnvRepl, false)
	}

	for _, keyvalue := range customEnv {
		items := strings.SplitN(keyvalue, "=", 2)
		require.Len(t, items, 2)
		key := items[0]
		value := items[1]
		// Only add replacement by default if value is part of EnvMatrix with more than 1 option and length is 4 or more chars
		// (to avoid matching "yes" and "no" values from template input parameters)
		cmd.Env = addEnvVar(t, cmd.Env, &repls, key, value, config.EnvRepl, len(config.EnvMatrix[key]) > 1 && len(value) >= 4)
	}

	for filterInd, filterEnv := range envFilters {
		filterEnvKey := strings.Split(filterEnv, "=")[0]
		for ind := range cmd.Env {
			// Search backwards, because the latest settings is what is actually applicable.
			envPair := cmd.Env[len(cmd.Env)-1-ind]
			if strings.Split(envPair, "=")[0] == filterEnvKey {
				if envPair == filterEnv {
					break
				} else {
					t.Skipf("Skipping because test environment %s does not match ENVFILTER#%d: %s", envPair, filterInd, filterEnv)
				}
			}
		}
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

	skipReason, err := runWithLog(t, cmd, out, tailOutput)

	if skipReason != "" {
		t.Skip("Skipping based on output: " + skipReason)
	}

	// Include exit code in output (if non-zero)
	formatOutput(out, err)
	require.NoError(t, out.Close())

	loadUserReplacements(t, &repls, tmpDir)

	printedRepls := false

	pathFilter := preparePathFilter(config, customEnv)

	// Compare expected outputs
	for relPath := range outputs {
		if shouldSkip(pathFilter, relPath) {
			continue
		}

		skipRepls := false
		if relPath == internal.MaterializedConfigFile {
			skipRepls = true
		}
		doComparison(t, repls, dir, tmpDir, relPath, &printedRepls, skipRepls)
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
		if strings.HasPrefix(filepath.Base(relPath), "LOG") {
			prefix := relPath + ": "
			messages := testutil.ReadFile(t, filepath.Join(tmpDir, relPath))
			messages = strings.TrimRight(messages, "\r\n \t")
			messages = prefix + strings.ReplaceAll(messages, "\n", "\n"+prefix)
			if strings.Contains(messages, "\n") {
				messages = "\n" + messages
			}
			t.Log(messages)
			continue
		}
		unexpected = append(unexpected, relPath)
		if strings.HasPrefix(relPath, "out") {
			// We have a new file starting with "out"
			// Show the contents & support overwrite mode for it:
			doComparison(t, repls, dir, tmpDir, relPath, &printedRepls, false)
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

func addEnvVar(t *testing.T, env []string, repls *testdiff.ReplacementsContext, key, value string, envRepl map[string]bool, defaultRepl bool) []string {
	newValue, newValueWithPlaceholders := internal.SubstituteEnv(value, env)
	if value != newValue {
		t.Logf("Substituted %s %#v -> %#v (%#v)", key, value, newValue, newValueWithPlaceholders)
	}

	shouldRepl, ok := envRepl[key]
	if !ok {
		shouldRepl = defaultRepl
	}

	if shouldRepl {
		repls.Set(newValue, "["+key+"]")
		// newValue won't match because parts of it were already replaced; we adding it anyway just in case but we need newValueWithPlaceholders:
		repls.Set(newValueWithPlaceholders, "["+key+"]")
	}

	return append(env, key+"="+newValue)
}

func doComparison(t *testing.T, repls testdiff.ReplacementsContext, dirRef, dirNew, relPath string, printedRepls *bool, skipRepls bool) {
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
	if !NoRepl && !skipRepls {
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

func getBuildDirRoot(cwd string) string {
	return filepath.Join(cwd, "build")
}

func getBuildDir(t *testing.T, cwd, osName, arch string) string {
	buildDir := filepath.Join(getBuildDirRoot(cwd), fmt.Sprintf("%s_%s", osName, arch))
	err := os.MkdirAll(buildDir, os.ModePerm)
	require.NoError(t, err)
	return buildDir
}

func BuildCLI(t *testing.T, buildDir, coverDir, osName, arch string) string {
	execPath := filepath.Join(buildDir, "databricks")
	if osName == "windows" {
		execPath += ".exe"
	}

	args := []string{
		"go", "build", "-o", execPath,
	}

	if coverDir != "" {
		args = append(args, "-cover")
	}

	if osName == "windows" {
		// Get this error on my local Windows:
		// error obtaining VCS status: exit status 128
		// Use -buildvcs=false to disable VCS stamping.
		args = append(args, "-buildvcs=false")
	}

	RunCommand(t, args, "..", []string{"GOOS=" + osName, "GOARCH=" + arch})
	return execPath
}

// CreateReleaseArtifacts builds release artifacts for the given OS using amd64 and arm64 architectures,
// archives them into zip files, and returns the directory containing the release artifacts.
func CreateReleaseArtifacts(t *testing.T, cwd, coverDir, osName string) string {
	releasesDir := filepath.Join(getBuildDirRoot(cwd), "releases")
	require.NoError(t, os.MkdirAll(releasesDir, os.ModePerm))
	for _, arch := range []string{"amd64", "arm64"} {
		CreateReleaseArtifact(t, cwd, releasesDir, coverDir, osName, arch)
	}
	return releasesDir
}

func CreateReleaseArtifact(t *testing.T, cwd, releasesDir, coverDir, osName, arch string) {
	buildDir := getBuildDir(t, cwd, osName, arch)
	execPath := BuildCLI(t, buildDir, coverDir, osName, arch)
	execInfo, err := os.Stat(execPath)
	require.NoError(t, err)

	zipName := fmt.Sprintf("databricks_cli_%s_%s.zip", osName, arch)
	zipPath := filepath.Join(releasesDir, zipName)

	zipFile, err := os.Create(zipPath)
	require.NoError(t, err)
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	header, err := zip.FileInfoHeader(execInfo)
	require.NoError(t, err)
	header.Name = "databricks"
	header.Method = zip.Deflate

	writer, err := zipWriter.CreateHeader(header)
	require.NoError(t, err)

	binaryFile, err := os.Open(execPath)
	require.NoError(t, err)
	defer binaryFile.Close()

	_, err = io.Copy(writer, binaryFile)
	require.NoError(t, err)

	t.Logf("Created %s %s release: %s", osName, arch, zipPath)
}

// DownloadCLI downloads a released CLI binary archive for the given version,
// extracts the executable, and returns its path.
func DownloadCLI(t *testing.T, buildDir, version string) string {
	// Prepare target directory for this version
	versionDir := filepath.Join(buildDir, version)
	require.NoError(t, os.MkdirAll(versionDir, 0o755))

	execName := "databricks"
	if runtime.GOOS == "windows" {
		execName += ".exe"
	}
	execPath := filepath.Join(versionDir, execName)

	// If already downloaded, reuse
	if _, err := os.Stat(execPath); err == nil {
		return execPath
	}

	osName := runtime.GOOS
	archName := runtime.GOARCH

	// Compose archive name per release naming scheme
	archiveName := fmt.Sprintf("databricks_cli_%s_%s_%s.zip", version, osName, archName)
	url := fmt.Sprintf("https://github.com/databricks/cli/releases/download/v%s/%s", version, archiveName)
	zipPath := filepath.Join(versionDir, archiveName)

	downloadToFile(t, url, zipPath)
	extractFileFromZip(t, zipPath, execName, versionDir)

	return execPath
}

// downloadToFile downloads contents from url into the given destination path
func downloadToFile(t *testing.T, url, destPath string) {
	require.NoError(t, os.MkdirAll(filepath.Dir(destPath), 0o755))
	out, err := os.Create(destPath)
	require.NoError(t, err)
	defer out.Close()

	resp, err := http.Get(url)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, "failed to download %s: %s", url, resp.Status)

	_, err = io.Copy(out, resp.Body)
	require.NoError(t, err)
}

// extractFileFromZip finds a file by name inside a zip archive and writes it
// into destDir using the file's base name. Fails the test if not found.
func extractFileFromZip(t *testing.T, zipPath, fileName, destDir string) {
	r, err := zip.OpenReader(zipPath)
	require.NoError(t, err)
	defer r.Close()

	targetBase := filepath.Base(fileName)
	destPath := filepath.Join(destDir, targetBase)
	var found bool
	for _, f := range r.File {
		if filepath.Base(f.Name) != targetBase {
			continue
		}
		rc, err := f.Open()
		require.NoError(t, err)
		defer rc.Close()

		out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
		require.NoError(t, err)
		_, err = io.Copy(out, rc)
		_ = out.Close()
		require.NoError(t, err)
		found = true
		break
	}
	require.True(t, found, "file %s not found in archive %s", targetBase, zipPath)
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

func RunCommand(t *testing.T, args []string, dir string, env []string) {
	start := time.Now()
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
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

func runWithLog(t *testing.T, cmd *exec.Cmd, out *os.File, tail bool) (string, error) {
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
		return "", err
	}

	go func() {
		processErrCh <- cmd.Wait()
		_ = w.Close()
	}()

	mostRecentLine := ""

	reader := bufio.NewReader(r)
	for {
		line, err := reader.ReadString('\n')
		if tail {
			msg := strings.TrimRight(line, "\n")
			if len(msg) > 0 {
				logWithDurationSince(t, start, msg)
			}
		}
		if len(line) > 0 {
			mostRecentLine = line
			if strings.HasPrefix(line, TestLogPrefix) {
				// if tail is true, we already logged it above
				if !tail {
					logWithDurationSince(t, start, strings.TrimRight(line, "\n"))
				}
			} else {
				_, err = out.WriteString(line)
				require.NoError(t, err)
			}
		}
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
	}

	mostRecentLine = strings.TrimRight(mostRecentLine, "\n")
	skipReason := ""
	if strings.HasPrefix(mostRecentLine, "SKIP_TEST") {
		skipReason = mostRecentLine
	}

	return skipReason, <-processErrCh
}

func logWithDurationSince(t *testing.T, start time.Time, msg string) {
	d := time.Since(start)
	t.Logf("%2d.%03d %s", d/time.Second, (d%time.Second)/time.Millisecond, msg)
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
		return "aws"
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
		return "i3.xlarge"
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

	RunCommand(t, []string{"uv", "build", "--no-cache", "-q", "--wheel", "--out-dir", buildDir}, "../python", []string{})

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

func BuildYamlfmt(t *testing.T) {
	// Using make here instead of "go build" directly cause it's faster when it's already built
	args := []string{
		"make", "-s", "tools/yamlfmt" + exeSuffix,
	}
	RunCommand(t, args, "..", []string{})
}

func loadUserReplacements(t *testing.T, repls *testdiff.ReplacementsContext, tmpDir string) {
	b, err := os.ReadFile(filepath.Join(tmpDir, userReplacementsFilename))
	if os.IsNotExist(err) {
		return
	}
	require.NoError(t, err)
	lines := strings.Split(string(b), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		items := strings.Split(line, ":")
		if len(items) <= 1 {
			t.Errorf("Error parsing %s: %#v", userReplacementsFilename, line)
			continue
		}
		repl := items[len(items)-1]
		old := line[:len(line)-len(repl)-1]
		repls.SetWithOrder(old, "["+repl+"]", -100)
	}
}

type pathFilter struct {
	// contains substrings from the variants other than current.
	// E.g. if EnvVaryOutput is DATABRICKS_BUNDLE_ENGINE and current test running DATABRICKS_BUNDLE_ENGINE="terraform" then
	// notSelected contains ".direct." meaning if filename contains that (e.g. out.deploy.direct.txt) then we ignore it here.
	notSelected []string
}

// preparePathFilter builds filter based on EnvVaryOutput and current variant env.
func preparePathFilter(config internal.TestConfig, customEnv []string) pathFilter {
	if config.EnvVaryOutput == nil {
		return pathFilter{}
	}
	key := *config.EnvVaryOutput
	vals := config.EnvMatrix[key]
	if len(vals) <= 1 {
		return pathFilter{}
	}
	selected := ""
	for _, kv := range customEnv {
		items := strings.SplitN(kv, "=", 2)
		if len(items) == 2 && items[0] == key {
			selected = items[1]
			break
		}
	}
	if selected == "" {
		return pathFilter{}
	}
	var others []string
	for _, v := range vals {
		if v == selected {
			continue
		}
		others = append(others, "."+v+".")
	}
	return pathFilter{
		notSelected: others,
	}
}

// shouldSkip returns true if the given file belongs to a non-selected variant.
func shouldSkip(filter pathFilter, relPath string) bool {
	for _, infix := range filter.notSelected {
		if strings.Contains(relPath, infix) {
			return true
		}
	}
	return false
}

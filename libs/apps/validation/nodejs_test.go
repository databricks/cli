package validation_test

import (
	"encoding/json"
	"os"
	osexec "os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/apps/validation"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var validationScripts = []string{"typegen", "lint:ast-grep", "typecheck", "build", "test"}

// These values define the protocol shared with testdata/package-manager and .cmd.
const (
	validationTestFailEnv  = "VALIDATION_TEST_FAIL"
	validationTestExitCode = 17
	validationTestStdout   = "validation stdout"
	validationTestStderr   = "validation stderr"
)

const npmValidationCommands = `npm install
npm run --if-present typegen
npm run --if-present lint:ast-grep
npm run --if-present typecheck
npm run --if-present build
npm run --if-present test
`

// stubPackageManagers records commands without installing packages or requiring Node.js.
func stubPackageManagers(t *testing.T) {
	t.Helper()
	binDir := t.TempDir()
	for _, manager := range []string{"npm", "pnpm"} {
		for _, suffix := range []string{"", ".cmd"} {
			stub := testutil.ReadFile(t, "testdata/package-manager"+suffix)
			require.NoError(t, os.WriteFile(filepath.Join(binDir, manager+suffix), []byte(stub), 0o755))
		}
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv(validationTestFailEnv, "")
}

// writeValidationProject creates a manifest with the requested scripts.
func writeValidationProject(t *testing.T, dir string, scripts []string) {
	t.Helper()
	scriptMap := make(map[string]string, len(scripts))
	for _, script := range scripts {
		scriptMap[script] = "echo " + script
	}
	data, err := json.Marshal(map[string]any{"scripts": scriptMap})
	require.NoError(t, err)
	testutil.WriteFile(t, filepath.Join(dir, "package.json"), string(data))
}

// isolateValidationShell keeps installed package managers out of PATH.
func isolateValidationShell(t *testing.T) {
	t.Helper()
	shell := "sh"
	if runtime.GOOS == "windows" {
		shell = "cmd.exe"
	}
	path, err := osexec.LookPath(shell)
	require.NoError(t, err)
	binDir := t.TempDir()
	if runtime.GOOS == "windows" {
		// Creating symlinks requires elevated privileges on Windows.
		testutil.CopyFile(t, path, filepath.Join(binDir, shell))
	} else {
		require.NoError(t, os.Symlink(path, filepath.Join(binDir, shell)))
	}
	t.Setenv("PATH", binDir)
}

func TestNodeJsValidatePackageManagers(t *testing.T) {
	stubPackageManagers(t)
	tests := []struct {
		name     string
		manager  string
		lockfile string
		manifest string
	}{
		{name: "npm default", manager: "npm"},
		{name: "npm lockfile", manager: "npm", lockfile: "package-lock.json"},
		{name: "pnpm lockfile", manager: "pnpm", lockfile: "pnpm-lock.yaml"},
		{name: "pnpm declaration", manager: "pnpm", manifest: `{"packageManager":"pnpm@10.30.3"}`},
		{name: "npm declaration overrides lockfile", manager: "npm", lockfile: "pnpm-lock.yaml", manifest: `{"packageManager":"npm@10.9.2"}`},
		{name: "pnpm declaration overrides lockfile", manager: "pnpm", lockfile: "package-lock.json", manifest: `{"packageManager":"pnpm@10.30.3"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeValidationProject(t, dir, validationScripts)
			if tt.manifest != "" {
				testutil.WriteFile(t, filepath.Join(dir, "package.json"), tt.manifest)
			}
			if tt.lockfile != "" {
				testutil.Touch(t, dir, tt.lockfile)
			}
			validator := validation.ValidationNodeJs{}
			result, err := validator.Validate(cmdio.MockDiscard(t.Context()), dir, validation.ValidateOptions{})
			require.NoError(t, err)
			require.True(t, result.Success, "%+v", result.Details)
			wantCommands := strings.ReplaceAll(npmValidationCommands, "npm ", tt.manager+" ")
			commands := testutil.ReadFile(t, filepath.Join(dir, "commands.log"))
			assert.Equal(t, wantCommands, strings.ReplaceAll(commands, "\r\n", "\n"))
		})
	}
}

func TestNodeJsValidateSkips(t *testing.T) {
	stubPackageManagers(t)
	commands := strings.ReplaceAll(npmValidationCommands, "npm ", "pnpm ")
	tests := []struct {
		name        string
		scripts     []string
		nodeModules bool
		skipTests   bool
		want        string
	}{
		{name: "missing scripts", want: commands},
		{name: "optional scripts", scripts: []string{"build"}, want: commands},
		{name: "installed dependencies", scripts: []string{"build"}, nodeModules: true, want: strings.TrimPrefix(commands, "pnpm install\n")},
		{name: "skip tests", scripts: []string{"build", "test"}, skipTests: true, want: strings.TrimSuffix(commands, "pnpm run --if-present test\n")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeValidationProject(t, dir, tt.scripts)
			testutil.Touch(t, dir, "pnpm-lock.yaml")
			if tt.nodeModules {
				require.NoError(t, os.Mkdir(filepath.Join(dir, "node_modules"), 0o755))
			}
			validator := validation.ValidationNodeJs{}
			result, err := validator.Validate(cmdio.MockDiscard(t.Context()), dir, validation.ValidateOptions{SkipTests: tt.skipTests})
			require.NoError(t, err)
			require.True(t, result.Success, "%+v", result.Details)
			commands := testutil.ReadFile(t, filepath.Join(dir, "commands.log"))
			assert.Equal(t, tt.want, strings.ReplaceAll(commands, "\r\n", "\n"))
		})
	}
}

func TestNodeJsValidateCommandFailure(t *testing.T) {
	stubPackageManagers(t)
	for _, command := range []string{"install", "run --if-present typecheck", "run --if-present test"} {
		t.Run(command, func(t *testing.T) {
			dir := t.TempDir()
			writeValidationProject(t, dir, validationScripts)
			testutil.Touch(t, dir, "pnpm-lock.yaml")
			t.Setenv(validationTestFailEnv, command)
			validator := validation.ValidationNodeJs{}
			result, err := validator.Validate(cmdio.MockDiscard(t.Context()), dir, validation.ValidateOptions{})
			require.NoError(t, err)
			require.False(t, result.Success)
			require.NotNil(t, result.Details)
			assert.Equal(t, validationTestExitCode, result.Details.ExitCode)
			assert.Contains(t, result.Details.Stdout, validationTestStdout)
			assert.Contains(t, result.Details.Stderr, validationTestStderr)
			commands := testutil.ReadFile(t, filepath.Join(dir, "commands.log"))
			assert.True(t, strings.HasSuffix(strings.ReplaceAll(commands, "\r\n", "\n"), "pnpm "+command+"\n"), commands)
		})
	}
}

func TestNodeJsValidateManagerFromRelativePath(t *testing.T) {
	// Read the stub before chdir so the relative testdata path still resolves.
	stub := testutil.ReadFile(t, "testdata/package-manager")
	stubCmd := testutil.ReadFile(t, "testdata/package-manager.cmd")
	for _, workDir := range []string{".", "app"} {
		t.Run(workDir, func(t *testing.T) {
			isolateValidationShell(t)
			dir := t.TempDir()
			projectDir := filepath.Join(dir, workDir)
			// The manager is only available relative to the project's directory.
			relBin := filepath.Join("node_modules", ".bin")
			require.NoError(t, os.MkdirAll(filepath.Join(projectDir, relBin), 0o755))
			require.NoError(t, os.WriteFile(filepath.Join(projectDir, relBin, "pnpm"), []byte(stub), 0o755))
			require.NoError(t, os.WriteFile(filepath.Join(projectDir, relBin, "pnpm.cmd"), []byte(stubCmd), 0o755))
			writeValidationProject(t, projectDir, []string{"build"})
			testutil.Touch(t, projectDir, "pnpm-lock.yaml")

			t.Chdir(dir)
			t.Setenv("PATH", relBin+string(os.PathListSeparator)+os.Getenv("PATH"))
			t.Setenv(validationTestFailEnv, "")

			validator := validation.ValidationNodeJs{}
			result, err := validator.Validate(cmdio.MockDiscard(t.Context()), workDir, validation.ValidateOptions{})
			require.NoError(t, err)
			require.True(t, result.Success, "%+v", result.Details)
			commands := testutil.ReadFile(t, filepath.Join(projectDir, "commands.log"))
			// node_modules exists, so install is skipped and scripts run via the relative pnpm.
			want := strings.ReplaceAll(strings.TrimPrefix(npmValidationCommands, "npm install\n"), "npm ", "pnpm ")
			assert.Equal(t, want, strings.ReplaceAll(commands, "\r\n", "\n"))
		})
	}
}

func TestNodeJsValidateMissingPackageManager(t *testing.T) {
	isolateValidationShell(t)
	dir := t.TempDir()
	writeValidationProject(t, dir, validationScripts)
	testutil.Touch(t, dir, "pnpm-lock.yaml")
	validator := validation.ValidationNodeJs{}
	result, err := validator.Validate(cmdio.MockDiscard(t.Context()), dir, validation.ValidateOptions{})
	require.NoError(t, err)
	require.False(t, result.Success)
	require.NotNil(t, result.Details)
	assert.NotZero(t, result.Details.ExitCode)
	assert.Contains(t, result.Details.Stderr, "pnpm")
	assert.NoFileExists(t, filepath.Join(dir, "commands.log"))
}

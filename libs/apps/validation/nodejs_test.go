package validation_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/apps/validation"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var validationScripts = []string{"typegen", "lint:ast-grep", "typecheck", "build", "test"}

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
	t.Setenv("VALIDATION_TEST_FAIL", "")
	t.Setenv("VALIDATION_TEST_UPDATE", "")
	t.Setenv("VALIDATION_TEST_PACKAGE_JSON", "")
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

func TestNodeJsValidatePackageManagers(t *testing.T) {
	stubPackageManagers(t)
	tests := []struct {
		manager  string
		lockfile string
	}{
		{manager: "npm"},
		{manager: "npm", lockfile: "package-lock.json"},
		{manager: "pnpm", lockfile: "pnpm-lock.yaml"},
	}
	for _, tt := range tests {
		t.Run(tt.manager+"/"+tt.lockfile, func(t *testing.T) {
			dir := t.TempDir()
			writeValidationProject(t, dir, validationScripts)
			if tt.lockfile != "" {
				testutil.Touch(t, dir, tt.lockfile)
			}
			validator := validation.ValidationNodeJs{}
			result, err := validator.Validate(cmdio.MockDiscard(t.Context()), dir, validation.ValidateOptions{})
			require.NoError(t, err)
			require.True(t, result.Success, "%+v", result.Details)
			wantCommands := []string{tt.manager + " install"}
			for _, script := range validationScripts {
				wantCommands = append(wantCommands, tt.manager+" run --if-present "+script)
			}
			commands := testutil.ReadFile(t, filepath.Join(dir, "commands.log"))
			assert.Equal(t, strings.Join(wantCommands, "\n")+"\n", strings.ReplaceAll(commands, "\r\n", "\n"))
		})
	}
}

func TestNodeJsValidateSkips(t *testing.T) {
	stubPackageManagers(t)
	tests := []struct {
		name        string
		scripts     []string
		nodeModules bool
		skipTests   bool
		want        string
	}{
		{name: "missing scripts", want: "pnpm install\n"},
		{name: "optional scripts", scripts: []string{"build"}, want: "pnpm install\npnpm run --if-present build\n"},
		{name: "installed dependencies", scripts: []string{"build"}, nodeModules: true, want: "pnpm run --if-present build\n"},
		{name: "skip tests", scripts: []string{"build", "test"}, skipTests: true, want: "pnpm install\npnpm run --if-present build\n"},
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
			t.Setenv("VALIDATION_TEST_FAIL", command)
			validator := validation.ValidationNodeJs{}
			result, err := validator.Validate(cmdio.MockDiscard(t.Context()), dir, validation.ValidateOptions{})
			require.NoError(t, err)
			require.False(t, result.Success)
			require.NotNil(t, result.Details)
			assert.Equal(t, 17, result.Details.ExitCode)
			assert.Contains(t, result.Details.Stdout, "validation stdout")
			assert.Contains(t, result.Details.Stderr, "validation stderr")
			commands := testutil.ReadFile(t, filepath.Join(dir, "commands.log"))
			assert.True(t, strings.HasSuffix(strings.ReplaceAll(commands, "\r\n", "\n"), "pnpm "+command+"\n"), commands)
		})
	}
}

func TestNodeJsValidateUpdatedScripts(t *testing.T) {
	stubPackageManagers(t)
	tests := []struct {
		name           string
		command        string
		scripts        []string
		updatedScripts []string
		wantCommands   string
		wantSuccess    bool
	}{
		{
			name:           "install adds build",
			command:        "install",
			updatedScripts: []string{"build"},
			wantCommands:   "npm install\nnpm run --if-present build\n",
		},
		{
			name:           "typegen adds build",
			command:        "run --if-present typegen",
			scripts:        []string{"typegen"},
			updatedScripts: []string{"typegen", "build"},
			wantCommands:   "npm install\nnpm run --if-present typegen\nnpm run --if-present build\n",
		},
		{
			name:         "install removes build",
			command:      "install",
			scripts:      []string{"build"},
			wantCommands: "npm install\n",
			wantSuccess:  true,
		},
		{
			name:           "typegen removes build",
			command:        "run --if-present typegen",
			scripts:        []string{"typegen", "build"},
			updatedScripts: []string{"typegen"},
			wantCommands:   "npm install\nnpm run --if-present typegen\n",
			wantSuccess:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			writeValidationProject(t, dir, tt.scripts)
			updatedDir := t.TempDir()
			writeValidationProject(t, updatedDir, tt.updatedScripts)
			t.Setenv("VALIDATION_TEST_UPDATE", tt.command)
			t.Setenv("VALIDATION_TEST_PACKAGE_JSON", filepath.Join(updatedDir, "package.json"))
			t.Setenv("VALIDATION_TEST_FAIL", "run --if-present build")
			validator := validation.ValidationNodeJs{}
			result, err := validator.Validate(cmdio.MockDiscard(t.Context()), dir, validation.ValidateOptions{})
			require.NoError(t, err)
			assert.Equal(t, tt.wantSuccess, result.Success)
			if !tt.wantSuccess {
				require.NotNil(t, result.Details)
				assert.Equal(t, "Failed to build", result.Message)
				assert.Equal(t, 17, result.Details.ExitCode)
			}
			commands := testutil.ReadFile(t, filepath.Join(dir, "commands.log"))
			assert.Equal(t, tt.wantCommands, strings.ReplaceAll(commands, "\r\n", "\n"))
		})
	}
}

func TestNodeJsValidatePackageJSONBOM(t *testing.T) {
	stubPackageManagers(t)
	dir := t.TempDir()
	writeValidationProject(t, dir, []string{"build"})
	path := filepath.Join(dir, "package.json")
	testutil.WriteFile(t, path, "\xef\xbb\xbf"+testutil.ReadFile(t, path))
	validator := validation.ValidationNodeJs{}
	result, err := validator.Validate(cmdio.MockDiscard(t.Context()), dir, validation.ValidateOptions{})
	require.NoError(t, err)
	require.True(t, result.Success, "%+v", result.Details)
	commands := testutil.ReadFile(t, filepath.Join(dir, "commands.log"))
	assert.Equal(t, "npm install\nnpm run --if-present build\n", strings.ReplaceAll(commands, "\r\n", "\n"))
}

func TestNodeJsValidateNonRunnableScripts(t *testing.T) {
	stubPackageManagers(t)
	dir := t.TempDir()
	// null, "", and non-string values aren't runnable scripts for npm/pnpm, and a
	// non-string value must not fail decoding. Only "build" should run.
	manifest := `{"scripts": {"typecheck": null, "test": "", "//": ["a comment"], "build": "echo build"}}`
	testutil.WriteFile(t, filepath.Join(dir, "package.json"), manifest)
	testutil.Touch(t, dir, "pnpm-lock.yaml")
	validator := validation.ValidationNodeJs{}
	result, err := validator.Validate(cmdio.MockDiscard(t.Context()), dir, validation.ValidateOptions{})
	require.NoError(t, err)
	require.True(t, result.Success, "%+v", result.Details)
	commands := testutil.ReadFile(t, filepath.Join(dir, "commands.log"))
	assert.Equal(t, "pnpm install\npnpm run --if-present build\n", strings.ReplaceAll(commands, "\r\n", "\n"))
}

func TestNodeJsValidateInvalidPackageJSON(t *testing.T) {
	stubPackageManagers(t)
	dir := t.TempDir()
	testutil.WriteFile(t, filepath.Join(dir, "package.json"), "{")
	validator := validation.ValidationNodeJs{}
	result, err := validator.Validate(t.Context(), dir, validation.ValidateOptions{})
	require.ErrorContains(t, err, "failed to parse package.json")
	assert.ErrorAs(t, err, new(*json.SyntaxError))
	assert.Nil(t, result)
	assert.NoFileExists(t, filepath.Join(dir, "commands.log"))
}

func TestNodeJsValidateMissingPackageJSON(t *testing.T) {
	stubPackageManagers(t)
	dir := t.TempDir()
	validator := validation.ValidationNodeJs{}
	result, err := validator.Validate(t.Context(), dir, validation.ValidateOptions{})
	require.ErrorContains(t, err, "failed to read package.json")
	assert.ErrorIs(t, err, os.ErrNotExist)
	assert.Nil(t, result)
	assert.NoFileExists(t, filepath.Join(dir, "commands.log"))
}

func TestNodeJsValidateUpdatedInvalidPackageJSON(t *testing.T) {
	stubPackageManagers(t)
	dir := t.TempDir()
	writeValidationProject(t, dir, validationScripts)
	updatedPath := filepath.Join(t.TempDir(), "package.json")
	testutil.WriteFile(t, updatedPath, "{")
	t.Setenv("VALIDATION_TEST_UPDATE", "install")
	t.Setenv("VALIDATION_TEST_PACKAGE_JSON", updatedPath)
	validator := validation.ValidationNodeJs{}
	result, err := validator.Validate(cmdio.MockDiscard(t.Context()), dir, validation.ValidateOptions{})
	require.ErrorContains(t, err, "failed to parse package.json")
	assert.ErrorAs(t, err, new(*json.SyntaxError))
	assert.Nil(t, result)
	commands := testutil.ReadFile(t, filepath.Join(dir, "commands.log"))
	assert.Equal(t, "npm install\n", strings.ReplaceAll(commands, "\r\n", "\n"))
}

func TestNodeJsValidateManagerFromRelativePath(t *testing.T) {
	dir := t.TempDir()
	// Read the stub before chdir so the relative testdata path still resolves.
	stub := testutil.ReadFile(t, "testdata/package-manager")
	stubCmd := testutil.ReadFile(t, "testdata/package-manager.cmd")
	// Install the manager under a relative PATH entry so LookPath returns ErrDot,
	// as it does when pnpm is a devDependency resolved from node_modules/.bin.
	relBin := filepath.Join("node_modules", ".bin")
	require.NoError(t, os.MkdirAll(filepath.Join(dir, relBin), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, relBin, "pnpm"), []byte(stub), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, relBin, "pnpm.cmd"), []byte(stubCmd), 0o755))
	writeValidationProject(t, dir, []string{"build"})
	testutil.Touch(t, dir, "pnpm-lock.yaml")

	t.Chdir(dir)
	t.Setenv("PATH", relBin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("VALIDATION_TEST_FAIL", "")

	validator := validation.ValidationNodeJs{}
	result, err := validator.Validate(cmdio.MockDiscard(t.Context()), dir, validation.ValidateOptions{})
	require.NoError(t, err)
	require.True(t, result.Success, "%+v", result.Details)
	commands := testutil.ReadFile(t, filepath.Join(dir, "commands.log"))
	// node_modules exists, so install is skipped and build runs via the relative pnpm.
	assert.Equal(t, "pnpm run --if-present build\n", strings.ReplaceAll(commands, "\r\n", "\n"))
}

func TestNodeJsValidateMissingPackageManager(t *testing.T) {
	dir := t.TempDir()
	writeValidationProject(t, dir, validationScripts)
	testutil.Touch(t, dir, "pnpm-lock.yaml")
	t.Setenv("PATH", t.TempDir())
	validator := validation.ValidationNodeJs{}
	_, err := validator.Validate(t.Context(), dir, validation.ValidateOptions{})
	require.ErrorContains(t, err, "cannot run pnpm; install it and ensure it is on PATH")
	assert.NoFileExists(t, filepath.Join(dir, "commands.log"))
}

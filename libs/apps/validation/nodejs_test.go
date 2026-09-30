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
				wantCommands = append(wantCommands, tt.manager+" run "+script)
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
		{name: "optional scripts", scripts: []string{"build"}, want: "pnpm install\npnpm run build\n"},
		{name: "installed dependencies", scripts: []string{"build"}, nodeModules: true, want: "pnpm run build\n"},
		{name: "skip tests", scripts: []string{"build", "test"}, skipTests: true, want: "pnpm install\npnpm run build\n"},
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
	for _, command := range []string{"install", "run typecheck", "run test"} {
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

func TestNodeJsValidateInvalidPackageJSON(t *testing.T) {
	dir := t.TempDir()
	testutil.WriteFile(t, filepath.Join(dir, "package.json"), "{")
	validator := validation.ValidationNodeJs{}
	_, err := validator.Validate(t.Context(), dir, validation.ValidateOptions{})
	require.ErrorContains(t, err, "failed to parse package.json")
	assert.ErrorAs(t, err, new(*json.SyntaxError))
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

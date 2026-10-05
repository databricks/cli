//go:build appkit_smoke

package acceptance_test

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/testserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

const (
	appkitSmokeVersion = "0.82.0"
	appkitSmokeRun     = "36871528478"
	appkitSmokeRelease = "appkit-release-216"
	appkitSmokeBundle  = "appkit-template-216"
	appkitSmokePin     = "pnpm@11.0.8"
	appkitSmokeProfile = "appkit-smoke"
)

// TestAppsInitAppKit082 uses the actual release artifacts and package managers.
// The build tag keeps network downloads and package installation out of TestAccept.
func TestAppsInitAppKit082(t *testing.T) {
	var workDir string
	if KeepTmp {
		var err error
		workDir, err = os.MkdirTemp("", "appkit-082-smoke-") //nolint:usetesting // -keeptmp preserves the generated apps for inspection.
		require.NoError(t, err)
	} else {
		workDir = t.TempDir()
	}
	t.Logf("Smoke test directory: %s", filepath.ToSlash(workDir))

	cli := CLIPath
	if cli == "" {
		cli = BuildCLI(t, workDir, "", runtime.GOOS, runtime.GOARCH)
	}
	cli, err := filepath.Abs(cli)
	require.NoError(t, err)

	// Keep GitHub/package registry credentials, but isolate workspace credentials
	// and template overrides from the developer's environment.
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(key, "DATABRICKS_") || strings.HasPrefix(key, "OTEL_") || strings.HasPrefix(key, "MLFLOW_") {
			t.Setenv(key, "")
		}
	}
	t.Setenv("CI", "true")
	t.Setenv("NODE_ENV", "development")
	t.Setenv("DO_NOT_TRACK", "1")
	t.Setenv("OTEL_SDK_DISABLED", "true")
	t.Setenv("DATABRICKS_CACHE_DIR", filepath.Join(workDir, "cache"))

	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)
	configFile := filepath.Join(workDir, ".databrickscfg")
	testutil.WriteFile(t, configFile, fmt.Sprintf("[%s]\nhost = %s\ntoken = %s\n", appkitSmokeProfile, server.URL, testserver.UserNameTokenPrefix+"appkit-smoke"))
	t.Setenv("DATABRICKS_CONFIG_FILE", configFile)

	t.Run("version_tag", func(t *testing.T) {
		// Exercise the public --version path independently of the artifact path.
		RunCommand(t, []string{
			cli, "apps", "init", "--profile", appkitSmokeProfile,
			"--name", "version-app", "--version", appkitSmokeVersion,
			"--package-manager", "pnpm", "--skip-install",
		}, workDir, nil)
		appDir := filepath.Join(workDir, "version-app")
		checkAppKitSmokeProject(t, appDir, "version-app")
		var pkg struct {
			Dependencies map[string]string `json:"dependencies"`
		}
		require.NoError(t, json.Unmarshal([]byte(testutil.ReadFile(t, filepath.Join(appDir, "package.json"))), &pkg))
		for _, name := range []string{"@databricks/appkit", "@databricks/appkit-ui"} {
			assert.Equal(t, appkitSmokeVersion, pkg.Dependencies[name], name)
		}
	})

	t.Run("release_bundle", func(t *testing.T) {
		templateDir := downloadAppKitSmokeTemplate(t, workDir)
		t.Run("scaffold", func(t *testing.T) {
			RunCommand(t, []string{
				cli, "apps", "init", "--profile", appkitSmokeProfile,
				"--name", "scaffold-app", "--template", templateDir,
				"--package-manager", "pnpm", "--skip-install",
			}, workDir, nil)
			appDir := filepath.Join(workDir, "scaffold-app")
			checkAppKitSmokeProject(t, appDir, "scaffold-app")
			assert.NoDirExists(t, filepath.Join(appDir, "node_modules"))
			for _, name := range []string{
				"pnpm-lock.yaml", "pnpm-workspace.yaml", ".npmrc",
				"databricks-appkit-" + appkitSmokeVersion + ".tgz", "databricks-appkit-ui-" + appkitSmokeVersion + ".tgz",
			} {
				assert.Equal(t, testutil.ReadFile(t, filepath.Join(templateDir, name)), testutil.ReadFile(t, filepath.Join(appDir, name)), name)
			}
		})

		t.Run("frozen_install", func(t *testing.T) {
			RunCommand(t, []string{
				cli, "apps", "init", "--profile", appkitSmokeProfile,
				"--name", "frozen-app", "--template", templateDir,
				"--package-manager", "pnpm", "--skip-install",
			}, workDir, nil)
			appDir := filepath.Join(workDir, "frozen-app")
			RunCommand(t, []string{"node", "--version"}, appDir, nil)
			RunCommand(t, []string{"pnpm", "--version"}, appDir, nil)
			RunCommand(t, []string{"pnpm", "install", "--frozen-lockfile"}, appDir, nil)
			checkInstalledAppKitSmokeVersion(t, appDir)
		})

		t.Run("init_with_install", func(t *testing.T) {
			RunCommand(t, []string{
				cli, "apps", "init", "--profile", appkitSmokeProfile,
				"--name", "installed-app", "--template", templateDir,
				"--package-manager", "pnpm",
			}, workDir, nil)
			installedDir := filepath.Join(workDir, "installed-app")
			checkAppKitSmokeProject(t, installedDir, "installed-app")
			checkInstalledAppKitSmokeVersion(t, installedDir)
		})
	})
}

func downloadAppKitSmokeTemplate(t *testing.T, workDir string) string {
	t.Helper()
	artifactsDir := filepath.Join(workDir, "artifacts")
	RunCommand(t, []string{
		"gh", "run", "download", appkitSmokeRun, "--repo", "databricks/appkit",
		"--name", appkitSmokeRelease, "--name", appkitSmokeBundle, "--dir", artifactsDir,
	}, workDir, nil)
	releaseDir := filepath.Join(artifactsDir, appkitSmokeRelease)
	require.Equal(t, appkitSmokeVersion, strings.TrimSpace(testutil.ReadFile(t, filepath.Join(releaseDir, "VERSION"))))

	archive, err := zip.OpenReader(filepath.Join(artifactsDir, appkitSmokeBundle, "template.zip"))
	require.NoError(t, err)
	defer archive.Close()
	templateDir := filepath.Join(workDir, "template")
	require.NoError(t, os.CopyFS(templateDir, archive))

	checksums := make(map[string]string)
	for line := range strings.Lines(testutil.ReadFile(t, filepath.Join(releaseDir, "SHA256SUMS"))) {
		fields := strings.Fields(line)
		require.Len(t, fields, 2)
		checksums[fields[1]] = fields[0]
	}
	for _, name := range []string{"databricks-appkit-" + appkitSmokeVersion + ".tgz", "databricks-appkit-ui-" + appkitSmokeVersion + ".tgz"} {
		data, err := os.ReadFile(filepath.Join(templateDir, name))
		require.NoError(t, err)
		require.Contains(t, checksums, name)
		require.Equal(t, checksums[name], fmt.Sprintf("%x", sha256.Sum256(data)), "template tarball differs from release: %s", name)
	}
	return templateDir
}

func checkAppKitSmokeProject(t *testing.T, appDir, name string) {
	t.Helper()
	var pkg struct {
		Name           string            `json:"name"`
		PackageManager string            `json:"packageManager"`
		Scripts        map[string]string `json:"scripts"`
	}
	require.NoError(t, json.Unmarshal([]byte(testutil.ReadFile(t, filepath.Join(appDir, "package.json"))), &pkg))
	assert.Equal(t, name, pkg.Name)
	assert.Equal(t, appkitSmokePin, pkg.PackageManager)
	assert.Equal(t, "pnpm run build:server && pnpm run build:client", pkg.Scripts["build"])
	assert.Contains(t, pkg.Scripts["prebuild"], "pnpm run sync")
	var app struct {
		Command []string `yaml:"command"`
	}
	require.NoError(t, yaml.Unmarshal([]byte(testutil.ReadFile(t, filepath.Join(appDir, "app.yaml"))), &app))
	assert.Equal(t, []string{"pnpm", "run", "start"}, app.Command)
	for _, lockfile := range []string{"package-lock.json", "npm-shrinkwrap.json"} {
		assert.NoFileExists(t, filepath.Join(appDir, lockfile))
	}
	assert.FileExists(t, filepath.Join(appDir, "pnpm-lock.yaml"))
	assert.FileExists(t, filepath.Join(appDir, "databricks.yml"))
}

func checkInstalledAppKitSmokeVersion(t *testing.T, appDir string) {
	t.Helper()
	for _, name := range []string{"appkit", "appkit-ui"} {
		var pkg struct {
			Version string `json:"version"`
		}
		path := filepath.Join(appDir, "node_modules", "@databricks", name, "package.json")
		require.NoError(t, json.Unmarshal([]byte(testutil.ReadFile(t, path)), &pkg))
		assert.Equal(t, appkitSmokeVersion, pkg.Version, name)
	}
}

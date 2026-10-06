//go:build appkit_smoke

package acceptance_test

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

// TestAppsInitAppKitLocal exercises a local AppKit template without downloading
// release artifacts. Dependencies are installed from the template's lockfiles.
func TestAppsInitAppKitLocal(t *testing.T) {
	// Read before setupAppKitSmoke clears inherited Databricks variables.
	templateDir := os.Getenv("DATABRICKS_APPKIT_TEMPLATE_PATH")
	if templateDir == "" {
		t.Skip("set DATABRICKS_APPKIT_TEMPLATE_PATH to the absolute path of an AppKit main template directory")
	}
	require.True(t, filepath.IsAbs(templateDir), "DATABRICKS_APPKIT_TEMPLATE_PATH must be absolute")
	var templatePkg struct {
		Dependencies map[string]string `json:"dependencies"`
	}
	require.NoError(t, json.Unmarshal([]byte(testutil.ReadFile(t, filepath.Join(templateDir, "package.json"))), &templatePkg))
	t.Logf("Local template: %s; AppKit dependencies: %v", filepath.ToSlash(templateDir), templatePkg.Dependencies)

	cli, workDir := setupAppKitSmoke(t, "appkit-local-smoke-")
	RunCommand(t, []string{cli, "apps", "manifest", "--profile", appkitSmokeProfile, "--template", templateDir}, workDir, nil)
	RunCommand(t, []string{"node", "--version"}, workDir, nil)

	for _, manager := range []string{"pnpm", "npm"} {
		t.Run(manager, func(t *testing.T) {
			for _, stage := range []struct {
				name        string
				skipInstall bool
			}{
				{name: "scaffold", skipInstall: true},
				{name: "init_with_install"},
			} {
				t.Run(stage.name, func(t *testing.T) {
					name := manager + "-" + strings.ReplaceAll(stage.name, "_", "-")
					appDir := filepath.Join(workDir, name)
					args := []string{
						cli, "apps", "init", "--profile", appkitSmokeProfile,
						"--name", name, "--template", templateDir,
						"--package-manager", manager, "--run", "none",
					}
					pin := appkitSmokePin
					if manager == "npm" {
						pin = ""
						if !stage.skipInstall {
							cmd := exec.CommandContext(t.Context(), "npm", "--version")
							cmd.Dir = workDir
							cmd.Env = append(os.Environ(), "COREPACK_ENABLE_PROJECT_SPEC=0")
							out, err := cmd.Output()
							require.NoError(t, err)
							pin = "npm@" + strings.TrimSpace(string(out))
						}
					}
					if stage.skipInstall {
						args = append(args, "--skip-install")
					}
					RunCommand(t, args, workDir, nil)

					var pkg struct {
						Name           string            `json:"name"`
						PackageManager string            `json:"packageManager"`
						Scripts        map[string]string `json:"scripts"`
						Dependencies   map[string]string `json:"dependencies"`
					}
					require.NoError(t, json.Unmarshal([]byte(testutil.ReadFile(t, filepath.Join(appDir, "package.json"))), &pkg))
					assert.Equal(t, name, pkg.Name)
					assert.Equal(t, pin, pkg.PackageManager)
					assert.Equal(t, templatePkg.Dependencies, pkg.Dependencies)
					assert.Equal(t, fmt.Sprintf("%s run build:server && %s run build:client", manager, manager), pkg.Scripts["build"])
					assert.Contains(t, pkg.Scripts["prebuild"], manager+" run sync")
					var app struct {
						Command []string `yaml:"command"`
					}
					require.NoError(t, yaml.Unmarshal([]byte(testutil.ReadFile(t, filepath.Join(appDir, "app.yaml"))), &app))
					assert.Equal(t, []string{manager, "run", "start"}, app.Command)
					assert.FileExists(t, filepath.Join(appDir, "databricks.yml"))
					assert.NoFileExists(t, filepath.Join(appDir, "npm-shrinkwrap.json"))
					for _, file := range []string{"package-lock.json", "pnpm-lock.yaml", "pnpm-workspace.yaml", ".npmrc"} {
						keep := file == ".npmrc" || (manager == "npm" && file == "package-lock.json") || (manager == "pnpm" && strings.HasPrefix(file, "pnpm-"))
						if keep {
							assert.Equal(t, testutil.ReadFile(t, filepath.Join(templateDir, file)), testutil.ReadFile(t, filepath.Join(appDir, file)), file)
						} else {
							assert.NoFileExists(t, filepath.Join(appDir, file))
						}
					}
					if stage.skipInstall {
						assert.NoDirExists(t, filepath.Join(appDir, "node_modules"))
						return
					}

					for _, packageName := range []string{"appkit", "appkit-ui"} {
						var installed struct {
							Version string `json:"version"`
						}
						path := filepath.Join(appDir, "node_modules", "@databricks", packageName, "package.json")
						require.NoError(t, json.Unmarshal([]byte(testutil.ReadFile(t, path)), &installed))
						assert.Equal(t, templatePkg.Dependencies["@databricks/"+packageName], installed.Version, packageName)
					}
					t.Run("build", func(t *testing.T) {
						RunCommand(t, []string{manager, "run", "build"}, appDir, nil)
						assert.FileExists(t, filepath.Join(appDir, "dist", "server.js"))
						assert.FileExists(t, filepath.Join(appDir, "client", "dist", "index.html"))
					})
				})
			}
		})
	}
}

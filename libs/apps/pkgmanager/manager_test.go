package pkgmanager_test

import (
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/apps/pkgmanager"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolvePnpm(t *testing.T) {
	m, err := pkgmanager.Resolve("pnpm")
	require.NoError(t, err)
	assert.Equal(t, "pnpm", m.Name)
	assert.Equal(t, "pnpm install --frozen-lockfile", m.InstallCommand)
	assert.Equal(t, "pnpm-lock.yaml", m.LockfileName)
	assert.Equal(t, "pnpm-workspace.yaml", m.WorkspaceConfigName)
	assert.Equal(t, "pnpm@11.0.8", m.Pin)
}

func TestResolveNpm(t *testing.T) {
	m, err := pkgmanager.Resolve("npm")
	require.NoError(t, err)
	assert.Equal(t, "npm", m.Name)
	assert.Equal(t, "npm ci", m.InstallCommand)
	assert.Equal(t, "package-lock.json", m.LockfileName)
	assert.Empty(t, m.WorkspaceConfigName)
	assert.Equal(t, "npm@10.9.2", m.Pin)
}

func TestResolveUnknown(t *testing.T) {
	_, err := pkgmanager.Resolve("unknown")
	require.Error(t, err)
	assert.Equal(t, `unknown package manager "unknown" (allowed: npm, pnpm)`, err.Error())
}

func TestDefault(t *testing.T) {
	m := pkgmanager.Default()
	assert.Equal(t, "pnpm", m.Name)
	assert.Equal(t, "pnpm install --frozen-lockfile", m.InstallCommand)
	resolved, err := pkgmanager.Resolve("")
	require.NoError(t, err)
	assert.Equal(t, m, resolved)
}

func TestValidateTemplate(t *testing.T) {
	for _, name := range []string{"npm", "pnpm"} {
		m, err := pkgmanager.Resolve(name)
		require.NoError(t, err)
		for _, files := range [][]string{nil, {"package-lock.json"}, {"pnpm-lock.yaml"}, {"package-lock.json", "pnpm-lock.yaml"}} {
			t.Run(name+"/"+strings.Join(files, "+"), func(t *testing.T) {
				dir := t.TempDir()
				for _, file := range files {
					require.NoError(t, os.WriteFile(filepath.Join(dir, file), []byte("lockfile"), 0o644))
				}
				err := m.ValidateTemplate(dir)
				if len(files) == 1 && files[0] != m.LockfileName {
					other := "npm"
					if name == "npm" {
						other = "pnpm"
					}
					require.ErrorContains(t, err, "use --package-manager "+other)
					assert.ErrorContains(t, err, m.LockfileName)
				} else {
					require.NoError(t, err)
				}
			})
		}
	}
}

func TestPrunePnpm(t *testing.T) {
	dir := t.TempDir()

	// Create all PM-specific artifacts
	for _, file := range []string{"pnpm-lock.yaml", "package-lock.json", "pnpm-workspace.yaml", "package.json"} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, file), []byte("content"), 0o644))
	}

	m, err := pkgmanager.Resolve("pnpm")
	require.NoError(t, err)
	require.NoError(t, m.Prune(dir))

	// pnpm should keep its lockfile and workspace config
	assert.FileExists(t, filepath.Join(dir, "pnpm-lock.yaml"))
	assert.FileExists(t, filepath.Join(dir, "pnpm-workspace.yaml"))
	assert.FileExists(t, filepath.Join(dir, "package.json"))

	// pnpm should remove npm lockfile
	assert.NoFileExists(t, filepath.Join(dir, "package-lock.json"))
}

func TestPruneNpm(t *testing.T) {
	dir := t.TempDir()

	// Create all PM-specific artifacts
	for _, file := range []string{"pnpm-lock.yaml", "package-lock.json", "pnpm-workspace.yaml", "package.json"} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, file), []byte("content"), 0o644))
	}

	m, err := pkgmanager.Resolve("npm")
	require.NoError(t, err)
	require.NoError(t, m.Prune(dir))

	// npm should keep its lockfile only
	assert.FileExists(t, filepath.Join(dir, "package-lock.json"))
	assert.FileExists(t, filepath.Join(dir, "package.json"))

	// npm should remove pnpm lockfile and workspace config
	assert.NoFileExists(t, filepath.Join(dir, "pnpm-lock.yaml"))
	assert.NoFileExists(t, filepath.Join(dir, "pnpm-workspace.yaml"))
}

func TestPruneIdempotent(t *testing.T) {
	dir := t.TempDir()

	// Only create pnpm files
	require.NoError(t, os.WriteFile(filepath.Join(dir, "pnpm-lock.yaml"), []byte("content"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "package.json"), []byte("content"), 0o644))

	m, err := pkgmanager.Resolve("pnpm")
	require.NoError(t, err)

	// Prune should succeed even when npm artifacts don't exist
	require.NoError(t, m.Prune(dir))
	require.NoError(t, m.Prune(dir))

	// pnpm artifacts should remain
	assert.FileExists(t, filepath.Join(dir, "pnpm-lock.yaml"))
	assert.FileExists(t, filepath.Join(dir, "package.json"))
}

func TestRewrite(t *testing.T) {
	tests := []struct {
		name            string
		selectedManager string
		input           map[string]any
		expectedPin     string
		expectedScripts map[string]string
	}{
		{
			name:            "pnpm selected - rewrite npm scripts",
			selectedManager: "pnpm",
			input: map[string]any{
				"name": "my-app",
				"scripts": map[string]any{
					"build": "npm run build",
					"dev":   "npm run dev",
				},
				"dependencies": map[string]any{
					"react": "^18.0.0",
				},
			},
			expectedPin: "pnpm@11.0.8",
			expectedScripts: map[string]string{
				"build": "pnpm run build",
				"dev":   "pnpm run dev",
			},
		},
		{
			name:            "npm selected - rewrite pnpm scripts",
			selectedManager: "npm",
			input: map[string]any{
				"name": "my-app",
				"scripts": map[string]any{
					"build": "pnpm run build",
					"dev":   "pnpm dev",
				},
			},
			expectedPin: "npm@10.9.2",
			expectedScripts: map[string]string{
				"build": "npm run build",
				"dev":   "npm run dev",
			},
		},
		{
			name:            "non-PM leading token - unchanged",
			selectedManager: "pnpm",
			input: map[string]any{
				"scripts": map[string]any{
					"dev": "vite dev",
				},
			},
			expectedPin: "pnpm@11.0.8",
			expectedScripts: map[string]string{
				"dev": "vite dev",
			},
		},
		{
			name:            "already has run keyword - no double run",
			selectedManager: "pnpm",
			input: map[string]any{
				"scripts": map[string]any{
					"build": "npm run build",
				},
			},
			expectedPin: "pnpm@11.0.8",
			expectedScripts: map[string]string{
				"build": "pnpm run build",
			},
		},
		{
			name:            "mixed scripts - some rewritten, some unchanged",
			selectedManager: "pnpm",
			input: map[string]any{
				"scripts": map[string]any{
					"build":  "npm run build",
					"lint":   "eslint .",
					"dev":    "pnpm dev",
					"bundle": "vite build",
				},
			},
			expectedPin: "pnpm@11.0.8",
			expectedScripts: map[string]string{
				"build":  "pnpm run build",
				"lint":   "eslint .",
				"dev":    "pnpm run dev",
				"bundle": "vite build",
			},
		},
		{
			name:            "non-string script value - skip",
			selectedManager: "pnpm",
			input: map[string]any{
				"scripts": map[string]any{
					"build":  "npm run build",
					"matrix": []string{"npm run a", "npm run b"},
				},
			},
			expectedPin: "pnpm@11.0.8",
			expectedScripts: map[string]string{
				"build": "pnpm run build",
			},
		},
		{
			name:            "dependencies and overrides unchanged",
			selectedManager: "pnpm",
			input: map[string]any{
				"scripts": map[string]any{
					"build": "npm run build",
				},
				"dependencies": map[string]any{
					"react": "^18.0.0",
				},
				"devDependencies": map[string]any{
					"typescript": "^5.0.0",
				},
				"overrides": map[string]any{
					"lodash": "4.17.21",
				},
			},
			expectedPin: "pnpm@11.0.8",
			expectedScripts: map[string]string{
				"build": "pnpm run build",
			},
		},
		{
			name:            "npm-prefixed alias in dependencies - unchanged",
			selectedManager: "pnpm",
			input: map[string]any{
				"scripts": map[string]any{
					"build": "npm run build",
				},
				"dependencies": map[string]any{
					"foo": "npm:bar@^1.0.0",
				},
			},
			expectedPin: "pnpm@11.0.8",
			expectedScripts: map[string]string{
				"build": "pnpm run build",
			},
		},
		{
			name:            "yarn and bun scripts also rewritten",
			selectedManager: "pnpm",
			input: map[string]any{
				"scripts": map[string]any{
					"yarn-build": "yarn run build",
					"bun-dev":    "bun dev",
					"dev":        "vite dev",
				},
			},
			expectedPin: "pnpm@11.0.8",
			expectedScripts: map[string]string{
				"yarn-build": "pnpm run build",
				"bun-dev":    "pnpm run dev",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := pkgmanager.Resolve(tt.selectedManager)
			require.NoError(t, err)

			// Make a copy of the input to avoid mutating the test's input
			inputCopy := make(map[string]any)
			maps.Copy(inputCopy, tt.input)

			// Call Rewrite
			result := pkgmanager.Rewrite(inputCopy, m)

			// Verify packageManager is set to the expected pin
			assert.Equal(t, tt.expectedPin, result["packageManager"])

			// Verify scripts are rewritten as expected
			scripts, ok := result["scripts"].(map[string]any)
			assert.True(t, ok, "scripts should still be a map")

			for name, expectedVal := range tt.expectedScripts {
				actualVal, ok := scripts[name].(string)
				assert.True(t, ok, "script %q should be a string", name)
				assert.Equal(t, expectedVal, actualVal, "script %q", name)
			}

			// Verify dependencies, devDependencies, and overrides are unchanged
			if deps, ok := tt.input["dependencies"].(map[string]any); ok {
				resultDeps, ok := result["dependencies"].(map[string]any)
				assert.True(t, ok, "dependencies should still be a map")
				assert.Equal(t, deps, resultDeps, "dependencies should be unchanged")
			}
			if devDeps, ok := tt.input["devDependencies"].(map[string]any); ok {
				resultDevDeps, ok := result["devDependencies"].(map[string]any)
				assert.True(t, ok, "devDependencies should still be a map")
				assert.Equal(t, devDeps, resultDevDeps, "devDependencies should be unchanged")
			}
			if overrides, ok := tt.input["overrides"].(map[string]any); ok {
				resultOverrides, ok := result["overrides"].(map[string]any)
				assert.True(t, ok, "overrides should still be a map")
				assert.Equal(t, overrides, resultOverrides, "overrides should be unchanged")
			}
		})
	}
}

func TestEffectiveManager(t *testing.T) {
	pnpm, err := pkgmanager.Resolve("pnpm")
	require.NoError(t, err)
	npm, err := pkgmanager.Resolve("npm")
	require.NoError(t, err)

	tests := []struct {
		name            string
		selectedManager pkgmanager.Manager
		version         string
		expectManager   string
		expectDowngrade bool
	}{
		// Explicit pnpm requests with concrete versions
		{
			name:            "pnpm with version below threshold",
			selectedManager: pnpm,
			version:         "template-v0.81.0",
			expectManager:   "npm",
			expectDowngrade: true,
		},
		{
			name:            "pnpm with version at threshold",
			selectedManager: pnpm,
			version:         "template-v0.82.0",
			expectManager:   "pnpm",
			expectDowngrade: false,
		},
		{
			name:            "pnpm with version above threshold",
			selectedManager: pnpm,
			version:         "template-v0.83.0",
			expectManager:   "pnpm",
			expectDowngrade: false,
		},
		{
			name:            "pnpm with v-prefixed version below threshold",
			selectedManager: pnpm,
			version:         "v0.81.0",
			expectManager:   "npm",
			expectDowngrade: true,
		},
		{
			name:            "pnpm with unprefixed version below threshold",
			selectedManager: pnpm,
			version:         "0.81.0",
			expectManager:   "npm",
			expectDowngrade: true,
		},

		// Explicit pnpm requests with non-concrete versions
		{
			name:            "pnpm with main branch",
			selectedManager: pnpm,
			version:         "main",
			expectManager:   "pnpm",
			expectDowngrade: false,
		},
		{
			name:            "pnpm with empty version",
			selectedManager: pnpm,
			version:         "",
			expectManager:   "pnpm",
			expectDowngrade: false,
		},
		{
			name:            "pnpm with branch name",
			selectedManager: pnpm,
			version:         "feature-x",
			expectManager:   "pnpm",
			expectDowngrade: false,
		},

		// Explicit npm requests
		{
			name:            "npm with version below threshold",
			selectedManager: npm,
			version:         "template-v0.81.0",
			expectManager:   "npm",
			expectDowngrade: false,
		},
		{
			name:            "npm with version above threshold",
			selectedManager: npm,
			version:         "template-v0.83.0",
			expectManager:   "npm",
			expectDowngrade: false,
		},
		{
			name:            "npm with main branch",
			selectedManager: npm,
			version:         "main",
			expectManager:   "npm",
			expectDowngrade: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			effective, downgraded := pkgmanager.EffectiveManager(tt.selectedManager, tt.version)
			assert.Equal(t, tt.expectManager, effective.Name, "manager name mismatch")
			assert.Equal(t, tt.expectDowngrade, downgraded, "downgrade bool mismatch")
		})
	}
}

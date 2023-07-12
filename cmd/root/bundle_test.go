package root

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/stretchr/testify/assert"
)

func setupDatabricksCfg(t *testing.T) {
	tempHomeDir := t.TempDir()
	homeEnvVar := "HOME"
	if runtime.GOOS == "windows" {
		homeEnvVar = "USERPROFILE"
	}

	cfg := []byte("[PROFILE-1]\nhost = https://a.com\ntoken = a\n[PROFILE-2]\nhost = https://a.com\ntoken = b\n")
	err := os.WriteFile(filepath.Join(tempHomeDir, ".databrickscfg"), cfg, 0644)
	assert.NoError(t, err)

	t.Setenv("DATABRICKS_CONFIG_FILE", "")
	t.Setenv(homeEnvVar, tempHomeDir)
}

func setup(t *testing.T, host string) *bundle.Bundle {
	setupDatabricksCfg(t)

	ctx := context.Background()
	RootCmd.SetContext(ctx)
	_, err := initializeLogger(ctx)
	assert.NoError(t, err)

	err = configureBundle(RootCmd, []string{"validate"}, func() (*bundle.Bundle, error) {
		return &bundle.Bundle{
			Config: config.Root{
				Bundle: config.Bundle{
					Name: "test",
				},
				Workspace: config.Workspace{
					Host: host,
				},
			},
		}, nil
	})
	assert.NoError(t, err)

	return bundle.Get(RootCmd.Context())
}

func TestBundleConfigureDefault(t *testing.T) {
	b := setup(t, "https://x.com")
	assert.NotPanics(t, func() {
		b.WorkspaceClient()
	})
}

func TestBundleConfigureWithMultipleMatches(t *testing.T) {
	b := setup(t, "https://a.com")
	assert.Panics(t, func() {
		b.WorkspaceClient()
	})
}

func TestBundleConfigureWithNonExistentProfileFlag(t *testing.T) {
	RootCmd.Flag("profile").Value.Set("NOEXIST")

	b := setup(t, "https://x.com")
	assert.PanicsWithError(t, "no matching config profiles found", func() {
		b.WorkspaceClient()
	})
}

func TestBundleConfigureWithMismatchedProfile(t *testing.T) {
	RootCmd.Flag("profile").Value.Set("PROFILE-1")

	b := setup(t, "https://x.com")
	assert.PanicsWithError(t, "config host mismatch: profile uses host https://a.com, but CLI configured to use https://x.com", func() {
		b.WorkspaceClient()
	})
}

func TestBundleConfigureWithCorrectProfile(t *testing.T) {
	RootCmd.Flag("profile").Value.Set("PROFILE-1")

	b := setup(t, "https://a.com")
	assert.NotPanics(t, func() {
		b.WorkspaceClient()
	})
}

func TestBundleConfigureWithMismatchedProfileEnvVariable(t *testing.T) {
	t.Setenv("DATABRICKS_CONFIG_PROFILE", "PROFILE-1")
	t.Cleanup(func() {
		t.Setenv("DATABRICKS_CONFIG_PROFILE", "")
	})

	b := setup(t, "https://x.com")
	assert.PanicsWithError(t, "config host mismatch: profile uses host https://a.com, but CLI configured to use https://x.com", func() {
		b.WorkspaceClient()
	})
}

func TestBundleConfigureWithProfileFlagAndEnvVariable(t *testing.T) {
	t.Setenv("DATABRICKS_CONFIG_PROFILE", "NOEXIST")
	t.Cleanup(func() {
		t.Setenv("DATABRICKS_CONFIG_PROFILE", "")
	})
	RootCmd.Flag("profile").Value.Set("PROFILE-1")

	b := setup(t, "https://a.com")
	assert.NotPanics(t, func() {
		b.WorkspaceClient()
	})
}

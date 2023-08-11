package root

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/spf13/cobra"
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

func emptyCommand(t *testing.T) *cobra.Command {
	ctx := context.Background()
	cmd := &cobra.Command{}
	cmd.SetContext(ctx)
	initProfileFlag(cmd)
	return cmd
}

func setup(t *testing.T, cmd *cobra.Command, host string) *bundle.Bundle {
	setupDatabricksCfg(t)

	err := configureBundle(cmd, []string{"validate"}, func(_ context.Context) (*bundle.Bundle, error) {
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
	return bundle.Get(cmd.Context())
}

func TestBundleConfigureDefault(t *testing.T) {
	cmd := emptyCommand(t)
	b := setup(t, cmd, "https://x.com")
	assert.NotPanics(t, func() {
		b.WorkspaceClient()
	})
}

func TestBundleConfigureWithMultipleMatches(t *testing.T) {
	cmd := emptyCommand(t)
	b := setup(t, cmd, "https://a.com")
	assert.Panics(t, func() {
		b.WorkspaceClient()
	})
}

func TestBundleConfigureWithNonExistentProfileFlag(t *testing.T) {
	cmd := emptyCommand(t)
	cmd.Flag("profile").Value.Set("NOEXIST")

	b := setup(t, cmd, "https://x.com")
	assert.PanicsWithError(t, "no matching config profiles found", func() {
		b.WorkspaceClient()
	})
}

func TestBundleConfigureWithMismatchedProfile(t *testing.T) {
	cmd := emptyCommand(t)
	cmd.Flag("profile").Value.Set("PROFILE-1")

	b := setup(t, cmd, "https://x.com")
	assert.PanicsWithError(t, "config host mismatch: profile uses host https://a.com, but CLI configured to use https://x.com", func() {
		b.WorkspaceClient()
	})
}

func TestBundleConfigureWithCorrectProfile(t *testing.T) {
	cmd := emptyCommand(t)
	cmd.Flag("profile").Value.Set("PROFILE-1")

	b := setup(t, cmd, "https://a.com")
	assert.NotPanics(t, func() {
		b.WorkspaceClient()
	})
}

func TestBundleConfigureWithMismatchedProfileEnvVariable(t *testing.T) {
	t.Setenv("DATABRICKS_CONFIG_PROFILE", "PROFILE-1")
	t.Cleanup(func() {
		t.Setenv("DATABRICKS_CONFIG_PROFILE", "")
	})

	cmd := emptyCommand(t)
	b := setup(t, cmd, "https://x.com")
	assert.PanicsWithError(t, "config host mismatch: profile uses host https://a.com, but CLI configured to use https://x.com", func() {
		b.WorkspaceClient()
	})
}

func TestBundleConfigureWithProfileFlagAndEnvVariable(t *testing.T) {
	t.Setenv("DATABRICKS_CONFIG_PROFILE", "NOEXIST")
	t.Cleanup(func() {
		t.Setenv("DATABRICKS_CONFIG_PROFILE", "")
	})

	cmd := emptyCommand(t)
	cmd.Flag("profile").Value.Set("PROFILE-1")

	b := setup(t, cmd, "https://a.com")
	assert.NotPanics(t, func() {
		b.WorkspaceClient()
	})
}

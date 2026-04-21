package ucm

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/databricks/cli/ucm/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupDatabricksCfg writes a .databrickscfg with a mix of profiles into a
// temp HOME and points DATABRICKS_CONFIG_FILE at the default location so the
// SDK's profile loader picks it up. Mirrors cmd/root/bundle_test.go.
func setupDatabricksCfg(t *testing.T) {
	t.Helper()
	tempHomeDir := t.TempDir()
	homeEnvVar := "HOME"
	if runtime.GOOS == "windows" {
		homeEnvVar = "USERPROFILE"
	}

	cfg := []byte(strings.Join([]string{
		"[PROFILE-UNIQUE]",
		"host = https://unique.example.com",
		"token = u",
		"",
		"[PROFILE-DUP-1]",
		"host = https://dup.example.com",
		"token = d1",
		"",
		"[PROFILE-DUP-2]",
		"host = https://dup.example.com",
		"token = d2",
		"",
	}, "\n"))
	err := os.WriteFile(filepath.Join(tempHomeDir, ".databrickscfg"), cfg, 0o644)
	require.NoError(t, err)

	t.Setenv("DATABRICKS_CONFIG_FILE", "")
	t.Setenv(homeEnvVar, tempHomeDir)
}

func TestWorkspaceClientE_UniqueMatchResolvesProfile(t *testing.T) {
	testutil.CleanupEnvironment(t)
	setupDatabricksCfg(t)

	u := &Ucm{
		Config: config.Root{
			Workspace: config.Workspace{
				Host: "https://unique.example.com",
			},
		},
	}

	client, err := u.WorkspaceClientE()
	require.NoError(t, err)
	assert.Equal(t, "PROFILE-UNIQUE", client.Config.Profile)
	assert.Equal(t, "https://unique.example.com", client.Config.Host)
}

func TestWorkspaceClientE_MultipleMatchesReturnsAmbiguity(t *testing.T) {
	testutil.CleanupEnvironment(t)
	setupDatabricksCfg(t)

	u := &Ucm{
		Config: config.Root{
			Workspace: config.Workspace{
				Host: "https://dup.example.com",
			},
		},
	}

	_, err := u.WorkspaceClientE()
	require.Error(t, err)
	names, ok := databrickscfg.AsMultipleProfiles(err)
	require.True(t, ok, "expected AsMultipleProfiles to detect ambiguity in: %v", err)
	assert.ElementsMatch(t, []string{"PROFILE-DUP-1", "PROFILE-DUP-2"}, names)
}

func TestWorkspaceClientE_NoMatchingHost(t *testing.T) {
	testutil.CleanupEnvironment(t)
	setupDatabricksCfg(t)

	u := &Ucm{
		Config: config.Root{
			Workspace: config.Workspace{
				Host: "https://nobody.example.com",
			},
		},
	}

	client, err := u.WorkspaceClientE()
	// The SDK's ResolveProfileFromHost loader swallows the "no matching
	// profile" case and returns nil, so subsequent auth depends on env/default
	// credential chain. What we assert here is that the result is NOT an
	// ambiguity error — the user must not be dropped into a profile picker.
	if err != nil {
		_, isMulti := databrickscfg.AsMultipleProfiles(err)
		assert.False(t, isMulti, "unexpected ambiguity error for no-match host: %v", err)
		return
	}
	// If no error, the host should still match what we configured and profile
	// must stay empty (no stealth selection).
	assert.Equal(t, "https://nobody.example.com", client.Config.Host)
	assert.Empty(t, client.Config.Profile)
}

func TestWorkspaceClientE_Memoizes(t *testing.T) {
	testutil.CleanupEnvironment(t)
	setupDatabricksCfg(t)

	u := &Ucm{
		Config: config.Root{
			Workspace: config.Workspace{
				Host: "https://unique.example.com",
			},
		},
	}

	c1, err := u.WorkspaceClientE()
	require.NoError(t, err)
	c2, err := u.WorkspaceClientE()
	require.NoError(t, err)
	assert.Same(t, c1, c2, "expected WorkspaceClientE to memoize")
}

func TestWorkspaceClientE_ClearReResolves(t *testing.T) {
	testutil.CleanupEnvironment(t)
	setupDatabricksCfg(t)

	u := &Ucm{
		Config: config.Root{
			Workspace: config.Workspace{
				Host: "https://unique.example.com",
			},
		},
	}

	c1, err := u.WorkspaceClientE()
	require.NoError(t, err)

	u.ClearWorkspaceClient()
	c2, err := u.WorkspaceClientE()
	require.NoError(t, err)
	assert.NotSame(t, c1, c2, "expected ClearWorkspaceClient to reset memoization")
}

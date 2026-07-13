package versioncheck

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/databricks/cli/internal/build"
	"github.com/stretchr/testify/assert"
)

func TestDetectInstallMethod(t *testing.T) {
	tests := []struct {
		name       string
		goos       string
		execPath   string
		wantMethod InstallMethod
		wantCmd    string
	}{
		{"homebrew apple silicon", "darwin", "/opt/homebrew/Cellar/databricks/0.240.0/bin/databricks", InstallHomebrew, upgradeHomebrew},
		{"homebrew intel", "darwin", "/usr/local/Cellar/databricks/0.240.0/bin/databricks", InstallHomebrew, upgradeHomebrew},
		{"homebrew linux", "linux", "/home/linuxbrew/.linuxbrew/Cellar/databricks/0.240.0/bin/databricks", InstallHomebrew, upgradeHomebrew},
		{"script unix", "linux", "/usr/local/bin/databricks", InstallScript, upgradeScript},
		{"script macos", "darwin", "/usr/local/bin/databricks", InstallScript, upgradeScript},
		{"unknown unix", "linux", "/home/me/bin/databricks", InstallUnknown, ""},
		{"winget", "windows", `C:\Users\me\AppData\Local\Microsoft\WinGet\Packages\Databricks.DatabricksCLI_X\databricks.exe`, InstallWinget, upgradeWinget},
		{"chocolatey", "windows", `C:\ProgramData\chocolatey\bin\databricks.exe`, InstallChocolatey, upgradeChocolatey},
		{"script windows", "windows", `C:\Windows\databricks.exe`, InstallScript, upgradeScript},
		{"unknown windows", "windows", `C:\tools\databricks.exe`, InstallUnknown, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			method, cmd := detectInstallMethod(tt.goos, tt.execPath)
			assert.Equal(t, tt.wantMethod, method)
			assert.Equal(t, tt.wantCmd, cmd)
		})
	}
}

func TestIsNewer(t *testing.T) {
	tests := []struct {
		current string
		latest  string
		want    bool
	}{
		{"0.240.0", "0.245.0", true},
		{"0.245.0", "0.245.0", false},
		{"0.245.0", "0.240.0", false},
		{"0.245.0", "1.0.0", true},
		{"1.0.0", "0.245.0", false},
		{"0.240.0", "not-a-version", false},
	}

	for _, tt := range tests {
		t.Run(tt.current+"_vs_"+tt.latest, func(t *testing.T) {
			assert.Equal(t, tt.want, isNewer(tt.current, tt.latest))
		})
	}
}

func TestCheck(t *testing.T) {
	// Check reads the running build version via package-global state; restore it
	// so the mutation doesn't leak into other tests in this binary.
	original := build.GetInfo().Version
	t.Cleanup(func() { build.SetBuildVersion(original) })

	newReleaseServer := func(t *testing.T, status int, tag string) *httptest.Server {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, latestReleasePath, r.URL.Path)
			w.WriteHeader(status)
			if tag != "" {
				_, _ = w.Write([]byte(`{"tag_name":"` + tag + `"}`))
			}
		}))
		t.Cleanup(srv.Close)
		return srv
	}

	t.Run("update available", func(t *testing.T) {
		build.SetBuildVersion("0.240.0")
		srv := newReleaseServer(t, http.StatusOK, "v0.245.0")
		t.Setenv(gitHubAPIURLEnv, srv.URL)

		result := Check(t.Context())
		assert.Equal(t, "0.240.0", result.CurrentVersion)
		assert.Equal(t, "0.245.0", result.LatestVersion)
		assert.True(t, result.UpdateAvailable)
		assert.False(t, result.DevelopmentBuild)
	})

	t.Run("up to date", func(t *testing.T) {
		build.SetBuildVersion("0.245.0")
		srv := newReleaseServer(t, http.StatusOK, "v0.245.0")
		t.Setenv(gitHubAPIURLEnv, srv.URL)

		result := Check(t.Context())
		assert.False(t, result.UpdateAvailable)
		assert.Equal(t, "0.245.0", result.LatestVersion)
	})

	t.Run("development build skips the network call", func(t *testing.T) {
		build.SetBuildVersion("0.0.0-dev+abc123")
		// No server env set: a network call here would fail, proving we skip it.
		result := Check(t.Context())
		assert.True(t, result.DevelopmentBuild)
		assert.False(t, result.UpdateAvailable)
		assert.Empty(t, result.LatestVersion)
	})

	t.Run("server error fails gently", func(t *testing.T) {
		build.SetBuildVersion("0.240.0")
		srv := newReleaseServer(t, http.StatusInternalServerError, "")
		t.Setenv(gitHubAPIURLEnv, srv.URL)

		result := Check(t.Context())
		assert.True(t, result.CheckFailed)
		assert.False(t, result.UpdateAvailable)
		assert.Empty(t, result.LatestVersion)
	})

	t.Run("unreachable GitHub fails gently", func(t *testing.T) {
		build.SetBuildVersion("0.240.0")
		// A server closed immediately gives a definitely-refused port.
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		url := srv.URL
		srv.Close()
		t.Setenv(gitHubAPIURLEnv, url)

		result := Check(t.Context())
		assert.True(t, result.CheckFailed)
	})
}

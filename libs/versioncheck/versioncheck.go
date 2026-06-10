// Package versioncheck reports whether a newer Databricks CLI release is
// available and, based on how the running binary was installed, how to upgrade.
package versioncheck

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
	"golang.org/x/mod/semver"
)

const (
	// GitHub returns the latest non-prerelease, non-draft release here, which is
	// exactly the "latest stable" the CLI ships from.
	// https://docs.github.com/en/rest/releases/releases#get-the-latest-release
	defaultGitHubAPIURL = "https://api.github.com"
	latestReleasePath   = "/repos/databricks/cli/releases/latest"

	// gitHubAPIURLEnv overrides the GitHub API base URL. It lets acceptance
	// tests point the lookup at the mock server (and gives airgapped setups an
	// escape hatch); it is not a documented, user-facing setting.
	gitHubAPIURLEnv = "DATABRICKS_CLI_GITHUB_API_URL"

	// fetchTimeout bounds the release lookup. The explicit `version --check`
	// waits on it, so keep it short: a quick "couldn't reach GitHub" beats a hang.
	fetchTimeout = 2 * time.Second
)

// Upgrade commands per install method, matching the documented install flows:
// https://docs.databricks.com/dev-tools/cli/install.html
const (
	upgradeHomebrew   = "brew upgrade databricks"
	upgradeWinget     = "winget upgrade Databricks.DatabricksCLI"
	upgradeChocolatey = "choco upgrade databricks-cli"
	upgradeScript     = "curl -fsSL https://raw.githubusercontent.com/databricks/setup-cli/main/install.sh | sh"
)

// InstallMethod identifies how the running binary was installed, which
// determines the command a user runs to upgrade.
type InstallMethod string

const (
	InstallHomebrew   InstallMethod = "homebrew"
	InstallWinget     InstallMethod = "winget"
	InstallChocolatey InstallMethod = "chocolatey"
	InstallScript     InstallMethod = "script" // the curl | sh installer
	InstallUnknown    InstallMethod = "unknown"
)

// Result is the outcome of an update check.
type Result struct {
	CurrentVersion   string `json:"current_version"`
	LatestVersion    string `json:"latest_version,omitempty"`
	UpdateAvailable  bool   `json:"update_available"`
	DevelopmentBuild bool   `json:"development_build,omitempty"`
	// CheckFailed is set when the latest version couldn't be fetched (offline,
	// timeout, rate-limited). The command reports this gently instead of erroring.
	CheckFailed    bool          `json:"check_failed,omitempty"`
	InstallMethod  InstallMethod `json:"install_method,omitempty"`
	UpgradeCommand string        `json:"upgrade_command,omitempty"`
}

// Check fetches the latest released version and compares it with the running
// build, reporting how to upgrade based on the detected install method. It
// never fails: lookup errors are reported via Result.CheckFailed.
//
// Development and snapshot builds have no meaningful released version to
// compare against, so they short-circuit without a network call.
func Check(ctx context.Context) *Result {
	info := build.GetInfo()
	if isDevelopmentBuild(info) {
		return &Result{
			CurrentVersion:   info.Version,
			DevelopmentBuild: true,
		}
	}

	method, command := DetectInstallMethod(ctx)

	ctx, cancel := context.WithTimeout(ctx, fetchTimeout)
	defer cancel()
	latest, err := fetchLatestVersion(ctx)
	if err != nil {
		// Fail gently: a failed lookup shouldn't fail the command. The caller
		// renders a "couldn't reach GitHub" message instead.
		log.Debugf(ctx, "version check: %v", err)
		return &Result{
			CurrentVersion: info.Version,
			CheckFailed:    true,
		}
	}

	return &Result{
		CurrentVersion:  info.Version,
		LatestVersion:   latest,
		UpdateAvailable: isNewer(info.Version, latest),
		InstallMethod:   method,
		UpgradeCommand:  command,
	}
}

// isDevelopmentBuild reports whether the binary was not built from a tagged
// release. Snapshot builds (goreleaser --snapshot) and local `go build`
// binaries (version 0.0.0-dev+<sha>) fall into this category.
func isDevelopmentBuild(info build.Info) bool {
	return info.IsSnapshot || strings.HasPrefix(info.Version, "0.0.0")
}

// isNewer reports whether latest is a higher semver than current. Both are
// bare versions without a leading "v".
func isNewer(current, latest string) bool {
	cv := "v" + current
	lv := "v" + latest
	if !semver.IsValid(cv) || !semver.IsValid(lv) {
		return false
	}
	return semver.Compare(lv, cv) > 0
}

func fetchLatestVersion(ctx context.Context) (string, error) {
	base := env.Get(ctx, gitHubAPIURLEnv)
	if base == "" {
		base = defaultGitHubAPIURL
	}

	url := base + latestReleasePath
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	// GitHub rejects requests without a User-Agent and recommends pinning the
	// API version. https://docs.github.com/en/rest/using-the-rest-api
	req.Header.Set("User-Agent", "databricks-cli/"+build.GetInfo().Version)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetch latest release: unexpected status %s", resp.Status)
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", fmt.Errorf("parse latest release: %w", err)
	}

	version := strings.TrimPrefix(release.TagName, "v")
	if version == "" {
		return "", errors.New("latest release has an empty tag_name")
	}
	return version, nil
}

// DetectInstallMethod inspects the running executable's path to infer how the
// CLI was installed and the command to upgrade it. It returns InstallUnknown
// with an empty command when the install method can't be determined.
func DetectInstallMethod(ctx context.Context) (InstallMethod, string) {
	exe, err := os.Executable()
	if err != nil {
		log.Debugf(ctx, "version check: cannot determine executable path: %v", err)
		return InstallUnknown, ""
	}
	// Resolve symlinks so a Homebrew shim (e.g. /usr/local/bin/databricks ->
	// ../Cellar/...) is classified by its real location rather than the shim.
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return detectInstallMethod(runtime.GOOS, exe)
}

func detectInstallMethod(goos, execPath string) (InstallMethod, string) {
	// Normalize separators independently of the host OS (not filepath.ToSlash,
	// which only swaps the host's separator) so the goos parameter fully
	// determines classification and the logic is testable cross-platform.
	p := strings.ReplaceAll(execPath, `\`, "/")
	if goos == "windows" {
		lower := strings.ToLower(p)
		switch {
		case strings.Contains(lower, "/winget/"):
			return InstallWinget, upgradeWinget
		case strings.Contains(lower, "/chocolatey/"):
			return InstallChocolatey, upgradeChocolatey
		case lower == "c:/windows/databricks.exe":
			return InstallScript, upgradeScript
		}
		return InstallUnknown, ""
	}

	// macOS and Linux.
	switch {
	case strings.Contains(p, "/Cellar/"), strings.Contains(p, "/homebrew/"), strings.Contains(p, "/linuxbrew/"):
		return InstallHomebrew, upgradeHomebrew
	case p == "/usr/local/bin/databricks":
		return InstallScript, upgradeScript
	}
	return InstallUnknown, ""
}

// Package upgradecheck implements a best-effort "your CLI is outdated" check.
//
// The design goal is that the check must never add latency to a command. Every
// invocation reads the latest known version from an on-disk cache (a synchronous
// local read, no network). The GitHub API is only contacted by [Refresh], which
// the caller runs in a background goroutine when the cache is [Stale]. The
// upgrade notice is always rendered from the cache, so even the run that
// refreshes it does not block on the network.
package upgradecheck

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/mod/semver"
)

const (
	// checkInterval is the cache TTL: the GitHub API is refreshed at most once
	// per day. Between refreshes the latest version is served from the on-disk
	// cache, so the common path performs no network I/O.
	checkInterval = 24 * time.Hour

	// fetchTimeout bounds the background refresh so it can never run unbounded.
	fetchTimeout = 3 * time.Second

	// defaultBaseURL is the GitHub REST API root. Overridable in tests via [WithBaseURL].
	defaultBaseURL = "https://api.github.com"

	// releasePath is the "latest release" endpoint for the CLI repository.
	releasePath = "/repos/databricks/cli/releases/latest"

	// Cache file is readable and writable by the owner only.
	cacheFilePerm = 0o600
	cacheDirPerm  = 0o755
)

// baseURLKey is the context key used to override the GitHub API root in tests.
type baseURLKey struct{}

// WithBaseURL overrides the GitHub API root used by [Refresh]. Intended for tests.
func WithBaseURL(ctx context.Context, url string) context.Context {
	return context.WithValue(ctx, baseURLKey{}, url)
}

func baseURL(ctx context.Context) string {
	if v, ok := ctx.Value(baseURLKey{}).(string); ok {
		return v
	}
	return defaultBaseURL
}

// release mirrors the subset of the GitHub release payload that we use.
type release struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
}

// cacheEntry is the on-disk record stored at the cache file path.
type cacheEntry struct {
	CheckedAt     time.Time `json:"checked_at"`
	LatestVersion string    `json:"latest_version"`
	LatestURL     string    `json:"latest_url"`
}

// Stale reports whether the cache should be refreshed: it is missing, unreadable,
// or older than [checkInterval]. An unreadable cache is treated as stale so a
// corrupt file self-heals on the next refresh.
func Stale(cacheFile string, now time.Time) bool {
	entry, ok, err := readCache(cacheFile)
	if err != nil || !ok {
		return true
	}
	return now.Sub(entry.CheckedAt) > checkInterval
}

// Refresh fetches the latest release from GitHub and writes it to the cache file.
// It is meant to be called from a background goroutine; the write is atomic so a
// cancelled or short-lived process can never leave a corrupt cache behind.
func Refresh(ctx context.Context, cacheFile string) error {
	ctx, cancel := context.WithTimeout(ctx, fetchTimeout)
	defer cancel()

	rel, err := fetchLatest(ctx)
	if err != nil {
		return err
	}

	return writeCache(cacheFile, cacheEntry{
		CheckedAt:     time.Now(),
		LatestVersion: rel.TagName,
		LatestURL:     rel.HTMLURL,
	})
}

// Outdated reports whether the cached latest version is newer than currentVersion.
// It returns the latest version and its release URL when an upgrade is available.
func Outdated(cacheFile, currentVersion string) (latestVersion, latestURL string, ok bool) {
	entry, found, err := readCache(cacheFile)
	if err != nil || !found {
		return "", "", false
	}
	if !isNewer(entry.LatestVersion, currentVersion) {
		return "", "", false
	}
	return entry.LatestVersion, entry.LatestURL, true
}

func fetchLatest(ctx context.Context) (release, error) {
	url := baseURL(ctx) + releasePath
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return release{}, err
	}
	// Recommended by GitHub for REST API requests.
	// https://docs.github.com/en/rest/using-the-rest-api/getting-started-with-the-rest-api#about-the-rest-api
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return release{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return release{}, fmt.Errorf("github request failed: %s", resp.Status)
	}

	var rel release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return release{}, fmt.Errorf("decode github response: %w", err)
	}
	return rel, nil
}

func readCache(cacheFile string) (cacheEntry, bool, error) {
	raw, err := os.ReadFile(cacheFile)
	if errors.Is(err, fs.ErrNotExist) {
		return cacheEntry{}, false, nil
	}
	if err != nil {
		return cacheEntry{}, false, err
	}
	var entry cacheEntry
	if err := json.Unmarshal(raw, &entry); err != nil {
		return cacheEntry{}, false, err
	}
	return entry, true, nil
}

// writeCache writes the cache entry atomically: it writes to a temporary file in
// the same directory and renames it into place, so a concurrent reader never
// observes a partially written file.
func writeCache(cacheFile string, entry cacheEntry) error {
	raw, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(cacheFile)
	if err := os.MkdirAll(dir, cacheDirPerm); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, filepath.Base(cacheFile)+".*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op once the rename succeeds

	if _, err := tmp.Write(raw); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpName, cacheFilePerm); err != nil {
		return err
	}
	return os.Rename(tmpName, cacheFile)
}

// IsReleaseVersion reports whether version is a published release: valid semver
// with no prerelease segment. Dev builds and goreleaser snapshots carry a "-dev"
// prerelease (e.g. "0.0.0-dev", "0.230.1-dev+abc123"), so they return false —
// there is no newer release to point such a build at.
func IsReleaseVersion(version string) bool {
	v := normalize(version)
	return semver.IsValid(v) && semver.Prerelease(v) == ""
}

// isNewer reports whether latest is a strictly higher semantic version than
// current. Build versions have no "v" prefix while GitHub tags do, so both are
// normalized before comparison. Unparseable versions are treated as not newer.
func isNewer(latest, current string) bool {
	l := normalize(latest)
	c := normalize(current)
	if !semver.IsValid(l) || !semver.IsValid(c) {
		return false
	}
	return semver.Compare(l, c) > 0
}

func normalize(v string) string {
	if v == "" || strings.HasPrefix(v, "v") {
		return v
	}
	return "v" + v
}

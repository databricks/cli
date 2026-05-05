package clicompat

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
	"golang.org/x/mod/semver"
)

const (
	// manifestURL is the raw GitHub URL for the compatibility manifest.
	manifestURL = "https://raw.githubusercontent.com/databricks/cli/main/internal/build/cli-compat.json"

	// fetchTimeout is the HTTP timeout for fetching the manifest at runtime.
	fetchTimeout = 3 * time.Second

	// nextKey is the special manifest key for CLI versions newer than any entry.
	nextKey = "next"

	// cacheTTL is how long a locally cached manifest is considered fresh.
	cacheTTL = 1 * time.Hour

	// localManifestFile is the filename for the locally cached manifest.
	localManifestFile = "compat-manifest.json"

	// devVersionPrefix identifies dev builds that always use the "next" entry.
	devVersionPrefix = "0.0.0-dev"

	fetchRetries      = 2
	fetchRetryBackoff = 300 * time.Millisecond
)

// Entry maps a CLI version to compatible AppKit and Agent Skills versions.
type Entry struct {
	AppKit      string `json:"appkit"`
	AgentSkills string `json:"skills"`
}

// Manifest is the compatibility manifest: a map of CLI version strings to entries.
type Manifest map[string]Entry

// cachedManifest holds a parsed manifest together with its on-disk mod time.
type cachedManifest struct {
	manifest Manifest
	modTime  time.Time
}

// isFresh reports whether the cached manifest is younger than maxAge.
func (c cachedManifest) isFresh(maxAge time.Duration) bool {
	return time.Since(c.modTime) < maxAge
}

// httpClient is the HTTP client used for manifest fetches. Package-level var
// so tests can replace it.
var httpClient = &http.Client{Timeout: fetchTimeout}

// FetchManifest returns the compatibility manifest using a 4-tier fallback:
//  1. Local cached file (if fresh, < 1 hour old)
//  2. Remote fetch from GitHub (with retry)
//  3. Stale local file (if remote fails but a previously cached file exists)
//  4. Embedded manifest compiled into the binary
func FetchManifest(ctx context.Context) (Manifest, error) {
	localPath := manifestLocalPath(ctx)

	// Read local file once — reuse across tiers.
	local, localErr := readLocalManifest(localPath)

	// Tier 1: local file is fresh.
	if localErr == nil && local.isFresh(cacheTTL) {
		log.Debugf(ctx, "Using cached manifest from %s", localPath)
		return local.manifest, nil
	}

	// Tier 2: fetch from remote (local file missing or stale).
	m, fetchErr := fetchRemoteWithRetry(ctx)
	if fetchErr == nil {
		writeLocalManifest(ctx, localPath, m)
		return m, nil
	}

	// Tier 3a: local file exists but stale — use it anyway.
	if localErr == nil {
		log.Debugf(ctx, "Using stale cached manifest (remote failed: %v)", fetchErr)
		return local.manifest, nil
	}

	// Tier 3b: embedded manifest.
	m, embeddedErr := parseManifest(build.CLICompatManifestJSON)
	if embeddedErr == nil {
		log.Debugf(ctx, "Using embedded manifest (remote and local cache failed)")
		return m, nil
	}

	return nil, fmt.Errorf("all manifest sources failed (remote: %w, embedded: %w)", fetchErr, embeddedErr)
}

// EmbeddedDefaultAppKitVersion returns the "next" entry's AppKit version from
// the embedded manifest. Used for help text defaults where a network call is
// not appropriate. Returns "" if the embedded manifest is invalid.
func EmbeddedDefaultAppKitVersion() string {
	m, err := parseManifest(build.CLICompatManifestJSON)
	if err != nil {
		return ""
	}
	if next, ok := m[nextKey]; ok {
		return next.AppKit
	}
	return ""
}

// Resolve returns the manifest entry for the given CLI version.
//
// Resolution order:
//  1. Dev builds (version starts with "0.0.0-dev") use the "next" entry.
//  2. Exact match on CLI version.
//  3. Nearest lower version (semver-sorted). This also handles CLI versions
//     newer than all entries, returning the highest known entry.
//  4. If CLI is older than all entries, use the lowest (oldest) entry.
func Resolve(m Manifest, cliVersion string) (Entry, error) {
	if len(m) == 0 {
		return Entry{}, errors.New("empty compatibility manifest")
	}

	next, ok := m[nextKey]
	if !ok {
		return Entry{}, fmt.Errorf("compatibility manifest missing %q key", nextKey)
	}

	// Dev builds always use "next".
	if strings.HasPrefix(cliVersion, devVersionPrefix) {
		return next, nil
	}

	// Exact match.
	if entry, ok := m[cliVersion]; ok {
		return entry, nil
	}

	// Collect and sort versioned keys (exclude "next").
	var versions []string
	for k := range m {
		if k != nextKey {
			versions = append(versions, k)
		}
	}

	// Sort descending by semver. The semver package requires a "v" prefix.
	slices.SortFunc(versions, func(a, b string) int {
		return semver.Compare("v"+b, "v"+a)
	})

	// Find the nearest lower version.
	vCLI := "v" + cliVersion
	for _, v := range versions {
		if semver.Compare("v"+v, vCLI) <= 0 {
			return m[v], nil
		}
	}

	// CLI is older than all entries — use the lowest (oldest) entry as closest match.
	// If there are no versioned entries (only "next"), fall back to "next".
	if len(versions) == 0 {
		return next, nil
	}
	return m[versions[len(versions)-1]], nil
}

// resolveEntry fetches the manifest, resolves for the given CLI version.
func resolveEntry(ctx context.Context, cliVersion string) (Entry, error) {
	m, err := FetchManifest(ctx)
	if err != nil {
		return Entry{}, err
	}
	return Resolve(m, cliVersion)
}

// ResolveAppKitVersion resolves the AppKit template version for the current CLI.
func ResolveAppKitVersion(ctx context.Context) (string, error) {
	entry, err := resolveEntry(ctx, build.GetInfo().Version)
	if err != nil {
		return "", err
	}
	return entry.AppKit, nil
}

// ResolveAgentSkillsVersion resolves the Agent Skills version for the current CLI.
func ResolveAgentSkillsVersion(ctx context.Context) (string, error) {
	entry, err := resolveEntry(ctx, build.GetInfo().Version)
	if err != nil {
		return "", err
	}
	return entry.AgentSkills, nil
}

// --- Local manifest cache ---

// manifestLocalPath returns the path to the locally cached manifest file.
func manifestLocalPath(ctx context.Context) string {
	if dir := env.Get(ctx, "DATABRICKS_CACHE_DIR"); dir != "" {
		return filepath.Join(dir, localManifestFile)
	}
	home, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, "databricks", localManifestFile)
}

// readLocalManifest reads and parses the locally cached manifest file.
func readLocalManifest(path string) (cachedManifest, error) {
	if path == "" {
		return cachedManifest{}, errors.New("no cache path")
	}
	info, err := os.Stat(path)
	if err != nil {
		return cachedManifest{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return cachedManifest{}, err
	}
	m, err := parseManifest(data)
	if err != nil {
		return cachedManifest{}, err
	}
	return cachedManifest{manifest: m, modTime: info.ModTime()}, nil
}

// writeLocalManifest writes the manifest to the local cache path using a
// temp-file-then-rename pattern. The os.Remove before os.Rename is needed
// for Windows, where Rename fails if the destination exists.
func writeLocalManifest(ctx context.Context, path string, m Manifest) {
	if path == "" {
		return
	}
	data, err := json.Marshal(m)
	if err != nil {
		log.Debugf(ctx, "Failed to marshal manifest for cache: %v", err)
		return
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		log.Debugf(ctx, "Failed to create cache directory %s: %v", dir, err)
		return
	}
	tmp, err := os.CreateTemp(dir, ".compat-manifest-*.tmp")
	if err != nil {
		log.Debugf(ctx, "Failed to create temp cache file: %v", err)
		return
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}()
	if _, err := tmp.Write(data); err != nil {
		log.Debugf(ctx, "Failed to write temp cache file: %v", err)
		return
	}
	if err := tmp.Close(); err != nil {
		log.Debugf(ctx, "Failed to close temp cache file: %v", err)
		return
	}
	_ = os.Remove(path)
	if err := os.Rename(tmpPath, path); err != nil {
		log.Debugf(ctx, "Failed to rename temp cache file: %v", err)
	}
}

// --- Remote fetch ---

// fetchRemoteWithRetry wraps fetchRemote with retries.
func fetchRemoteWithRetry(ctx context.Context) (Manifest, error) {
	var lastErr error
	for attempt := range fetchRetries + 1 {
		if attempt > 0 {
			log.Debugf(ctx, "Retrying manifest fetch (attempt %d)", attempt+1)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(fetchRetryBackoff):
			}
		}
		m, err := fetchRemote(ctx)
		if err == nil {
			return m, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

func fetchRemote(ctx context.Context) (Manifest, error) {
	log.Debugf(ctx, "Fetching compatibility manifest from %s", manifestURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, manifestURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d fetching manifest", resp.StatusCode)
	}

	// Cap response size to guard against corrupted or malicious responses.
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}

	return parseManifest(body)
}

func parseManifest(data []byte) (Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("invalid manifest JSON: %w", err)
	}
	if len(m) == 0 {
		return nil, errors.New("empty compatibility manifest")
	}
	if _, ok := m[nextKey]; !ok {
		return nil, fmt.Errorf("compatibility manifest missing %q key", nextKey)
	}
	for k := range m {
		if k != nextKey && !semver.IsValid("v"+k) {
			return nil, fmt.Errorf("invalid semver key %q in compatibility manifest", k)
		}
	}
	return m, nil
}

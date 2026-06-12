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
	"sync"
	"time"

	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
	"golang.org/x/mod/semver"
)

// ErrNotFound indicates that a requested resource (tag, branch, manifest)
// does not exist at the remote.
var ErrNotFound = errors.New("not found")

// HTTPStatusError captures a non-200 HTTP status code from a manifest fetch.
type HTTPStatusError struct {
	StatusCode int
}

// Error implements the error interface.
func (e *HTTPStatusError) Error() string {
	return fmt.Sprintf("HTTP %d fetching manifest", e.StatusCode)
}

const (
	// manifestURL is the raw GitHub URL for the compatibility manifest.
	manifestURL = "https://raw.githubusercontent.com/databricks/cli/main/internal/build/cli-compat.json"

	// fetchTimeout is the HTTP timeout for fetching the manifest at runtime.
	fetchTimeout = 3 * time.Second

	// cacheTTL is how long a locally cached manifest is considered fresh.
	cacheTTL = 1 * time.Hour

	// localManifestFile is the filename for the locally cached manifest.
	localManifestFile = "compat-manifest.json"

	// devVersionPrefix identifies dev builds whose semver (0.0.0) is lower than
	// all real CLI versions. These are treated as bleeding-edge and resolve to
	// the highest versioned entry.
	devVersionPrefix = "0.0.0-dev"

	maxFetchAttempts  = 3
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
// so tests can replace it. Not safe for parallel tests; the clicompat test
// suite does not use t.Parallel().
var httpClient = &http.Client{Timeout: fetchTimeout}

// FetchManifest returns the compatibility manifest using a 4-tier fallback:
//  1. Local cached file (if fresh, < 1 hour old)
//  2. Remote fetch from GitHub (with retry)
//  3. Stale local file (if remote fails but a previously cached file exists)
//  4. Embedded manifest compiled into the binary
//
// Set DATABRICKS_FORCE_EMBEDDED_COMPAT=true to skip all tiers and use only
// the embedded manifest. Useful for local development when testing with a
// locally compiled binary.
//
// Set DATABRICKS_CACHE_ENABLED=false to bypass the local cache (tiers 1 and 3a),
// which is useful to recover from a bad cached manifest.
func FetchManifest(ctx context.Context) (Manifest, error) {
	if force, ok := env.GetBool(ctx, "DATABRICKS_FORCE_EMBEDDED_COMPAT"); ok && force {
		log.Debugf(ctx, "Using embedded manifest (DATABRICKS_FORCE_EMBEDDED_COMPAT=true)")
		return parseEmbeddedManifest()
	}

	cacheEnabled := true
	if enabled, ok := env.GetBool(ctx, "DATABRICKS_CACHE_ENABLED"); ok {
		cacheEnabled = enabled
	}

	localPath := manifestLocalPath(ctx)

	// Read local file once — reuse across tiers.
	var local cachedManifest
	var localErr error
	if cacheEnabled {
		local, localErr = readLocalManifest(localPath)
	} else {
		localErr = errors.New("cache disabled")
	}

	// Tier 1: local file is fresh.
	if localErr == nil && local.isFresh(cacheTTL) {
		log.Debugf(ctx, "Using cached manifest from %s", localPath)
		return local.manifest, nil
	}

	// Tier 2: fetch from remote (local file missing or stale).
	m, fetchErr := fetchRemoteWithRetry(ctx)
	if fetchErr == nil {
		if cacheEnabled {
			writeLocalManifest(ctx, localPath, m)
		}
		return m, nil
	}

	// Tier 3a: local file exists but stale — use it anyway.
	if localErr == nil {
		log.Debugf(ctx, "Using stale cached manifest (remote failed: %v)", fetchErr)
		return local.manifest, nil
	}

	// Tier 3b: embedded manifest.
	m, embeddedErr := parseEmbeddedManifest()
	if embeddedErr == nil {
		log.Debugf(ctx, "Using embedded manifest (remote and local cache failed)")
		return m, nil
	}

	return nil, fmt.Errorf("all manifest sources failed (remote: %w, embedded: %w)", fetchErr, embeddedErr)
}

// parseEmbeddedManifest parses the embedded manifest once and caches the result.
var parseEmbeddedManifest = sync.OnceValues(func() (Manifest, error) {
	return parseManifest(build.CLICompatManifestJSON)
})

// ResolveEmbeddedAppKitVersion resolves the AppKit version from only the
// embedded manifest for the current CLI version. Used as a fallback when the
// primary version (from remote or cached manifest) points to a non-existent tag,
// and for help text defaults where a network call is not appropriate.
func ResolveEmbeddedAppKitVersion() (string, error) {
	m, err := parseEmbeddedManifest()
	if err != nil {
		return "", fmt.Errorf("embedded manifest: %w", err)
	}
	entry, err := Resolve(m, build.GetInfo().Version)
	if err != nil {
		return "", fmt.Errorf("embedded manifest resolve: %w", err)
	}
	return entry.AppKit, nil
}

// ResolveEmbeddedAgentSkillsVersion resolves the Agent Skills version from only
// the embedded manifest for the current CLI version. Used as a fallback when the
// primary version points to a non-existent tag.
func ResolveEmbeddedAgentSkillsVersion() (string, error) {
	m, err := parseEmbeddedManifest()
	if err != nil {
		return "", fmt.Errorf("embedded manifest: %w", err)
	}
	entry, err := Resolve(m, build.GetInfo().Version)
	if err != nil {
		return "", fmt.Errorf("embedded manifest resolve: %w", err)
	}
	return entry.AgentSkills, nil
}

// IsNotFoundError reports whether the error indicates a "not found" condition
// (e.g. HTTP 404, missing git branch/tag). Used by consumers to decide whether
// to fall back to the embedded manifest.
func IsNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrNotFound) {
		return true
	}
	if httpErr, ok := errors.AsType[*HTTPStatusError](err); ok && httpErr.StatusCode == http.StatusNotFound {
		return true
	}
	// Git clone errors include "not found" in stderr when a branch/tag does not
	// exist (e.g. "Remote branch X not found in upstream origin"). This is a
	// pragmatic fallback until git.Clone wraps a typed error.
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found")
}

// Resolve returns the manifest entry for the given CLI version.
//
// Each versioned entry defines a range floor: it applies to that CLI version
// and all versions above it, up to (but not including) the next entry.
//
// Resolution order:
//  1. Dev builds (version starts with "0.0.0-dev") use the highest versioned entry.
//  2. Exact match on CLI version.
//  3. Nearest lower version (semver-sorted). This also handles CLI versions
//     newer than all entries, returning the highest known entry.
//  4. If CLI is older than all entries, use the lowest (oldest) entry.
func Resolve(m Manifest, cliVersion string) (Entry, error) {
	if len(m) == 0 {
		return Entry{}, errors.New("empty compatibility manifest")
	}

	// Collect and sort versioned keys descending.
	versions := sortedVersions(m)
	if len(versions) == 0 {
		return Entry{}, errors.New("compatibility manifest has no versioned entries")
	}

	// Dev builds (0.0.0-dev*) have semver lower than all real CLI versions,
	// so they would incorrectly resolve to the lowest entry. Use the highest
	// versioned entry instead, since dev builds represent the bleeding edge.
	if strings.HasPrefix(cliVersion, devVersionPrefix) {
		return m[versions[0]], nil
	}

	// Exact match.
	if entry, ok := m[cliVersion]; ok {
		return entry, nil
	}

	// Find the nearest lower version.
	vCLI := "v" + cliVersion
	for _, v := range versions {
		if semver.Compare("v"+v, vCLI) <= 0 {
			return m[v], nil
		}
	}

	// CLI is older than all entries — use the lowest (oldest) entry.
	return m[versions[len(versions)-1]], nil
}

// sortedVersions returns manifest keys sorted descending by semver.
func sortedVersions(m Manifest) []string {
	versions := make([]string, 0, len(m))
	for k := range m {
		versions = append(versions, k)
	}
	slices.SortFunc(versions, func(a, b string) int {
		return semver.Compare("v"+b, "v"+a)
	})
	return versions
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
		log.Debugf(ctx, "Could not determine user cache directory: %v", err)
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
// temp-file-then-rename pattern for atomicity.
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
		log.Warnf(ctx, "Failed to create cache directory %s: %v", dir, err)
		return
	}
	tmp, err := os.CreateTemp(dir, ".compat-manifest-*.tmp")
	if err != nil {
		log.Warnf(ctx, "Failed to create temp cache file: %v", err)
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
	if err := os.Rename(tmpPath, path); err != nil {
		log.Warnf(ctx, "Failed to rename temp cache file: %v", err)
	}
}

// --- Remote fetch ---

// fetchRemoteWithRetry wraps fetchRemote with retries on transient errors.
// Client errors (4xx) are not retried since they will not recover.
func fetchRemoteWithRetry(ctx context.Context) (Manifest, error) {
	var lastErr error
	for attempt := range maxFetchAttempts {
		if attempt > 0 {
			log.Debugf(ctx, "Retrying manifest fetch (attempt %d/%d)", attempt+1, maxFetchAttempts)
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

		// Do not retry client errors (4xx) — they won't resolve on retry.
		if httpErr, ok := errors.AsType[*HTTPStatusError](err); ok && httpErr.StatusCode >= 400 && httpErr.StatusCode < 500 {
			return nil, lastErr
		}
	}
	return nil, lastErr
}

func fetchRemote(ctx context.Context) (Manifest, error) {
	log.Debugf(ctx, "Fetching compatibility manifest from %s", manifestURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, manifestURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "databricks-cli/"+build.GetInfo().Version)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, &HTTPStatusError{StatusCode: resp.StatusCode}
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
	for k := range m {
		if !semver.IsValid("v" + k) {
			return nil, fmt.Errorf("invalid semver key %q in compatibility manifest", k)
		}
	}
	for k, entry := range m {
		if entry.AppKit == "" {
			return nil, fmt.Errorf("manifest entry %q has empty appkit version", k)
		}
		if entry.AgentSkills == "" {
			return nil, fmt.Errorf("manifest entry %q has empty skills version", k)
		}
		if !semver.IsValid("v" + entry.AppKit) {
			return nil, fmt.Errorf("manifest entry %q has invalid appkit version %q", k, entry.AppKit)
		}
		if !semver.IsValid("v" + entry.AgentSkills) {
			return nil, fmt.Errorf("manifest entry %q has invalid skills version %q", k, entry.AgentSkills)
		}
	}
	return m, nil
}

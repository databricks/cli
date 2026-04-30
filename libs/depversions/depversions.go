package depversions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/cache"
	"github.com/databricks/cli/libs/log"
	"golang.org/x/mod/semver"
)

const (
	// manifestURL is the raw GitHub URL for the compatibility manifest.
	manifestURL = "https://raw.githubusercontent.com/databricks/appkit/main/cli-compat.json"

	// fetchTimeout is the HTTP timeout for fetching the manifest at runtime.
	fetchTimeout = 3 * time.Second

	// nextKey is the special manifest key for CLI versions newer than any entry.
	nextKey = "next"

	cacheComponent    = "compat-manifest"
	cacheTTL          = 1 * time.Hour
	fetchRetries      = 2
	fetchRetryBackoff = 300 * time.Millisecond
)

// manifestFingerprint is the cache key for the compatibility manifest.
type manifestFingerprint struct {
	URL string `json:"url"`
}

// Entry maps a CLI version to compatible AppKit and Agent Skills versions.
type Entry struct {
	AppKit      string `json:"appkit"`
	AgentSkills string `json:"skills"`
}

// Manifest is the compatibility manifest: a map of CLI version strings to entries.
type Manifest map[string]Entry

// httpClient is the HTTP client used for manifest fetches. Package-level var
// so tests can replace it.
var httpClient = &http.Client{Timeout: fetchTimeout}

// FetchManifest returns the compatibility manifest, checking in order:
// 1h local disk cache, then remote fetch with retry.
func FetchManifest(ctx context.Context) (Manifest, error) {
	c := cache.NewCache(ctx, cacheComponent, cacheTTL, nil)
	fp := manifestFingerprint{URL: manifestURL}
	return cache.GetOrCompute(ctx, c, fp, func(ctx context.Context) (Manifest, error) {
		return fetchRemoteWithRetry(ctx)
	})
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
	if strings.HasPrefix(cliVersion, "0.0.0-dev") {
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

// resolveEntry fetches the manifest, resolves for the given CLI version,
// and falls back to the build-time dep versions on failure.
func resolveEntry(ctx context.Context, cliVersion string) (Entry, error) {
	m, fetchErr := FetchManifest(ctx)
	if fetchErr == nil {
		entry, err := Resolve(m, cliVersion)
		if err == nil {
			return entry, nil
		}
		log.Debugf(ctx, "Resolve failed (%v), trying build-time fallback", err)
	}

	dv := build.GetDepVersions()
	if dv.AppKit != "" {
		log.Debugf(ctx, "Using build-time dep versions: appkit=%s skills=%s", dv.AppKit, dv.AgentSkills)
		return Entry{AppKit: dv.AppKit, AgentSkills: dv.AgentSkills}, nil
	}

	if fetchErr != nil {
		return Entry{}, fmt.Errorf("manifest fetch failed and no build-time versions available: %w", fetchErr)
	}
	return Entry{}, errors.New("no compatible versions available")
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

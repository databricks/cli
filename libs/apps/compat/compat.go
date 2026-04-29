package compat

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/databricks/cli/libs/log"
	"golang.org/x/mod/semver"
)

const (
	// manifestURL is the raw GitHub URL for the compatibility manifest.
	manifestURL = "https://raw.githubusercontent.com/databricks/appkit/main/cli-compat.json"

	// fetchTimeout is the HTTP timeout for fetching the manifest at runtime.
	fetchTimeout = 5 * time.Second

	// nextKey is the special manifest key for CLI versions newer than any entry.
	nextKey = "next"
)

// Entry maps a CLI version to compatible AppKit and skills versions.
type Entry struct {
	Appkit string `json:"appkit"`
	Skills string `json:"skills"`
}

// Manifest is the compatibility manifest: a map of CLI version strings to entries.
type Manifest map[string]Entry

//go:embed cli-compat.json
var embeddedManifest []byte

// httpClient is the HTTP client used for manifest fetches. Package-level var
// so tests can replace it.
var httpClient = &http.Client{Timeout: fetchTimeout}

// FetchManifest fetches the latest manifest from GitHub.
// If the fetch fails, it falls back to the embedded manifest.
func FetchManifest(ctx context.Context) (Manifest, error) {
	m, err := fetchRemote(ctx)
	if err != nil {
		log.Debugf(ctx, "Manifest fetch failed (%v), using embedded fallback", err)
		return parseManifest(embeddedManifest)
	}
	return m, nil
}

// Resolve returns the manifest entry for the given CLI version.
//
// Resolution order:
//  1. Dev builds (version starts with "0.0.0-dev") use the "next" entry.
//  2. Exact match on CLI version.
//  3. Nearest lower version (semver-sorted).
//  4. If CLI is newer than all entries, use "next".
//  5. If CLI is older than all entries, use "next" (best effort).
func Resolve(m Manifest, cliVersion string) (Entry, error) {
	if len(m) == 0 {
		return Entry{}, fmt.Errorf("empty compatibility manifest")
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
	sort.Slice(versions, func(i, j int) bool {
		return semver.Compare("v"+versions[i], "v"+versions[j]) > 0
	})

	// Find the nearest lower version.
	vCLI := "v" + cliVersion
	for _, v := range versions {
		if semver.Compare("v"+v, vCLI) <= 0 {
			return m[v], nil
		}
	}

	// CLI is older than all entries — best effort: use "next".
	return next, nil
}

func fetchRemote(ctx context.Context) (Manifest, error) {
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

	body, err := io.ReadAll(resp.Body)
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
		return nil, fmt.Errorf("empty compatibility manifest")
	}
	if _, ok := m[nextKey]; !ok {
		return nil, fmt.Errorf("compatibility manifest missing %q key", nextKey)
	}
	return m, nil
}

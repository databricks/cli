package installer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/databricks/cli/libs/log"
)

// ManifestSource abstracts how the skills manifest is fetched.
type ManifestSource interface {
	// FetchManifest fetches the skills manifest at the given ref.
	FetchManifest(ctx context.Context, ref string) (*Manifest, error)
}

// GitHubManifestSource fetches manifests from GitHub.
type GitHubManifestSource struct{}

// FetchManifest fetches the skills manifest from GitHub at the given ref.
func (s *GitHubManifestSource) FetchManifest(ctx context.Context, ref string) (*Manifest, error) {
	log.Debugf(ctx, "Fetching skills manifest from %s/%s@%s", skillsRepoOwner, skillsRepoName, ref)
	url := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/manifest.json",
		skillsRepoOwner, skillsRepoName, ref)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch manifest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch manifest: HTTP %d", resp.StatusCode)
	}

	var manifest Manifest
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	return &manifest, nil
}

package installer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
)

// Clock abstracts time for testing.
type Clock interface {
	Now() time.Time
}

// RealClock returns the actual current time.
type RealClock struct{}

// Now returns the current time.
func (RealClock) Now() time.Time { return time.Now() }

// ManifestSource abstracts how the skills manifest and release info are fetched.
type ManifestSource interface {
	// FetchManifest fetches the skills manifest at the given ref.
	FetchManifest(ctx context.Context, ref string) (*Manifest, error)

	// FetchLatestRelease returns the latest release tag.
	// Falls back to defaultSkillsRepoRef on any error.
	FetchLatestRelease(ctx context.Context) (string, error)
}

// GitHubManifestSource fetches manifests and release info from GitHub.
type GitHubManifestSource struct{}

// FetchManifest fetches the skills manifest from GitHub at the given ref.
func (s *GitHubManifestSource) FetchManifest(ctx context.Context, ref string) (*Manifest, error) {
	log.Infof(ctx, "Fetching skills manifest from %s/%s@%s", skillsRepoOwner, skillsRepoName, ref)
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

// FetchLatestRelease returns the latest release tag from GitHub.
// If DATABRICKS_SKILLS_REF is set, it is returned immediately.
// On any error (network, non-200, parse), falls back to defaultSkillsRepoRef.
func (s *GitHubManifestSource) FetchLatestRelease(ctx context.Context) (string, error) {
	if ref := env.Get(ctx, "DATABRICKS_SKILLS_REF"); ref != "" {
		return ref, nil
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest",
		skillsRepoOwner, skillsRepoName)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		log.Debugf(ctx, "Failed to create release request, falling back to %s: %v", defaultSkillsRepoRef, err)
		return defaultSkillsRepoRef, nil
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Debugf(ctx, "Failed to fetch latest release, falling back to %s: %v", defaultSkillsRepoRef, err)
		return defaultSkillsRepoRef, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Debugf(ctx, "Latest release returned HTTP %d, falling back to %s", resp.StatusCode, defaultSkillsRepoRef)
		return defaultSkillsRepoRef, nil
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		log.Debugf(ctx, "Failed to parse release response, falling back to %s: %v", defaultSkillsRepoRef, err)
		return defaultSkillsRepoRef, nil
	}

	if release.TagName == "" {
		log.Debugf(ctx, "Empty tag_name in release response, falling back to %s", defaultSkillsRepoRef)
		return defaultSkillsRepoRef, nil
	}

	return release.TagName, nil
}

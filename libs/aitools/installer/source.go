package installer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/databricks/cli/libs/clicompat"
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
	url := fmt.Sprintf("%s/%s/manifest.json", GetSkillsBaseURL(ctx), ref)

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

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("skills manifest at %s@%s: %w", skillsRepoName, ref, clicompat.ErrNotFound)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch manifest: HTTP %d", resp.StatusCode)
	}

	var manifest Manifest
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	normalizeManifest(&manifest)
	return &manifest, nil
}

// normalizeManifest stamps SourceName, defaults RepoDir, and suffixes
// experimental install keys while preserving unsuffixed upstream fetch paths.
func normalizeManifest(m *Manifest) {
	if m.Skills == nil {
		m.Skills = map[string]SkillMeta{}
	}
	out := make(map[string]SkillMeta, len(m.Skills))
	for name, meta := range m.Skills {
		if meta.RepoDir == "" {
			meta.RepoDir = stableSkillsRepoPath
		}
		meta.SourceName = name
		if meta.IsExperimental() {
			out[name+experimentalSuffix] = meta
		} else {
			out[name] = meta
		}
	}
	m.Skills = out
}

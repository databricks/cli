package installer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"regexp"
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
		return nil, &FetchError{Stage: FetchStageManifest, Err: fmt.Errorf("failed to fetch manifest: %w", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, &FetchError{Stage: FetchStageManifest, StatusCode: resp.StatusCode, Err: fmt.Errorf("skills manifest at %s@%s: %w", skillsRepoName, ref, clicompat.ErrNotFound)}
	}
	if resp.StatusCode != http.StatusOK {
		return nil, newHTTPFetchError(FetchStageManifest, resp, fmt.Errorf("failed to fetch manifest: HTTP %d", resp.StatusCode))
	}

	// Read the whole body before parsing it: a connection dropped mid-body is a
	// fetch failure, while a body that arrived but is empty or truncated JSON is
	// an invalid manifest. A streaming decoder reports both as io.ErrUnexpectedEOF.
	// The cap is far above the manifest's size, so a large non-manifest body (a
	// captive portal, a wrong mirror) is not read whole. One byte past the cap
	// tells an oversized body apart from one that fits exactly.
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxFetchBodySize+1))
	if err != nil {
		return nil, &FetchError{Stage: FetchStageManifest, StatusCode: resp.StatusCode, Err: fmt.Errorf("failed to read manifest: %w", err)}
	}
	if len(data) > maxFetchBodySize {
		return nil, &FetchError{Stage: FetchStageManifest, StatusCode: resp.StatusCode, Err: ErrManifestTooLarge}
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, &FetchError{Stage: FetchStageManifest, StatusCode: resp.StatusCode, Err: fmt.Errorf("failed to parse manifest: %w", err)}
	}

	if err := validateManifestPaths(&manifest); err != nil {
		return nil, &FetchError{Stage: FetchStageManifest, StatusCode: resp.StatusCode, Err: fmt.Errorf("invalid skills manifest: %w", err)}
	}
	normalizeManifest(&manifest)
	return &manifest, nil
}

// skillNamePattern is the Agent Skills name format
// (https://agentskills.io/specification#name-field): lowercase letters, digits
// and single hyphens. Every name is a directory under the skills dir, next to
// the installer's state file, so the format also keeps names from aliasing each
// other or that file on case-insensitive filesystems or through Windows
// trailing dots and spaces.
var skillNamePattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// windowsDeviceNamePattern matches the names Windows resolves to a device in
// any directory. They are rejected on every OS so a manifest is valid or
// invalid the same way everywhere.
var windowsDeviceNamePattern = regexp.MustCompile(`^(con|prn|aux|nul|com[1-9]|lpt[1-9])$`)

// validateManifestPaths rejects skill names and files that would be written
// outside the skill's own directory. The manifest comes from the network, so it
// must not be able to name ../../.bashrc.
func validateManifestPaths(m *Manifest) error {
	for name, meta := range m.Skills {
		if len(name) > 64 || !skillNamePattern.MatchString(name) {
			return fmt.Errorf("skill name %q must be at most 64 lowercase letters, digits and single hyphens", name)
		}
		if windowsDeviceNamePattern.MatchString(name) {
			return fmt.Errorf("skill name %q is a reserved Windows device name", name)
		}
		for _, file := range meta.Files {
			if !filepath.IsLocal(file) {
				return fmt.Errorf("skill %q lists file %q outside its directory", name, file)
			}
		}
	}
	return nil
}

// normalizeManifest stamps SourceName and defaults RepoDir for older manifests.
func normalizeManifest(m *Manifest) {
	if m.Skills == nil {
		m.Skills = map[string]SkillMeta{}
	}
	for name, meta := range m.Skills {
		if meta.RepoDir == "" {
			meta.RepoDir = stableSkillsRepoPath
		}
		meta.SourceName = name
		m.Skills[name] = meta
	}
}

package installer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"path"
	"regexp"
	"slices"
	"strings"
	"sync"
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

	normalizeManifest(&manifest)
	return &manifest, nil
}

// maxSkillNameLength is the Agent Skills limit on the length of a skill name.
const maxSkillNameLength = 64

// skillNamePattern is the Agent Skills name format
// (https://agentskills.io/specification#name-field): lowercase letters, digits
// and single hyphens. Every name is a directory under the skills dir, next to
// the installer's state file, so the format also keeps names from aliasing each
// other on case-insensitive filesystems or through Windows trailing dots and
// spaces.
var skillNamePattern = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
})

// checkManifestPaths fails a manifest that names a skill or file outside its
// directory, or a skill named like the state file. The manifest comes from the
// network, so it must not be able to name ../../.bashrc, and only a broken or
// tampered manifest lists such an entry. The rule is the same on every OS.
// Entries that stay inside their directory but are not portable are left to
// unportableSkill, which skips only that skill.
func checkManifestPaths(m *Manifest) error {
	for _, name := range slices.Sorted(maps.Keys(m.Skills)) {
		if name == "" || name == "." || name == ".." || name == stateFileName || strings.ContainsAny(name, `/\`) {
			return fmt.Errorf("%w: skill name %q", ErrUnsafeManifestPath, name)
		}
		for _, file := range m.Skills[name].Files {
			if escapesDir(file) {
				return fmt.Errorf("%w: skill %q lists file %q", ErrUnsafeManifestPath, name, file)
			}
		}
	}
	return nil
}

// escapesDir reports whether file, joined onto a directory, names a path
// outside it on some OS: an absolute path, a Windows drive, or more ".."
// elements than the elements before them, with either slash as the separator.
func escapesDir(file string) bool {
	if strings.HasPrefix(file, "/") || strings.HasPrefix(file, `\`) || (len(file) >= 2 && file[1] == ':') {
		return true
	}
	depth := 0
	for _, elem := range strings.FieldsFunc(file, func(r rune) bool { return r == '/' || r == '\\' }) {
		switch elem {
		case ".":
		case "..":
			depth--
			if depth < 0 {
				return true
			}
		default:
			depth++
		}
	}
	return false
}

// unportableSkill returns why a skill that stays inside its directory still
// cannot be installed the same way on every OS, or "" if it can. Install skips
// such a skill and update keeps the installed copy, the way both treat a skill
// that needs a newer CLI. Each file must be a clean slash-separated path, so
// the paths recorded in the install state match the ones update walks on disk.
// Each element is limited to letters, digits, '.', '_' and '-', does not end in
// a dot and is not a Windows device name, and no two files differ only in case.
func unportableSkill(name string, files []string) string {
	if len(name) > maxSkillNameLength || !skillNamePattern().MatchString(name) {
		return fmt.Sprintf("the name is not at most %d lowercase letters, digits and single hyphens", maxSkillNameLength)
	}
	if isWindowsDeviceName(name) {
		return "the name is a reserved Windows device name"
	}
	seen := make(map[string]string, len(files))
	for _, file := range files {
		if file == "." || path.Clean(file) != file {
			return fmt.Sprintf("file %q is not a clean slash-separated path", file)
		}
		for elem := range strings.SplitSeq(file, "/") {
			if !portableElementPattern().MatchString(elem) || strings.HasSuffix(elem, ".") {
				return fmt.Sprintf("file %q has a path element that is not portable", file)
			}
			stem, _, _ := strings.Cut(elem, ".")
			if isWindowsDeviceName(stem) {
				return fmt.Sprintf("file %q names a reserved Windows device", file)
			}
		}
		folded := strings.ToLower(file)
		if other, ok := seen[folded]; ok {
			return fmt.Sprintf("files %q and %q differ only in case", other, file)
		}
		seen[folded] = file
	}
	return ""
}

// portableElementPattern is the characters a file path element may use: ones
// that mean the same thing in a file name on every OS.
var portableElementPattern = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
})

// isWindowsDeviceName reports whether s, in any case, is a name Windows
// resolves to a device in every directory.
func isWindowsDeviceName(s string) bool {
	s = strings.ToLower(s)
	switch s {
	case "con", "prn", "aux", "nul":
		return true
	}
	return len(s) == 4 && (strings.HasPrefix(s, "com") || strings.HasPrefix(s, "lpt")) && s[3] >= '1' && s[3] <= '9'
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

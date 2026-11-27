package installer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/databricks/cli/libs/clicompat"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/skills"
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
		return nil, skills.NewHTTPFetchError(FetchStageManifest, resp, fmt.Errorf("failed to fetch manifest: HTTP %d", resp.StatusCode))
	}

	// Read the whole body before parsing it: a connection dropped mid-body is a
	// fetch failure, while a body that arrived but is empty or truncated JSON is
	// an invalid manifest. A streaming decoder reports both as io.ErrUnexpectedEOF.
	// The cap is far above the manifest's size, so a large non-manifest body (a
	// captive portal, a wrong mirror) is not read whole. One byte past the cap
	// tells an oversized body apart from one that fits exactly.
	data, err := io.ReadAll(io.LimitReader(resp.Body, skills.MaxFetchBodySize+1))
	if err != nil {
		return nil, &FetchError{Stage: FetchStageManifest, StatusCode: resp.StatusCode, Err: fmt.Errorf("failed to read manifest: %w", err)}
	}
	if len(data) > skills.MaxFetchBodySize {
		return nil, &FetchError{Stage: FetchStageManifest, StatusCode: resp.StatusCode, Err: ErrManifestTooLarge}
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, &FetchError{Stage: FetchStageManifest, StatusCode: resp.StatusCode, Err: fmt.Errorf("failed to parse manifest: %w", err)}
	}

	skills.NormalizeManifest(&manifest)
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

// unportableSkill returns why a skill that stays inside its directory still
// cannot be installed the same way on every OS, or "" if it can. Install skips
// such a skill and update keeps the installed copy, the way both treat a skill
// that needs a newer CLI. Each file must be a clean slash-separated path, so
// the paths recorded in the install state match the ones update walks on disk.
// No path element may use a character Windows forbids in a file name, end in a
// dot or a space, or be a Windows device name. No two paths, folders included,
// may differ only in case, and no path may be both a file and a folder: either
// would merge or collide on a case-insensitive filesystem.
func unportableSkill(name string, files []string) string {
	if len(name) > maxSkillNameLength || !skillNamePattern().MatchString(name) {
		return fmt.Sprintf("the name is not at most %d lowercase letters, digits and single hyphens", maxSkillNameLength)
	}
	if isWindowsDeviceName(name) {
		return "the name is a reserved Windows device name"
	}
	type seenPath struct {
		path string
		dir  bool
	}
	seen := make(map[string]seenPath, len(files))
	for _, file := range files {
		if file == "." || path.Clean(file) != file {
			return fmt.Sprintf("file %q is not a clean slash-separated path", file)
		}
		elems := strings.Split(file, "/")
		for i, elem := range elems {
			if strings.ContainsFunc(elem, isUnportableRune) || strings.HasSuffix(elem, ".") || strings.HasSuffix(elem, " ") {
				return fmt.Sprintf("file %q has a path element that is not portable", file)
			}
			stem, _, _ := strings.Cut(elem, ".")
			if isWindowsDeviceName(strings.TrimRight(stem, " ")) {
				return fmt.Sprintf("file %q names a reserved Windows device", file)
			}
			p := strings.Join(elems[:i+1], "/")
			dir := i < len(elems)-1
			folded := strings.ToLower(p)
			prev, ok := seen[folded]
			switch {
			case !ok:
				seen[folded] = seenPath{path: p, dir: dir}
			case prev.path != p:
				return fmt.Sprintf("paths %q and %q differ only in case", prev.path, p)
			case prev.dir != dir:
				return fmt.Sprintf("path %q is both a file and a folder", p)
			}
		}
	}
	return ""
}

// isUnportableRune reports whether r cannot appear in a file name on Windows:
// a control character or one of <>:"/\|?*.
func isUnportableRune(r rune) bool {
	return r < 0x20 || r == 0x7f || strings.ContainsRune(`<>:"\|?*`, r)
}

// isWindowsDeviceName reports whether s, in any case, is a name Windows
// resolves to a device in every directory, as in Go's internal isReservedName.
func isWindowsDeviceName(s string) bool {
	s = strings.ToLower(s)
	switch s {
	case "con", "prn", "aux", "nul", "conin$", "conout$":
		return true
	}
	rest, ok := strings.CutPrefix(s, "com")
	if !ok {
		rest, ok = strings.CutPrefix(s, "lpt")
	}
	if !ok {
		return false
	}
	switch rest {
	case "1", "2", "3", "4", "5", "6", "7", "8", "9", "\u00b9", "\u00b2", "\u00b3":
		return true
	}
	return false
}

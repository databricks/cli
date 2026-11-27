package skills

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/databricks/cli/libs/clicompat"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/versioncheck"
	"golang.org/x/mod/semver"
)

const (
	// RepoOwner and RepoName locate the skills repo on GitHub.
	RepoOwner = "databricks"
	RepoName  = "databricks-agent-skills"
)

// FetchConcurrency caps the number of in-flight skill file fetches.
// Each file is one HTTPS GET to raw.githubusercontent.com; sequential
// fetches were latency-bound on TLS handshakes. 8 is enough to amortise
// the round-trip across a typical skill's files without overwhelming the
// upstream CDN.
const FetchConcurrency = 8

// HTTPClient is shared by the skill file fetches and the latest-release lookup
// so the underlying transport reuses TCP+TLS connections. The manifest fetch in
// libs/aitools/installer makes its own client and does not use it.
// MaxIdleConnsPerHost is raised above the default
// of 2 to leave headroom above FetchConcurrency, so a brief overlap between a
// closing and a new connection doesn't force a fresh handshake.
var HTTPClient = sync.OnceValue(func() *http.Client {
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.MaxIdleConnsPerHost = FetchConcurrency * 2
	return &http.Client{
		Timeout:   30 * time.Second,
		Transport: t,
	}
})

// LatestSkillsReleaseRefFn resolves the skills repo's latest release tag. Tests
// replace it so default ref resolution stays deterministic and offline.
var LatestSkillsReleaseRefFn = FetchLatestSkillsReleaseRef

// skillsLatestSentinel is the value cli-compat.json uses for the skills version
// to mean "track latest" rather than pinning a specific release.
const skillsLatestSentinel = clicompat.AgentSkillsLatest

// GetSkillsRef returns the skills repo ref to use and whether it was explicitly
// pinned (as opposed to tracking latest). An unresolved latest release is an error.
//
// By default we track the latest skills release tag, not the skills repo's main
// branch. cli-compat.json is the remote control for this: its newest range
// normally tracks latest, but if a future change makes a skills release
// incompatible with older CLIs, it can be edited remotely to pin those CLIs to
// the last compatible version with no CLI release. DATABRICKS_SKILLS_REF
// overrides everything for eval/reproducibility. (AppKit version resolution is
// unaffected; cli-compat still manages it via its own field.)
func GetSkillsRef(ctx context.Context) (ref string, explicit bool, err error) {
	return ResolveSkillsRef(ctx, false)
}

// ResolveSkillsRef is GetSkillsRef, except that with allowFallback a failed
// latest-release lookup falls back to the embedded cli-compat skills version
// when that version is a pinned release, logging a warning. The fallback ref is
// not explicit. When the embedded version is empty, "latest" or unreadable, the
// lookup error is returned as it would be without allowFallback.
func ResolveSkillsRef(ctx context.Context, allowFallback bool) (ref string, explicit bool, err error) {
	if ref := env.Get(ctx, "DATABRICKS_SKILLS_REF"); ref != "" {
		return ref, true, nil
	}
	v, err := clicompat.ResolveAgentSkillsVersion(ctx)
	if err != nil {
		log.Debugf(ctx, "could not resolve skills compatibility, resolving latest release: %v", err)
		return resolveLatestSkillsReleaseRef(ctx, allowFallback)
	}
	if v == "" || v == skillsLatestSentinel {
		return resolveLatestSkillsReleaseRef(ctx, allowFallback)
	}
	return "v" + v, true, nil
}

func resolveLatestSkillsReleaseRef(ctx context.Context, allowFallback bool) (ref string, explicit bool, err error) {
	ref, err = LatestSkillsReleaseRefFn(ctx)
	if err == nil {
		return ref, false, nil
	}
	if !allowFallback {
		return "", false, fmt.Errorf("resolve latest skills release: %w", err)
	}

	embedded, embeddedErr := clicompat.ResolveEmbeddedAgentSkillsVersion()
	if embeddedErr == nil && embedded != "" && embedded != skillsLatestSentinel {
		log.Warnf(ctx, "Could not resolve latest skills release (%v); falling back to embedded skills version %s", err, embedded)
		return "v" + embedded, false, nil
	}
	if embeddedErr != nil {
		return "", false, fmt.Errorf("resolve latest skills release: %w (embedded fallback failed: %v)", err, embeddedErr) //nolint:errorlint // only the primary error belongs in the chain
	}
	return "", false, fmt.Errorf("resolve latest skills release: %w", err)
}

// FetchLatestSkillsReleaseRef returns the skills repo's latest release tag as a
// v-prefixed semver string. A failed request, an unusable response or an invalid
// tag returns a *FetchError with stage release. A request that cannot be built,
// for example from a malformed DATABRICKS_CLI_GITHUB_API_URL, returns a plain
// error instead.
func FetchLatestSkillsReleaseRef(ctx context.Context) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/releases/latest", versioncheck.GitHubAPIURL(ctx), RepoOwner, RepoName)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := HTTPClient().Do(req)
	if err != nil {
		return "", &FetchError{Stage: FetchStageRelease, Err: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", NewHTTPFetchError(FetchStageRelease, resp, fmt.Errorf("HTTP %d fetching latest skills release", resp.StatusCode))
	}

	var payload struct {
		TagName string `json:"tag_name"`
	}
	// A response that arrived but names no usable release is still a failed
	// release lookup, so these errors carry the stage too.
	if err := json.NewDecoder(io.LimitReader(resp.Body, MaxFetchBodySize)).Decode(&payload); err != nil {
		return "", &FetchError{Stage: FetchStageRelease, StatusCode: resp.StatusCode, Err: fmt.Errorf("parse latest skills release: %w", err)}
	}
	tag := strings.TrimSpace(payload.TagName)
	if tag == "" {
		return "", &FetchError{Stage: FetchStageRelease, StatusCode: resp.StatusCode, Err: errors.New("latest skills release has empty tag_name")}
	}
	if !strings.HasPrefix(tag, "v") {
		tag = "v" + tag
	}
	if !semver.IsValid(tag) {
		return "", &FetchError{Stage: FetchStageRelease, StatusCode: resp.StatusCode, Err: fmt.Errorf("latest skills release has invalid tag %q", payload.TagName)}
	}
	return tag, nil
}

// GetSkillsBaseURL returns the manifest and skill fetch base URL.
// DATABRICKS_SKILLS_BASE_URL overrides GitHub raw URLs for acceptance tests.
func GetSkillsBaseURL(ctx context.Context) string {
	if base := env.Get(ctx, "DATABRICKS_SKILLS_BASE_URL"); base != "" {
		return strings.TrimRight(base, "/")
	}
	return "https://raw.githubusercontent.com/" + RepoOwner + "/" + RepoName
}

// Command repoadvisory reports Go modules affected by GitHub repository security
// advisories. Maintainers publish these advisories on their repository, and they
// are often missing from the Go vulnerability database and GitHub's global
// advisory database, so govulncheck does not report them.
//
// It reads "path version" lines for the modules in a build from stdin and writes
// govulncheck-format JSON for every affected module to stdout, so that the
// output can be piped into vulnbump. Only advisories that name a patched version
// are written; the others are reported on stderr and skipped. Usage:
//
//	go list -deps -f '{{with .Module}}{{if .Version}}{{.Path}} {{.Version}}{{end}}{{end}}' . | repoadvisory | vulnbump .
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"golang.org/x/mod/semver"
)

const (
	githubAPIURL = "https://api.github.com"
	githubPrefix = "github.com/"
	// goEcosystem is the package ecosystem GitHub uses for Go modules.
	goEcosystem = "go"
)

var (
	// goImportRE matches the go-import meta tag served by a vanity import path.
	goImportRE = regexp.MustCompile(`<meta\s+name="go-import"\s+content="([^"]+)"`)
	// constraintRE splits a constraint such as ">= 1.35.0" into operator and version.
	constraintRE = regexp.MustCompile(`^(>=|<=|>|<|=)?\s*(\S+)$`)
	// nextLinkRE extracts the next page URL from a GitHub Link header.
	nextLinkRE = regexp.MustCompile(`<([^>]+)>;\s*rel="next"`)
)

// module is a Go module in the build and the version it is built at.
type module struct {
	Path    string
	Version string
}

// repoAdvisory mirrors the fields of a GitHub repository security advisory that
// this tool reads. See https://docs.github.com/en/rest/security-advisories/repository-advisories.
type repoAdvisory struct {
	GHSAID          string              `json:"ghsa_id"`
	CVEID           string              `json:"cve_id"`
	HTMLURL         string              `json:"html_url"`
	WithdrawnAt     string              `json:"withdrawn_at"`
	Vulnerabilities []repoVulnerability `json:"vulnerabilities"`
}

// repoVulnerability is one affected package within a repository advisory.
type repoVulnerability struct {
	Package struct {
		Ecosystem string `json:"ecosystem"`
		Name      string `json:"name"`
	} `json:"package"`
	VulnerableVersionRange string `json:"vulnerable_version_range"`
	PatchedVersions        string `json:"patched_versions"`
}

// message is one line of govulncheck -format json output. Only the fields that
// vulnbump reads are set.
type message struct {
	OSV     *osvEntry `json:"osv,omitempty"`
	Finding *finding  `json:"finding,omitempty"`
}

type osvEntry struct {
	ID               string   `json:"id"`
	Aliases          []string `json:"aliases,omitempty"`
	DatabaseSpecific struct {
		URL string `json:"url"`
	} `json:"database_specific"`
}

type finding struct {
	OSV          string  `json:"osv"`
	FixedVersion string  `json:"fixed_version,omitempty"`
	Trace        []frame `json:"trace"`
}

type frame struct {
	Module  string `json:"module"`
	Version string `json:"version"`
}

// hit is an advisory that affects a module at the version being built. fixed is
// the patched version with a "v" prefix, or "" when the advisory names none.
type hit struct {
	advisory repoAdvisory
	fixed    string
}

// advisorySource resolves module paths to GitHub repositories and lists the
// published advisories of a repository. It is an interface so tests need no
// network.
type advisorySource interface {
	repoFor(modulePath string) (string, error)
	listAdvisories(repo string) ([]repoAdvisory, error)
}

// liveSource queries the GitHub REST API and go-import meta tags.
type liveSource struct {
	baseURL string
	token   string
	http    *http.Client
}

// main is the entry point. It delegates to run so the flow stays testable.
func main() {
	src := liveSource{
		baseURL: githubAPIURL,
		token:   os.Getenv("GITHUB_TOKEN"), //nolint:forbidigo // The tools module cannot import libs/env.
		http:    &http.Client{Timeout: 30 * time.Second},
	}
	if err := run(os.Stdin, os.Stdout, os.Stderr, src); err != nil {
		fmt.Fprintln(os.Stderr, "repoadvisory:", err)
		os.Exit(1)
	}
}

// run writes a govulncheck-format message for each advisory with a patched
// version that affects a module in in. It reports to errOut, and skips, each
// affected module whose advisory names no patched version.
func run(in io.Reader, out, errOut io.Writer, src advisorySource) error {
	mods, err := parseModules(in)
	if err != nil {
		return err
	}

	enc := json.NewEncoder(out)
	advisoriesByRepo := map[string][]repoAdvisory{}
	written := map[string]bool{}
	for _, m := range mods {
		repo, err := src.repoFor(m.Path)
		if err != nil {
			return fmt.Errorf("resolve repository for %s: %w", m.Path, err)
		}
		if repo == "" {
			continue
		}

		advisories, ok := advisoriesByRepo[repo]
		if !ok {
			advisories, err = src.listAdvisories(repo)
			if err != nil {
				return err
			}
			advisoriesByRepo[repo] = advisories
		}

		hits, err := affectedBy(m, advisories)
		if err != nil {
			return fmt.Errorf("check %s@%s: %w", m.Path, m.Version, err)
		}
		for _, h := range hits {
			id := h.advisory.GHSAID
			// Without a patched version there is nothing to bump to.
			if h.fixed == "" {
				fmt.Fprintf(errOut, "skipping %s for %s@%s: it names no patched version: %s\n",
					id, m.Path, m.Version, h.advisory.HTMLURL)
				continue
			}
			if !written[id] {
				written[id] = true
				if err := enc.Encode(message{OSV: newOSVEntry(h.advisory)}); err != nil {
					return fmt.Errorf("encode %s: %w", id, err)
				}
			}
			f := &finding{
				OSV:          id,
				FixedVersion: h.fixed,
				Trace:        []frame{{Module: m.Path, Version: m.Version}},
			}
			if err := enc.Encode(message{Finding: f}); err != nil {
				return fmt.Errorf("encode finding for %s: %w", m.Path, err)
			}
		}
	}
	return nil
}

// parseModules reads "path version" lines and returns the distinct modules in
// input order. Lines without a version are skipped; go list emits them for
// packages without a module.
func parseModules(r io.Reader) ([]module, error) {
	var mods []module
	seen := map[string]bool{}
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) == 0 {
			continue
		}
		if len(fields) != 2 {
			return nil, fmt.Errorf("parse module line %q: want path and version", sc.Text())
		}
		if seen[fields[0]] {
			continue
		}
		seen[fields[0]] = true
		mods = append(mods, module{Path: fields[0], Version: fields[1]})
	}
	return mods, sc.Err()
}

// affectedBy returns the advisories in advisories that cover the version of m.
// Withdrawn advisories are skipped.
func affectedBy(m module, advisories []repoAdvisory) ([]hit, error) {
	var hits []hit
	for _, adv := range advisories {
		if adv.WithdrawnAt != "" {
			continue
		}
		for _, v := range adv.Vulnerabilities {
			if v.Package.Ecosystem != goEcosystem || v.Package.Name != m.Path {
				continue
			}
			in, err := inRange(m.Version, v.VulnerableVersionRange)
			if err != nil {
				return nil, fmt.Errorf("advisory %s: %w", adv.GHSAID, err)
			}
			if !in {
				continue
			}
			h := hit{advisory: adv}
			if v.PatchedVersions != "" {
				h.fixed, err = parseSemver(v.PatchedVersions)
				if err != nil {
					return nil, fmt.Errorf("advisory %s: %w", adv.GHSAID, err)
				}
			}
			hits = append(hits, h)
			break
		}
	}
	return hits, nil
}

// inRange reports whether version satisfies every comma-separated constraint in
// spec, a GitHub vulnerable_version_range such as ">= 1.35.0, < 1.47.0".
func inRange(version, spec string) (bool, error) {
	for part := range strings.SplitSeq(spec, ",") {
		m := constraintRE.FindStringSubmatch(strings.TrimSpace(part))
		if m == nil {
			return false, fmt.Errorf("parse version constraint %q", part)
		}
		bound, err := parseSemver(m[2])
		if err != nil {
			return false, err
		}
		c := semver.Compare(version, bound)
		var ok bool
		switch m[1] {
		case ">=":
			ok = c >= 0
		case "<=":
			ok = c <= 0
		case ">":
			ok = c > 0
		case "<":
			ok = c < 0
		default:
			ok = c == 0
		}
		if !ok {
			return false, nil
		}
	}
	return true, nil
}

// parseSemver returns v with a "v" prefix, as golang.org/x/mod/semver requires,
// and an error if it is not a valid version. GitHub omits the prefix in
// version ranges and patched_versions.
func parseSemver(v string) (string, error) {
	if !strings.HasPrefix(v, "v") {
		v = "v" + v
	}
	if !semver.IsValid(v) {
		return "", fmt.Errorf("invalid version %q", v)
	}
	return v, nil
}

// newOSVEntry returns the osv message for an advisory. vulnbump labels the
// advisory with its CVE and links to the URL.
func newOSVEntry(adv repoAdvisory) *osvEntry {
	o := &osvEntry{ID: adv.GHSAID}
	if adv.CVEID != "" {
		o.Aliases = []string{adv.CVEID}
	}
	o.DatabaseSpecific.URL = adv.HTMLURL
	return o
}

// repoFor returns the owner/name GitHub repository that hosts modulePath, or ""
// when the module is not hosted on GitHub.
func (s liveSource) repoFor(modulePath string) (string, error) {
	if strings.HasPrefix(modulePath, githubPrefix) {
		return strings.Join(strings.Split(modulePath, "/")[1:3], "/"), nil
	}
	resp, err := s.http.Get("https://" + modulePath + "?go-get=1")
	if err != nil {
		return "", fmt.Errorf("fetch go-import meta for %s: %w", modulePath, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetch go-import meta for %s: %s", modulePath, resp.Status)
	}
	page, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read go-import meta for %s: %w", modulePath, err)
	}
	return parseGoImport(string(page)), nil
}

// parseGoImport returns the owner/name GitHub repository from the go-import meta
// tag in an HTML page, or "" when the tag does not point at github.com.
func parseGoImport(page string) string {
	m := goImportRE.FindStringSubmatch(page)
	if m == nil {
		return ""
	}
	// The content is "<import prefix> <vcs> <repository URL>".
	fields := strings.Fields(m[1])
	const httpsPrefix = "https://" + githubPrefix
	if len(fields) != 3 || fields[1] != "git" || !strings.HasPrefix(fields[2], httpsPrefix) {
		return ""
	}
	return strings.TrimSuffix(strings.TrimPrefix(fields[2], httpsPrefix), ".git")
}

// listAdvisories returns the published security advisories of the owner/name
// repository, following pagination.
func (s liveSource) listAdvisories(repo string) ([]repoAdvisory, error) {
	var all []repoAdvisory
	next := s.baseURL + "/repos/" + repo + "/security-advisories?state=published&per_page=100"
	for next != "" {
		req, err := http.NewRequest(http.MethodGet, next, nil)
		if err != nil {
			return nil, fmt.Errorf("build request for %s: %w", repo, err)
		}
		req.Header.Set("Accept", "application/vnd.github+json")
		if s.token != "" {
			req.Header.Set("Authorization", "Bearer "+s.token)
		}
		resp, err := s.http.Do(req)
		if err != nil {
			return nil, fmt.Errorf("list security advisories for %s: %w", repo, err)
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("list security advisories for %s: %s", repo, resp.Status)
		}
		var page []repoAdvisory
		err = json.NewDecoder(resp.Body).Decode(&page)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("decode security advisories for %s: %w", repo, err)
		}
		all = append(all, page...)
		next = nextLink(resp.Header.Get("Link"))
	}
	return all, nil
}

// nextLink returns the rel="next" URL of a GitHub Link header, or "" if there is none.
func nextLink(header string) string {
	m := nextLinkRE.FindStringSubmatch(header)
	if m == nil {
		return ""
	}
	return m[1]
}

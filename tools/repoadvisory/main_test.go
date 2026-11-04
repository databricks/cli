package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Fixtures are the published repository advisories of jackc/pgx and
// open-telemetry/opentelemetry-go, as returned by the GitHub REST API.
const (
	pgxRepo  = "jackc/pgx"
	otelRepo = "open-telemetry/opentelemetry-go"
)

// fakeSource resolves modules and lists advisories from fixed maps, so run can
// be tested without network access.
type fakeSource struct {
	repos      map[string]string
	advisories map[string][]repoAdvisory
}

func (f fakeSource) repoFor(modulePath string) (string, error) {
	return f.repos[modulePath], nil
}

func (f fakeSource) listAdvisories(repo string) ([]repoAdvisory, error) {
	return f.advisories[repo], nil
}

func loadAdvisories(t *testing.T, name string) []repoAdvisory {
	t.Helper()
	data, err := os.ReadFile("testdata/" + name)
	require.NoError(t, err)
	var advisories []repoAdvisory
	require.NoError(t, json.Unmarshal(data, &advisories))
	return advisories
}

func TestParseModules(t *testing.T) {
	in := "github.com/jackc/pgx/v5 v5.11.0\n\n\ngithub.com/jackc/pgx/v5 v5.11.0\ngolang.org/x/net v0.53.0\n"
	// The main module and standard library have no version; go list emits them
	// as blank lines.
	mods, err := parseModules(strings.NewReader(in))
	require.NoError(t, err)
	assert.Equal(t, []module{
		{Path: "github.com/jackc/pgx/v5", Version: "v5.11.0"},
		{Path: "golang.org/x/net", Version: "v0.53.0"},
	}, mods)
}

func TestParseModulesInvalid(t *testing.T) {
	_, err := parseModules(strings.NewReader("github.com/jackc/pgx/v5\n"))
	require.Error(t, err)
}

func TestParseGoImport(t *testing.T) {
	tests := []struct {
		name string
		page string
		want string
	}{
		{
			name: "github",
			page: `<meta name="go-import" content="gopkg.in/yaml.v3 git https://github.com/go-yaml/yaml.git">`,
			want: "go-yaml/yaml",
		},
		{
			name: "googlesource",
			page: `<meta name="go-import" content="golang.org/x/net git https://go.googlesource.com/net">`,
			want: "",
		},
		{
			name: "no tag",
			page: `<html></html>`,
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, parseGoImport(tt.page))
		})
	}
}

func TestInRange(t *testing.T) {
	tests := []struct {
		version string
		spec    string
		want    bool
	}{
		{version: "v5.11.0", spec: ">= 5.9.0", want: true},
		{version: "v5.8.0", spec: ">= 5.9.0", want: false},
		{version: "v1.44.0", spec: ">= 1.35.0, < 1.47.0", want: true},
		{version: "v1.47.0", spec: ">= 1.35.0, < 1.47.0", want: false},
		{version: "v1.2.3", spec: "= 1.2.3", want: true},
		{version: "v4.18.3", spec: "<= 4.18.3", want: true},
	}
	for _, tt := range tests {
		t.Run(tt.version+" in "+tt.spec, func(t *testing.T) {
			got, err := inRange(tt.version, tt.spec)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestInRangeInvalid(t *testing.T) {
	_, err := inRange("v1.0.0", ">= banana")
	require.Error(t, err)
}

func TestParseSemver(t *testing.T) {
	v, err := parseSemver("1.47.0")
	require.NoError(t, err)
	assert.Equal(t, "v1.47.0", v)

	v, err = parseSemver("v1.2.3")
	require.NoError(t, err)
	assert.Equal(t, "v1.2.3", v)

	_, err = parseSemver("banana")
	require.Error(t, err)
}

func TestAffectedByUnfixedAdvisories(t *testing.T) {
	advisories := loadAdvisories(t, "pgx.json")

	// Both pgx advisories cover v5.11.0 and name no patched version.
	hits, err := affectedBy(module{Path: "github.com/jackc/pgx/v5", Version: "v5.11.0"}, advisories)
	require.NoError(t, err)
	require.Len(t, hits, 2)
	assert.Equal(t, "GHSA-j38q-h3ph-jjpq", hits[0].advisory.GHSAID)
	assert.Equal(t, "GHSA-95fj-3fxw-g62g", hits[1].advisory.GHSAID)
	assert.Empty(t, hits[0].fixed)
	assert.Empty(t, hits[1].fixed)

	// v5.8.0 predates the second advisory's range.
	hits, err = affectedBy(module{Path: "github.com/jackc/pgx/v5", Version: "v5.8.0"}, advisories)
	require.NoError(t, err)
	require.Len(t, hits, 1)
	assert.Equal(t, "GHSA-j38q-h3ph-jjpq", hits[0].advisory.GHSAID)
}

func TestAffectedByFixedAdvisory(t *testing.T) {
	advisories := loadAdvisories(t, "otel.json")

	hits, err := affectedBy(module{Path: "go.opentelemetry.io/otel/trace", Version: "v1.44.0"}, advisories)
	require.NoError(t, err)
	require.Len(t, hits, 1)
	assert.Equal(t, "GHSA-q7gw-24fr-5p69", hits[0].advisory.GHSAID)
	assert.Equal(t, "v1.47.0", hits[0].fixed)

	hits, err = affectedBy(module{Path: "go.opentelemetry.io/otel/trace", Version: "v1.47.0"}, advisories)
	require.NoError(t, err)
	assert.Empty(t, hits)

	// The advisory lists several packages; only the matching one counts.
	hits, err = affectedBy(module{Path: "go.opentelemetry.io/otel", Version: "v1.44.0"}, advisories)
	require.NoError(t, err)
	assert.Empty(t, hits)
}

func TestAffectedBySkipsWithdrawn(t *testing.T) {
	advisories := []repoAdvisory{
		{
			GHSAID:      "GHSA-withdrawn",
			WithdrawnAt: "2026-11-01T00:00:00Z",
			Vulnerabilities: []repoVulnerability{
				{
					Package: struct {
						Ecosystem string `json:"ecosystem"`
						Name      string `json:"name"`
					}{Ecosystem: "go", Name: "example.com/mod"},
					VulnerableVersionRange: ">= 1.0.0",
				},
			},
		},
	}
	hits, err := affectedBy(module{Path: "example.com/mod", Version: "v1.0.0"}, advisories)
	require.NoError(t, err)
	assert.Empty(t, hits)
}

func TestListAdvisoriesFollowsPagination(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if r.URL.Query().Get("page") == "2" {
			fmt.Fprint(w, `[{"ghsa_id": "GHSA-2222-2222-2222"}]`)
			return
		}
		w.Header().Set("Link", fmt.Sprintf(`<%s/repos/o/r/security-advisories?state=published&per_page=100&page=2>; rel="next"`, srv.URL))
		fmt.Fprint(w, `[{"ghsa_id": "GHSA-1111-1111-1111"}]`)
	}))
	t.Cleanup(srv.Close)

	src := liveSource{baseURL: srv.URL, token: "test-token", http: srv.Client()}
	advisories, err := src.listAdvisories("o/r")
	require.NoError(t, err)
	require.Len(t, advisories, 2)
	assert.Equal(t, "GHSA-1111-1111-1111", advisories[0].GHSAID)
	assert.Equal(t, "GHSA-2222-2222-2222", advisories[1].GHSAID)
}

func TestListAdvisoriesHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "rate limited", http.StatusForbidden)
	}))
	t.Cleanup(srv.Close)

	src := liveSource{baseURL: srv.URL, http: srv.Client()}
	_, err := src.listAdvisories("o/r")
	require.Error(t, err)
}

func TestRun(t *testing.T) {
	src := fakeSource{
		repos: map[string]string{
			"github.com/jackc/pgx/v5":        pgxRepo,
			"go.opentelemetry.io/otel/trace": otelRepo,
			"golang.org/x/net":               "",
		},
		advisories: map[string][]repoAdvisory{
			pgxRepo:  loadAdvisories(t, "pgx.json"),
			otelRepo: loadAdvisories(t, "otel.json"),
		},
	}
	in := strings.NewReader("github.com/jackc/pgx/v5 v5.11.0\ngo.opentelemetry.io/otel/trace v1.44.0\ngolang.org/x/net v0.53.0\n")

	var out, errOut bytes.Buffer
	require.NoError(t, run(in, &out, &errOut, src))

	// Only the otel advisory names a patched version, so it is the only one
	// written. Modules that are not on GitHub are skipped.
	assert.Equal(t, `{"osv":{"id":"GHSA-q7gw-24fr-5p69","database_specific":{"url":"https://github.com/open-telemetry/opentelemetry-go/security/advisories/GHSA-q7gw-24fr-5p69"}}}
{"finding":{"osv":"GHSA-q7gw-24fr-5p69","fixed_version":"v1.47.0","trace":[{"module":"go.opentelemetry.io/otel/trace","version":"v1.44.0"}]}}
`, out.String())

	// The pgx advisories name no patched version; they are reported on stderr.
	assert.Equal(t, `skipping GHSA-j38q-h3ph-jjpq for github.com/jackc/pgx/v5@v5.11.0: it names no patched version: https://github.com/jackc/pgx/security/advisories/GHSA-j38q-h3ph-jjpq
skipping GHSA-95fj-3fxw-g62g for github.com/jackc/pgx/v5@v5.11.0: it names no patched version: https://github.com/jackc/pgx/security/advisories/GHSA-95fj-3fxw-g62g
`, errOut.String())
}

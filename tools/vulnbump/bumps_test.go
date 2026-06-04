package main

import (
	_ "embed"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/govulncheck.json
var sampleOutput string

func TestParseBumps(t *testing.T) {
	bumps, err := parseBumps(strings.NewReader(sampleOutput))
	require.NoError(t, err)

	// golang.org/x/crypto takes the higher of its two fixed versions (v0.52.0
	// over v0.51.0), advisories are deduplicated and sorted, the stdlib finding
	// is dropped, and non-finding messages are ignored. CVE aliases come from
	// the "osv" messages; GO-2026-5021's osv has only a GHSA alias, so it keeps
	// an empty CVE and later falls back to the Go ID as its label.
	assert.Equal(t, []bump{
		{
			Module:       "golang.org/x/crypto",
			FixedVersion: "v0.52.0",
			Advisories: []advisory{
				{ID: "GO-2026-5016", CVE: "CVE-2026-39827"},
				{ID: "GO-2026-5021"},
			},
		},
		{
			Module:       "golang.org/x/net",
			FixedVersion: "v0.55.0",
			Advisories: []advisory{
				{ID: "GO-2026-5026", CVE: "CVE-2026-39821"},
			},
		},
	}, bumps)
}

func TestParseBumpsEmpty(t *testing.T) {
	bumps, err := parseBumps(strings.NewReader(""))
	require.NoError(t, err)
	assert.Empty(t, bumps)
}

func TestParseBumpsInvalid(t *testing.T) {
	_, err := parseBumps(strings.NewReader("not json"))
	require.Error(t, err)
}

func TestRenderSummary(t *testing.T) {
	bumps := []bump{
		{
			Module:       "golang.org/x/crypto",
			FixedVersion: "v0.52.0",
			Advisories: []advisory{
				{ID: "GO-2026-5016", CVE: "CVE-2026-39827"},
				{ID: "GO-2026-5021"}, // no CVE assigned: falls back to the Go ID
			},
		},
	}

	// Reference-style links keep the list line short; definitions follow in a
	// block. The CVE is the visible label, linking to the Go advisory page.
	want := "- golang.org/x/crypto → v0.52.0 (fixes [CVE-2026-39827], [GO-2026-5021])\n" +
		"\n" +
		"[CVE-2026-39827]: https://pkg.go.dev/vuln/GO-2026-5016\n" +
		"[GO-2026-5021]: https://pkg.go.dev/vuln/GO-2026-5021\n"
	assert.Equal(t, want, renderSummary(bumps))
}

func TestRenderSummaryEmpty(t *testing.T) {
	assert.Empty(t, renderSummary(nil))
}

func TestFirstCVE(t *testing.T) {
	assert.Equal(t, "CVE-2026-39827", firstCVE([]string{"CVE-2026-39827", "GHSA-xxxx-xxxx-xxxx"}))
	assert.Equal(t, "CVE-2026-39827", firstCVE([]string{"GHSA-xxxx-xxxx-xxxx", "CVE-2026-39827"}))
	assert.Empty(t, firstCVE([]string{"GHSA-xxxx-xxxx-xxxx"}))
	assert.Empty(t, firstCVE(nil))
}

func TestAdvisoryLabelFallsBackToID(t *testing.T) {
	assert.Equal(t, "CVE-2026-39827", advisory{ID: "GO-2026-5016", CVE: "CVE-2026-39827"}.label())
	assert.Equal(t, "GO-2026-5021", advisory{ID: "GO-2026-5021"}.label())
}

// TestRenderSummaryNoCVE checks the no-alias path renders the Go ID as both the
// link text and the reference definition.
func TestRenderSummaryNoCVE(t *testing.T) {
	bumps := []bump{
		{
			Module:       "golang.org/x/crypto",
			FixedVersion: "v0.52.0",
			Advisories:   []advisory{{ID: "GO-2026-5021"}},
		},
	}

	want := "- golang.org/x/crypto → v0.52.0 (fixes [GO-2026-5021])\n" +
		"\n" +
		"[GO-2026-5021]: https://pkg.go.dev/vuln/GO-2026-5021\n"
	assert.Equal(t, want, renderSummary(bumps))
}

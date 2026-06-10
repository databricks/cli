package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"

	"golang.org/x/mod/semver"
)

// stdlibModule is govulncheck's module name for the Go standard library.
// Standard-library fixes map to a toolchain version and are handled by the
// separate "Bump Go toolchain" workflow, so we skip them here.
const stdlibModule = "stdlib"

// advisoryBaseURL is the canonical advisory page for a GO-YYYY-NNNN ID. The
// page cross-links the CVE alias and the per-package fixed version.
const advisoryBaseURL = "https://pkg.go.dev/vuln/"

// advisory identifies a Go vulnerability and its CVE alias.
type advisory struct {
	ID  string // GO-YYYY-NNNN, always present.
	CVE string // CVE-YYYY-NNNN, empty until one is assigned (it can lag the Go advisory by days).
}

// label is the link text for an advisory: the CVE when known, else the Go ID.
func (a advisory) label() string {
	if a.CVE != "" {
		return a.CVE
	}
	return a.ID
}

// bump is a single dependency upgrade: the highest fixed version across all
// advisories affecting a module, plus the advisories it resolves.
type bump struct {
	Module       string
	FixedVersion string
	Advisories   []advisory
}

// govulncheckFinding mirrors the "finding" message emitted by
// `govulncheck -format json`. See https://pkg.go.dev/golang.org/x/vuln/internal/govulncheck#Finding.
type govulncheckFinding struct {
	OSV          string `json:"osv"`
	FixedVersion string `json:"fixed_version"`
	Trace        []struct {
		Module string `json:"module"`
	} `json:"trace"`
}

// govulncheckOSV mirrors the "osv" message emitted by `govulncheck -format json`,
// which carries the full advisory record including its CVE aliases.
type govulncheckOSV struct {
	ID      string   `json:"id"`
	Aliases []string `json:"aliases"`
}

// parseBumps reads the concatenated JSON stream from `govulncheck -format json`
// and reduces it to one bump per module, choosing the highest fixed version and
// collecting the advisories it resolves. The standard library is excluded.
func parseBumps(r io.Reader) ([]bump, error) {
	// One accumulator per module: the highest fixed version seen and the set of
	// advisories resolved by upgrading to it.
	type acc struct {
		version    string
		advisories map[string]struct{}
	}
	byModule := map[string]*acc{}
	// govulncheck emits a separate "osv" message per advisory; collect the CVE
	// aliases so findings (which carry only the Go ID) can be labelled with it.
	cveByID := map[string]string{}

	dec := json.NewDecoder(r)
	for {
		var msg struct {
			Finding *govulncheckFinding `json:"finding"`
			OSV     *govulncheckOSV     `json:"osv"`
		}
		err := dec.Decode(&msg)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("decode govulncheck output: %w", err)
		}

		if o := msg.OSV; o != nil {
			cveByID[o.ID] = firstCVE(o.Aliases)
		}

		f := msg.Finding
		if f == nil || f.FixedVersion == "" || len(f.Trace) == 0 {
			continue
		}
		// trace[0] is the most specific frame; in module scan mode it carries
		// the vulnerable module itself.
		module := f.Trace[0].Module
		if module == "" || module == stdlibModule {
			continue
		}

		a := byModule[module]
		if a == nil {
			a = &acc{advisories: map[string]struct{}{}}
			byModule[module] = a
		}
		if a.version == "" || semver.Compare(f.FixedVersion, a.version) > 0 {
			a.version = f.FixedVersion
		}
		a.advisories[f.OSV] = struct{}{}
	}

	bumps := make([]bump, 0, len(byModule))
	for module, a := range byModule {
		var advisories []advisory
		for _, id := range slices.Sorted(maps.Keys(a.advisories)) {
			advisories = append(advisories, advisory{ID: id, CVE: cveByID[id]})
		}
		bumps = append(bumps, bump{
			Module:       module,
			FixedVersion: a.version,
			Advisories:   advisories,
		})
	}
	slices.SortFunc(bumps, func(a, b bump) int {
		return strings.Compare(a.Module, b.Module)
	})
	return bumps, nil
}

// firstCVE returns the first CVE alias in the list, or "" if there is none.
func firstCVE(aliases []string) string {
	for _, a := range aliases {
		if strings.HasPrefix(a, "CVE-") {
			return a
		}
	}
	return ""
}

// renderSummary formats the bumps as Markdown list items for the PR body. It
// uses reference-style links so the list lines stay short instead of wrapping,
// with the advisory link definitions collected in a block at the end.
func renderSummary(bumps []bump) string {
	var list, refs strings.Builder
	seen := map[string]struct{}{}

	for _, bump := range bumps {
		labels := make([]string, len(bump.Advisories))
		for i, adv := range bump.Advisories {
			labels[i] = "[" + adv.label() + "]"
			if _, ok := seen[adv.label()]; ok {
				continue
			}
			seen[adv.label()] = struct{}{}
			fmt.Fprintf(&refs, "[%s]: %s%s\n", adv.label(), advisoryBaseURL, adv.ID)
		}
		fmt.Fprintf(&list, "- %s → %s (fixes %s)\n",
			bump.Module, bump.FixedVersion, strings.Join(labels, ", "))
	}

	if refs.Len() == 0 {
		return list.String()
	}
	return list.String() + "\n" + refs.String()
}

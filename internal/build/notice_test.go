package build

import (
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/mod/modfile"
)

// moduleToGitHub maps non-GitHub go.mod paths to their GitHub org/repo slug.
var moduleToGitHub = map[string]string{
	"gopkg.in/ini.v1":    "go-ini/ini",
	"go.yaml.in/yaml/v3": "yaml/go-yaml",
	"dario.cat/mergo":    "darccio/mergo",
}

// Modules excluded from NOTICE requirements (Databricks-owned).
var noticeExclude = map[string]bool{
	"github.com/databricks/databricks-sdk-go": true,
}

// Additional entries required in the NOTICE file that are not direct go.mod
// dependencies (e.g. bundled binaries).
var noticeExtra = map[string][]string{
	"hashicorp/terraform": {"MPL-2.0"},
}

// Expected order of license sections in the NOTICE file.
var expectedSectionOrder = []string{
	"Apache-2.0",
	"MPL-2.0",
	"BSD-2-Clause",
	"BSD-3-Clause",
	"MIT",
}

var headerToSPDX = map[string]string{
	"apache 2.0":     "Apache-2.0",
	"mpl 2.0":        "MPL-2.0",
	"bsd (2-clause)": "BSD-2-Clause",
	"bsd (3-clause)": "BSD-3-Clause",
	"mit":            "MIT",
}

var (
	githubSlugRe    = regexp.MustCompile(`github\.com/([^/\s]+/[^/\s]+)`)
	sectionHeaderRe = regexp.MustCompile(`(?i)licensed under the (.+?) license`)
)

func githubSlugFromModule(modPath string) string {
	if slug, ok := moduleToGitHub[modPath]; ok {
		return slug
	}
	if strings.HasPrefix(modPath, "github.com/") {
		parts := strings.SplitN(modPath, "/", 4)
		if len(parts) >= 3 {
			return parts[1] + "/" + parts[2]
		}
	}
	if after, ok := strings.CutPrefix(modPath, "golang.org/x/"); ok {
		return "golang/" + after
	}
	return ""
}

// parseNoticeSections parses the NOTICE file into a map from SPDX license
// identifier to the list of GitHub org/repo slugs in that section.
// It also returns the order in which sections appear.
func parseNoticeSections(content string) (map[string][]string, []string) {
	sections := map[string][]string{}
	var order []string
	var currentSPDX string
	var block []string

	flush := func() {
		if currentSPDX != "" && len(block) > 0 {
			text := strings.Join(block, "\n")
			if m := githubSlugRe.FindStringSubmatch(text); m != nil {
				sections[currentSPDX] = append(sections[currentSPDX], m[1])
			}
		}
		block = nil
	}

	for line := range strings.SplitSeq(content, "\n") {
		if m := sectionHeaderRe.FindStringSubmatch(line); m != nil {
			flush()
			key := strings.ToLower(strings.TrimSpace(m[1]))
			if spdx, ok := headerToSPDX[key]; ok {
				currentSPDX = spdx
				order = append(order, spdx)
			}
			continue
		}

		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.Trim(trimmed, "-—") == "" {
			flush()
			continue
		}

		if currentSPDX != "" {
			block = append(block, line)
		}
	}
	flush()

	return sections, order
}

func extractLicenseComment(r *modfile.Require) string {
	for _, c := range r.Syntax.Suffix {
		text := strings.TrimPrefix(c.Token, "//")
		text = strings.TrimSpace(text)
		if text != "indirect" {
			return text
		}
	}
	return ""
}

func TestNoticeFileCompleteness(t *testing.T) {
	goModBytes, err := os.ReadFile("../../go.mod")
	require.NoError(t, err)
	modFile, err := modfile.Parse("../../go.mod", goModBytes, nil)
	require.NoError(t, err)

	// Build expected: license → sorted list of slugs.
	expected := map[string][]string{}
	for _, r := range modFile.Require {
		if r.Indirect || noticeExclude[r.Mod.Path] {
			continue
		}
		slug := githubSlugFromModule(r.Mod.Path)
		if slug == "" {
			assert.Failf(t, r.Mod.Path, "cannot map to GitHub slug for NOTICE verification")
			continue
		}
		license := extractLicenseComment(r)
		if license == "" {
			continue // license_test.go catches missing comments
		}
		ids, _ := parseSPDXExpression(license)
		for _, id := range ids {
			expected[id] = append(expected[id], slug)
		}
	}
	for slug, licenses := range noticeExtra {
		for _, id := range licenses {
			expected[id] = append(expected[id], slug)
		}
	}
	for k := range expected {
		slices.Sort(expected[k])
	}

	// Parse NOTICE file.
	noticeBytes, err := os.ReadFile("../../NOTICE")
	require.NoError(t, err)
	actual, sectionOrder := parseNoticeSections(string(noticeBytes))
	for k := range actual {
		slices.Sort(actual[k])
	}

	// Check section order.
	assert.Equal(t, expectedSectionOrder, sectionOrder, "NOTICE section order")

	// Check entries per section.
	for _, license := range expectedSectionOrder {
		assert.Equal(t, expected[license], actual[license], "NOTICE %s section", license)
	}
}

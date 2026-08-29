package build

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/mod/modfile"
)

// Allowlist of SPDX license identifiers we accept for direct dependencies.
// See https://spdx.org/licenses/ for the full list.
var spdxLicenses = map[string]bool{
	"Apache-2.0":   true,
	"BSD-2-Clause": true,
	"BSD-3-Clause": true,
	"MIT":          true,
	"MPL-2.0":      true,
}

// parseSPDXExpression validates that expr is a valid SPDX license expression
// composed of allowed identifiers joined by AND/OR operators.
// Returns the list of license identifiers found, or an error string.
func parseSPDXExpression(expr string) ([]string, string) {
	tokens := strings.Fields(expr)
	if len(tokens) == 0 {
		return nil, "empty expression"
	}

	var ids []string
	expectID := true
	for _, tok := range tokens {
		if expectID {
			if !spdxLicenses[tok] {
				return nil, tok + " is not an allowed SPDX license identifier; allowed: " + allowedList()
			}
			ids = append(ids, tok)
			expectID = false
		} else {
			if tok != "AND" && tok != "OR" {
				return nil, tok + " unexpected; expected AND or OR operator"
			}
			expectID = true
		}
	}

	if expectID {
		return nil, "expression ends with an operator"
	}

	return ids, ""
}

func allowedList() string {
	var out []string
	for k := range spdxLicenses {
		out = append(out, k)
	}
	return strings.Join(out, ", ")
}

func TestRequireSPDXLicenseComment(t *testing.T) {
	b, err := os.ReadFile("../../go.mod")
	require.NoError(t, err)

	modFile, err := modfile.Parse("../../go.mod", b, nil)
	require.NoError(t, err)

	for _, r := range modFile.Require {
		if r.Indirect {
			continue
		}

		// Find the license comment in suffix comments (excluding "indirect").
		var license string
		for _, c := range r.Syntax.Suffix {
			text := strings.TrimPrefix(c.Token, "//")
			text = strings.TrimSpace(text)
			if text == "indirect" {
				continue
			}
			license = text
		}

		if license == "" {
			assert.Failf(t, r.Mod.Path, "missing SPDX license comment; add one like: // MIT")
			continue
		}

		_, errMsg := parseSPDXExpression(license)
		if errMsg != "" {
			assert.Failf(t, r.Mod.Path, "license comment %q: %s", license, errMsg)
		}
	}
}

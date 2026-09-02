package dynvar

import (
	"fmt"
	"regexp"

	"github.com/databricks/cli/libs/dyn"
)

// BaseVarDef matches a single dot-separated path segment in ${...} references.
// A segment may have leading underscores before the first letter, and may contain
// letters, digits, hyphens and underscores after that.
// Keep in sync with _base_var_def in python/databricks/bundles/core/_transform.py.
// Behavioral parity is enforced by testdata/reference_vectors.json.
const BaseVarDef = `_*\p{L}+([-_]*[\p{L}\p{N}]+)*`

var re = regexp.MustCompile(fmt.Sprintf(`\$\{(%s(\.%s(\[[0-9]+\])*)*(\[[0-9]+\])*)\}`, BaseVarDef, BaseVarDef))

// Ref represents a variable reference.
// It is a string [dyn.Value] contained in a larger [dyn.Value].
// Its path within the containing [dyn.Value] is also stored.
type Ref struct {
	// Original value.
	Value dyn.Value

	// String value in the original [dyn.Value].
	Str string

	// Matches of the variable reference in the string.
	Matches [][]string
}

// NewRef returns a new Ref if the given [dyn.Value] contains a string
// with one or more variable references. It returns false if the given
// [dyn.Value] does not contain variable references.
//
// A reference preceded by "$" is treated as an escaped literal and is not
// included in the returned Ref. For example, "$${foo}" contains no references
// because the "${foo}" part is escaped by the leading "$".
//
// Examples of a valid variable references:
//   - "${a.b}"
//   - "${a.b.c}"
//   - "${a.b[0].c}"
//   - "${a} ${b} ${c}"
func NewRef(v dyn.Value) (Ref, bool) {
	s, ok := v.AsString()
	if !ok {
		return Ref{}, false
	}

	// Find all matches with their byte positions so we can detect escaped references.
	indices := re.FindAllStringSubmatchIndex(s, -1)
	if len(indices) == 0 {
		return Ref{}, false
	}

	// Convert position-based matches to string matches, filtering out references
	// escaped with a leading "$" (e.g. "$${foo}" should not match "${foo}").
	var m [][]string
	for _, idx := range indices {
		if idx[0] > 0 && s[idx[0]-1] == '$' {
			continue
		}
		match := make([]string, len(idx)/2)
		for i := range len(idx) / 2 {
			if idx[2*i] >= 0 {
				match[i] = s[idx[2*i] : idx[2*i+1]]
			}
		}
		m = append(m, match)
	}

	if len(m) == 0 {
		return Ref{}, false
	}

	return Ref{
		Value:   v,
		Str:     s,
		Matches: m,
	}, true
}

// IsPure returns true if the variable reference contains a single
// variable reference and nothing more. We need this so we can
// interpolate values of non-string types (i.e. it can be substituted).
func (v Ref) IsPure() bool {
	// Need single match, equal to the incoming string.
	if len(v.Matches) == 0 || len(v.Matches[0]) == 0 {
		panic("invalid variable reference; expect at least one match")
	}
	return v.Matches[0][0] == v.Str
}

func (v Ref) References() []string {
	var out []string
	for _, m := range v.Matches {
		out = append(out, m[1])
	}
	return out
}

func IsPureVariableReference(s string) bool {
	return len(s) > 0 && re.FindString(s) == s
}

func ContainsVariableReference(s string) bool {
	return re.MatchString(s)
}

// If s is a pure variable reference, this function returns the corresponding
// dyn.Path. Otherwise, it returns false.
func PureReferenceToPath(s string) (dyn.Path, bool) {
	ref, ok := NewRef(dyn.V(s))
	if !ok {
		return nil, false
	}

	if !ref.IsPure() {
		return nil, false
	}

	p, err := dyn.NewPathFromString(ref.References()[0])
	if err != nil {
		return nil, false
	}

	return p, true
}

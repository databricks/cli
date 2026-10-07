package structvar

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/databricks/cli/libs/structs/structpath"
)

// BaseVarDef matches a single dot-separated path segment in ${...} references.
// A segment may have leading underscores before the first letter, and may contain
// letters, digits, hyphens and underscores after that.
// Keep in sync with _base_var_def in python/databricks/bundles/core/_transform.py.
// Behavioral parity is enforced by testdata/reference_vectors.json.
const BaseVarDef = `_*\p{L}+([-_]*[\p{L}\p{N}]+)*`

var re = regexp.MustCompile(fmt.Sprintf(`\$\{(%s(\.%s(\[[0-9]+\])*)*(\[[0-9]+\])*)\}`, BaseVarDef, BaseVarDef))

// Ref represents the variable references in a string.
type Ref struct {
	// The string.
	Str string

	// Matches of the variable reference in the string.
	Matches [][]string

	// Byte offsets [start, end) of each entry in Matches within Str. Interpolation
	// must substitute by offset: an escaped occurrence of the same reference text
	// may appear earlier in Str, and a search-based replacement would hit that one.
	Spans [][2]int
}

// NewRef returns a new Ref if s contains one or more variable references.
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
func NewRef(s string) (Ref, bool) {
	// Find all matches with their byte positions so we can detect escaped references.
	indices := re.FindAllStringSubmatchIndex(s, -1)
	if len(indices) == 0 {
		return Ref{}, false
	}

	// Convert position-based matches to string matches, filtering out references
	// escaped with a leading "$" (e.g. "$${foo}" should not match "${foo}").
	var m [][]string
	var spans [][2]int
	for _, idx := range indices {
		if isEscapedAt(s, idx[0]) {
			continue
		}
		match := make([]string, len(idx)/2)
		for i := range len(idx) / 2 {
			if idx[2*i] >= 0 {
				match[i] = s[idx[2*i]:idx[2*i+1]]
			}
		}
		m = append(m, match)
		spans = append(spans, [2]int{idx[0], idx[1]})
	}

	if len(m) == 0 {
		return Ref{}, false
	}

	return Ref{
		Str:     s,
		Matches: m,
		Spans:   spans,
	}, true
}

// isEscapedAt reports whether the reference starting at byte offset start in s is
// escaped by an immediately preceding "$".
func isEscapedAt(s string, start int) bool {
	return start > 0 && s[start-1] == '$'
}

// Unescape turns escaped references into literal ones ("$${x}" -> "${x}"). Call it
// when building a value to send to the API: the extra "$" exists only to hide the
// reference from bundle variable resolution.
func Unescape(s string) string {
	return strings.ReplaceAll(s, "$${", "${")
}

// ReplaceRef substitutes every unescaped occurrence of reference (a full "${...}"
// string) in s with value. Escaped occurrences are left as they are, so
// strings.ReplaceAll is not a valid substitute: it would also rewrite the literal.
func ReplaceRef(s, reference, value string) string {
	var sb strings.Builder
	for pos := 0; ; {
		i := strings.Index(s[pos:], reference)
		if i < 0 {
			sb.WriteString(s[pos:])
			return sb.String()
		}
		start := pos + i
		sb.WriteString(s[pos:start])
		if isEscapedAt(s, start) {
			sb.WriteString(reference)
		} else {
			sb.WriteString(value)
		}
		pos = start + len(reference)
	}
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

// ContainsVariableReference reports whether s contains at least one reference that
// is not escaped. Callers use it to decide whether a string still needs resolution,
// so an escaped "$${foo}" must not count: it is a literal, and treating it as
// pending leaves the value permanently unresolved.
func ContainsVariableReference(s string) bool {
	for _, loc := range re.FindAllStringIndex(s, -1) {
		if !isEscapedAt(s, loc[0]) {
			return true
		}
	}
	return false
}

// PureReferenceToPath returns the path s refers to if s is a pure variable reference.
func PureReferenceToPath(s string) (*structpath.PathNode, bool) {
	ref, ok := NewRef(s)
	if !ok || !ref.IsPure() {
		return nil, false
	}

	p, err := structpath.ParsePath(ref.References()[0])
	if err != nil {
		return nil, false
	}

	return p, true
}

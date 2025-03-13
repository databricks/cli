package dynvar

import (
	"fmt"
	"regexp"

	"github.com/databricks/cli/libs/dyn"
)

var (
	// !!! Should be in sync with _variable_regex in Python code.
	// !!!
	// !!! See experimental/python/databricks/bundles/core/_transform.py
	baseVarDef = `[a-zA-Z]+([-_]*[a-zA-Z0-9]+)*`
	re         = regexp.MustCompile(fmt.Sprintf(`\$\{(%s(\.%s(\[[0-9]+\])*)*(\[[0-9]+\])*)\}`, baseVarDef, baseVarDef))
)

// ref represents a variable reference.
// It is a string [dyn.Value] contained in a larger [dyn.Value].
// Its path within the containing [dyn.Value] is also stored.
type ref struct {
	// Original value.
	value dyn.Value

	// String value in the original [dyn.Value].
	str string

	// Matches of the variable reference in the string.
	matches [][]string
}

// newRef returns a new ref if the given [dyn.Value] contains a string
// with one or more variable references. It returns false if the given
// [dyn.Value] does not contain variable references.
//
// Examples of a valid variable references:
//   - "${a.b}"
//   - "${a.b.c}"
//   - "${a.b[0].c}"
//   - "${a} ${b} ${c}"
func newRef(v dyn.Value) (ref, bool) {
	s, ok := v.AsString()
	if !ok {
		return ref{}, false
	}

	// Check if the string contains any variable references.
	m := re.FindAllStringSubmatch(s, -1)
	if len(m) == 0 {
		return ref{}, false
	}

	return ref{
		value:   v,
		str:     s,
		matches: m,
	}, true
}

// isPure returns true if the variable reference contains a single
// variable reference and nothing more. We need this so we can
// interpolate values of non-string types (i.e. it can be substituted).
func (v ref) isPure() bool {
	// Need single match, equal to the incoming string.
	if len(v.matches) == 0 || len(v.matches[0]) == 0 {
		panic("invalid variable reference; expect at least one match")
	}
	return v.matches[0][0] == v.str
}

func (v ref) references() []string {
	var out []string
	for _, m := range v.matches {
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
	ref, ok := newRef(dyn.V(s))
	if !ok {
		return nil, false
	}

	if !ref.isPure() {
		return nil, false
	}

	p, err := dyn.NewPathFromString(ref.references()[0])
	if err != nil {
		return nil, false
	}

	return p, true
}

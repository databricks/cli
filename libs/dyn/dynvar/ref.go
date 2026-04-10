package dynvar

import (
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar/interpolation"
)

// Ref represents a variable reference.
// It is a string [dyn.Value] contained in a larger [dyn.Value].
type Ref struct {
	// Original value.
	Value dyn.Value

	// String value in the original [dyn.Value].
	Str string

	// Parsed tokens from the interpolation parser.
	Tokens []interpolation.Token
}

// NewRef returns a new Ref if the given [dyn.Value] contains a string
// with one or more variable references. It returns false if the given
// [dyn.Value] does not contain variable references or if parsing fails.
//
// Examples of a valid variable references:
//   - "${a.b}"
//   - "${a.b.c}"
//   - "${a.b[0].c}"
//   - "${a} ${b} ${c}"
func NewRef(v dyn.Value) (Ref, bool) {
	ref, ok, _ := newRef(v)
	return ref, ok
}

// NewRefWithDiagnostics returns a new Ref along with any diagnostics.
// Parse errors for malformed references (e.g. "${foo.bar-}") are returned
// as warnings. The second return value is false when no valid references
// are found (either no references at all, or a parse error occurred).
func NewRefWithDiagnostics(v dyn.Value) (Ref, bool, diag.Diagnostics) {
	return newRef(v)
}

func newRef(v dyn.Value) (Ref, bool, diag.Diagnostics) {
	s, ok := v.AsString()
	if !ok {
		return Ref{}, false, nil
	}

	tokens, err := interpolation.Parse(s)
	if err != nil {
		// Return parse error as a warning diagnostic.
		return Ref{}, false, diag.Diagnostics{{
			Severity:  diag.Warning,
			Summary:   err.Error(),
			Locations: v.Locations(),
		}}
	}

	// Check if any token is a variable reference.
	hasRef := false
	for _, t := range tokens {
		if t.Kind == interpolation.TokenRef {
			hasRef = true
			break
		}
	}

	if !hasRef {
		return Ref{}, false, nil
	}

	return Ref{
		Value:  v,
		Str:    s,
		Tokens: tokens,
	}, true, nil
}

// IsPure returns true if the variable reference contains a single
// variable reference and nothing more. We need this so we can
// interpolate values of non-string types (i.e. it can be substituted).
func (v Ref) IsPure() bool {
	return len(v.Tokens) == 1 && v.Tokens[0].Kind == interpolation.TokenRef
}

// References returns the path strings of all variable references.
func (v Ref) References() []string {
	var out []string
	for _, t := range v.Tokens {
		if t.Kind == interpolation.TokenRef {
			out = append(out, t.Value)
		}
	}
	return out
}

// IsPureVariableReference returns true if s is a single variable reference
// with no surrounding text.
func IsPureVariableReference(s string) bool {
	if len(s) == 0 {
		return false
	}
	tokens, err := interpolation.Parse(s)
	if err != nil {
		return false
	}
	return len(tokens) == 1 && tokens[0].Kind == interpolation.TokenRef
}

// ContainsVariableReference returns true if s contains at least one variable reference.
func ContainsVariableReference(s string) bool {
	tokens, err := interpolation.Parse(s)
	if err != nil {
		return false
	}
	for _, t := range tokens {
		if t.Kind == interpolation.TokenRef {
			return true
		}
	}
	return false
}

// PureReferenceToPath returns the corresponding [dyn.Path] if s is a pure
// variable reference. Otherwise, it returns false.
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

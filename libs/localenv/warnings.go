package localenv

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
)

// depSpecRe splits a dependency string into its package name and the remainder
// (e.g. "pyarrow~=19.0" -> "pyarrow", "~=19.0"). It stops the name at the first
// PEP 508 separator; splitDepSpec handles the extras and markers that may lead the
// remainder.
var depSpecRe = regexp.MustCompile(`^([A-Za-z0-9._-]+)\s*(.*)$`)

// singleClauseRe parses one version clause: an operator and a dotted numeric
// release. Pre/post/dev suffixes and wildcards are not modeled — a clause we
// cannot parse this simply is treated as "unknown" and never yields a conflict.
var singleClauseRe = regexp.MustCompile(`^(>=|<=|==|~=|!=|<|>)?\s*([0-9]+(?:\.[0-9]+)*)`)

// splitDepSpec returns the normalized package name and the version specifier
// portion of a dependency string. ok is false when there is no recognizable name,
// or when the requirement carries an environment marker: a marker makes the
// dependency conditional on the resolving interpreter, which we do not evaluate,
// so comparing its range could flag a pin that never applies.
func splitDepSpec(dep string) (name, spec string, ok bool) {
	dep = strings.TrimSpace(dep)
	if strings.Contains(dep, ";") {
		return "", "", false
	}
	m := depSpecRe.FindStringSubmatch(dep)
	if m == nil {
		return "", "", false
	}
	// Extras select optional features and never narrow the version range, so drop
	// them to expose the specifier underneath ("pkg[a,b]==1.0" -> "==1.0").
	spec = strings.TrimSpace(m[2])
	if strings.HasPrefix(spec, "[") {
		if i := strings.Index(spec, "]"); i >= 0 {
			spec = strings.TrimSpace(spec[i+1:])
		}
	}
	return normalizePackageName(m[1]), spec, true
}

// userPyprojectTOML is a permissive view of the fields detectMergeWarnings reads
// from the *user's* pyproject.toml.
//
// It deliberately does not reuse pyprojectTOML: that struct decodes the
// Databricks-owned constraint artifact, whose shape we control, whereas a user's
// dependency group may legitimately hold PEP 735 table entries such as
// {include-group = "test"} alongside requirement strings. Decoding dev as []any
// (and declaring only the fields actually read) keeps one unrelated entry from
// failing the whole document and silently dropping every warning.
type userPyprojectTOML struct {
	Project struct {
		RequiresPython string   `toml:"requires-python"`
		Dependencies   []string `toml:"dependencies"`
	} `toml:"project"`
	DependencyGroups struct {
		Dev []any `toml:"dev"`
	} `toml:"dependency-groups"`
}

// devRequirements returns the requirement strings of a dependency group, skipping
// PEP 735 table entries (include-group) that are not requirements themselves.
func devRequirements(entries []any) []string {
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		if s, ok := e.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// detectMergeWarnings compares a user's existing pyproject.toml against the
// fetched env constraints and returns the categorical warnings the merge phase
// surfaces (spec §6 warnings[]). It is a read-only comparison run alongside the
// merge; MergeManaged still owns the actual byte edits.
//
// It is best-effort: unparseable input yields no warnings rather than an error,
// because a warning is advisory and must never fail the run. The user's file is
// decoded through userPyprojectTOML rather than the artifact's stricter
// pyprojectTOML, so a legitimate PEP 735 table entry cannot suppress the checks
// that do not depend on it. Greenfield projects (no pre-existing content) produce
// nothing — there is nothing of the user's to override. Warnings are deterministic
// and ordered (requires-python, then databricks-connect, then constraint conflicts
// sorted by package) so goldens are stable.
func detectMergeWarnings(userPyproject []byte, c Constraints) []Warning {
	if len(userPyproject) == 0 {
		return nil
	}
	var p userPyprojectTOML
	if err := toml.Unmarshal(userPyproject, &p); err != nil {
		return nil
	}

	var warnings []Warning

	// The user pinned a requires-python that differs from the env's pin; the merge
	// replaces it with the managed value.
	if up := strings.TrimSpace(p.Project.RequiresPython); up != "" && c.RequiresPython != "" && up != strings.TrimSpace(c.RequiresPython) {
		warnings = append(warnings, Warning{
			Code:    WarnRequiresPythonOverridden,
			Message: fmt.Sprintf("requires-python %q was replaced by the environment's %q", up, c.RequiresPython),
		})
	}

	// The user pinned databricks-connect to something other than the env's pin;
	// the merge replaces it. Only meaningful in default mode (c.DatabricksConnect
	// is empty in constraints-only, where the dev group is left untouched).
	if c.DatabricksConnect != "" {
		for _, entry := range devRequirements(p.DependencyGroups.Dev) {
			if !isDatabricksConnectDep(entry) {
				continue
			}
			if strings.TrimSpace(entry) != strings.TrimSpace(c.DatabricksConnect) {
				warnings = append(warnings, Warning{
					Code:    WarnDBConnectPinOverridden,
					Message: fmt.Sprintf("databricks-connect %q was replaced by the environment's %q", strings.TrimSpace(entry), c.DatabricksConnect),
				})
			}
			break
		}
	}

	warnings = append(warnings, constraintConflicts(p.Project.Dependencies, c.ConstraintDeps)...)
	return warnings
}

// constraintConflicts flags each user [project].dependencies pin that the env's
// constraint-dependencies also constrains to a provably non-overlapping version.
// It is deliberately conservative — it only fires when the two ranges are
// provably disjoint (see rangesDisjoint); an ambiguous pair yields nothing, so a
// false "conflict" is never reported. Results are sorted by package name for
// deterministic output.
func constraintConflicts(userDeps, envConstraints []string) []Warning {
	if len(userDeps) == 0 || len(envConstraints) == 0 {
		return nil
	}
	// Index the env constraints by normalized package name. Multiple entries for one
	// package compose as a conjunction, so they are joined with "," rather than
	// letting the last one win: parseClause treats a multi-clause spec as an unknown
	// range, which keeps the outcome independent of artifact ordering instead of
	// deciding against an arbitrary subset of the clauses.
	envByName := make(map[string]string, len(envConstraints))
	for _, ec := range envConstraints {
		name, spec, ok := splitDepSpec(ec)
		if !ok || spec == "" {
			continue
		}
		if prev, dup := envByName[name]; dup {
			spec = prev + "," + spec
		}
		envByName[name] = spec
	}
	if len(envByName) == 0 {
		return nil
	}

	var warnings []Warning
	for _, ud := range userDeps {
		name, userSpec, ok := splitDepSpec(ud)
		if !ok || userSpec == "" {
			continue
		}
		envSpec, ok := envByName[name]
		if !ok {
			continue
		}
		if rangesDisjoint(userSpec, envSpec) {
			warnings = append(warnings, Warning{
				Code:    WarnUserConstraintConflict,
				Message: fmt.Sprintf("dependency %q conflicts with the environment constraint %q", strings.TrimSpace(ud), name+envSpec),
			})
		}
	}
	// Sort by package name (embedded in Message after "dependency ") for stable
	// output; the count and code are what the contract carries, order is cosmetic.
	sortWarningsByMessage(warnings)
	return warnings
}

// sortWarningsByMessage orders warnings by their Message so multiple conflicts
// render deterministically (map iteration and slice order upstream are unstable).
func sortWarningsByMessage(w []Warning) {
	slices.SortStableFunc(w, func(a, b Warning) int { return strings.Compare(a.Message, b.Message) })
}

// clause is a parsed single version clause: an operator and a numeric release.
type clause struct {
	op  string
	rel []int
}

// parseClause parses the first "<op><release>" out of a specifier. ok is false
// when the specifier has multiple comma clauses or cannot be parsed simply — both
// are treated as "unknown range", which never produces a conflict.
func parseClause(spec string) (clause, bool) {
	spec = strings.TrimSpace(spec)
	if spec == "" || strings.Contains(spec, ",") {
		return clause{}, false
	}
	m := singleClauseRe.FindStringSubmatch(spec)
	if m == nil {
		return clause{}, false
	}
	// Reject anything trailing the numeric release (wildcards, pre/post/dev tags):
	// modeling those correctly is out of scope, and guessing risks a false conflict.
	if strings.TrimSpace(spec[len(m[0]):]) != "" {
		return clause{}, false
	}
	op := m[1]
	if op == "" {
		op = "==" // a bare version in dependencies means an exact pin
	}
	var rel []int
	for part := range strings.SplitSeq(m[2], ".") {
		n, err := strconv.Atoi(part)
		if err != nil {
			return clause{}, false
		}
		rel = append(rel, n)
	}
	// PEP 440 requires at least two release segments after "~=" (it expands to
	// ">=X.Y, ==X.*", which needs a segment to hold fixed). A single-segment base
	// has no defined range, and compatibleReleaseContains reports "not contained"
	// for it — which the disjointness callers would read as a proof of disjointness
	// and report a conflict. Refuse to parse it so it stays an unknown range.
	if op == "~=" && len(rel) < 2 {
		return clause{}, false
	}
	return clause{op: op, rel: rel}, true
}

// rangesDisjoint reports whether two version specifiers provably share no
// satisfying version. It is intentionally partial: it returns true only for
// operator pairs it can decide with certainty (== / ~= / >= / > / <= / <). Any
// pair it cannot decide — a "!=", a multi-clause range, an unparseable release —
// returns false, so an uncertain case is never reported as a conflict. uv remains
// the real resolver; this only surfaces the obvious clashes as an advisory.
func rangesDisjoint(userSpec, envSpec string) bool {
	a, aok := parseClause(userSpec)
	b, bok := parseClause(envSpec)
	if !aok || !bok {
		return false
	}
	// Normalize so "exact-ish" (== / ~=) comparisons can be done against the other
	// side regardless of argument order.
	return clausesDisjoint(a, b) || clausesDisjoint(b, a)
}

// clausesDisjoint decides disjointness treating a as the reference. It handles
// the exact-pin and bound cases; combinations it cannot decide return false.
func clausesDisjoint(a, b clause) bool {
	switch a.op {
	case "==":
		// a pins exactly a.rel; disjoint iff that version does not satisfy b.
		return !satisfies(b, a.rel)
	case "~=":
		// ~=X.Y admits [X.Y, X+1.0); if b is an exact pin, decide by membership.
		if b.op == "==" {
			return !compatibleReleaseContains(a.rel, b.rel)
		}
	}
	return false
}

// satisfies reports whether version v meets clause c. Only the operators used by
// clausesDisjoint's exact-pin path are decided; others conservatively return true
// (i.e. "assume satisfiable", so no conflict is reported).
func satisfies(c clause, v []int) bool {
	cmp := compareRelease(v, c.rel)
	switch c.op {
	case "==":
		return cmp == 0
	case ">=":
		return cmp >= 0
	case ">":
		return cmp > 0
	case "<=":
		return cmp <= 0
	case "<":
		return cmp < 0
	case "~=":
		return compatibleReleaseContains(c.rel, v)
	default:
		return true
	}
}

// compatibleReleaseContains reports whether v falls in the ~=base compatible
// range: v >= base and v shares base's leading release components up to the last
// (e.g. ~=2.4.4 admits [2.4.4, 2.5.0)). base must have at least two components.
func compatibleReleaseContains(base, v []int) bool {
	if len(base) < 2 {
		return false
	}
	if compareRelease(v, base) < 0 {
		return false
	}
	// The upper bound holds all but the last component of base fixed.
	for i := range len(base) - 1 {
		var vi int
		if i < len(v) {
			vi = v[i]
		}
		if vi != base[i] {
			return false
		}
	}
	return true
}

// compareRelease compares two dotted numeric releases component-wise, treating a
// missing trailing component as 0 (so 3.12 == 3.12.0). Returns -1, 0, or 1.
func compareRelease(a, b []int) int {
	n := max(len(a), len(b))
	for i := range n {
		var ai, bi int
		if i < len(a) {
			ai = a[i]
		}
		if i < len(b) {
			bi = b[i]
		}
		if ai != bi {
			if ai < bi {
				return -1
			}
			return 1
		}
	}
	return 0
}

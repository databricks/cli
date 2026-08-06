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
// portion of a dependency string, with any extras stripped. ok is false when there
// is no recognizable name, or when what follows the name carries an environment
// marker (see below).
func splitDepSpec(dep string) (name, spec string, ok bool) {
	m := depSpecRe.FindStringSubmatch(strings.TrimSpace(dep))
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
	// A marker gates the whole requirement on the resolving interpreter, which we do
	// not evaluate, so a pin that may not apply must not be compared. Checking the
	// remainder rather than the raw string scopes this to where a marker can actually
	// appear, leaving an extras list that happens to contain ";" alone. A ";" inside
	// a url requirement still lands here, which costs nothing: a url has no
	// comparable version range and would be undecidable regardless.
	if strings.Contains(spec, ";") {
		return "", "", false
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
	// Every group is decoded, not just dev, because a dev entry may pull in another
	// group by reference — see groupRequirements.
	DependencyGroups map[string][]any `toml:"dependency-groups"`
}

// devGroup is the dependency group whose databricks-connect pin the merge manages.
const devGroup = "dev"

// directRequirements returns only the requirement strings written in the named
// group's own array, without following include-group references. This is the set
// MergeManaged rewrites in place, so it is what distinguishes a pin the merge
// replaces from one it leaves alone (see dbconnectWarning).
func directRequirements(groups map[string][]any, name string) []string {
	var out []string
	for g, entries := range groups {
		// PEP 735 normalizes group names the same way PEP 503 normalizes package
		// names, so "Dev" and "dev" are the same group.
		if normalizePackageName(g) != normalizePackageName(name) {
			continue
		}
		for _, e := range entries {
			if s, ok := e.(string); ok {
				out = append(out, s)
			}
		}
	}
	return out
}

// groupRequirements returns the requirement strings reachable from the named
// dependency group, following PEP 735 {include-group = "..."} indirections.
//
// Resolving the indirection matters because MergeManaged writes the env's pin into
// dev regardless of where the user's own pin lives: with dev = [{include-group =
// "spark"}] and the pin inside the spark group, a non-recursive scan reports no
// override while the merged file ends up carrying two pins for the same package.
// Group names are compared under PEP 503 normalization, which PEP 735 also
// specifies for group names. A group already visited is skipped, so an
// include-group cycle terminates instead of recursing forever.
func groupRequirements(groups map[string][]any, name string) []string {
	byName := make(map[string][]any, len(groups))
	for g, entries := range groups {
		byName[normalizePackageName(g)] = entries
	}

	var out []string
	visited := make(map[string]bool, len(groups))
	var walk func(string)
	walk = func(g string) {
		g = normalizePackageName(g)
		if visited[g] {
			return
		}
		visited[g] = true
		for _, e := range byName[g] {
			switch v := e.(type) {
			case string:
				out = append(out, v)
			case map[string]any:
				if inc, ok := v["include-group"].(string); ok {
					walk(inc)
				}
			}
		}
	}
	walk(name)
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

	// The user's databricks-connect pin differs from the env's. Only meaningful in
	// default mode (c.DatabricksConnect is empty in constraints-only, where the dev
	// group is left untouched).
	if c.DatabricksConnect != "" {
		warnings = append(warnings, dbconnectWarning(p.DependencyGroups, c.DatabricksConnect)...)
	}

	// Conflicts are scanned in [project].dependencies *and* the dev group: uv applies
	// constraint-dependencies to the whole resolution, so a group pin outside the
	// env's constraint breaks `uv sync` exactly like a [project] one. The dev group is
	// where this command's audience keeps its pins, so skipping it would miss the
	// likeliest case.
	userDeps := slices.Concat(p.Project.Dependencies, groupRequirements(p.DependencyGroups, devGroup))
	warnings = append(warnings, constraintConflicts(userDeps, c.ConstraintDeps)...)
	return warnings
}

// dbconnectWarning reports how the merge will treat the user's databricks-connect
// pin, distinguishing two outcomes that need different user actions.
//
// MergeManaged rewrites the pin only when it sits directly in the dev group's own
// array; a pin reached through a PEP 735 include-group is left untouched and the
// env's pin is *inserted alongside it*. That leaves two pins for one package, which
// uv cannot resolve — a strictly worse outcome than an override, and one the user
// has to fix by hand. Reporting both as "was replaced" would state something
// factually untrue and hide the resolution failure behind a reassuring advisory.
func dbconnectWarning(groups map[string][]any, envPin string) []Warning {
	// A pin in dev's own array is what the merge rewrites in place.
	for _, entry := range directRequirements(groups, devGroup) {
		if !isDatabricksConnectDep(entry) {
			continue
		}
		if strings.TrimSpace(entry) == strings.TrimSpace(envPin) {
			return nil
		}
		return []Warning{{
			Code:    WarnDBConnectPinOverridden,
			Message: fmt.Sprintf("databricks-connect %q was replaced by the environment's %q", strings.TrimSpace(entry), envPin),
		}}
	}

	// Nothing in dev itself: an included group may still hold a pin the merge will
	// not touch, so the merged file would carry both.
	for _, entry := range groupRequirements(groups, devGroup) {
		if !isDatabricksConnectDep(entry) {
			continue
		}
		if strings.TrimSpace(entry) == strings.TrimSpace(envPin) {
			return nil
		}
		return []Warning{{
			Code: WarnDBConnectPinDuplicated,
			Message: fmt.Sprintf("databricks-connect %q comes from an included dependency group; the environment's %q was added to %q alongside it, leaving two pins to reconcile",
				strings.TrimSpace(entry), envPin, devGroup),
		}}
	}
	return nil
}

// constraintConflicts flags each user dependency pin that the env's
// constraint-dependencies also constrains to a provably non-overlapping version.
// It is deliberately conservative — it only fires when the two ranges are provably
// disjoint (see rangesDisjoint); an ambiguous pair yields nothing, so a false
// "conflict" is never reported. Warnings come out in userDeps order, which is the
// user's own declaration order.
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
	return warnings
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
	// ">=X.Y, ==X.*", which needs a segment to hold fixed). A single-segment base has
	// no defined range, and clause.interval would index rel[:len(rel)-1] to build the
	// ceiling — yielding an empty prefix and a nonsense bound. Refuse to parse it so
	// it stays an unknown range rather than a wrong one.
	if op == "~=" && len(rel) < 2 {
		return clause{}, false
	}
	return clause{op: op, rel: rel}, true
}

// rangesDisjoint reports whether two version specifiers provably share no
// satisfying version. It is intentionally partial: any pair it cannot decide — a
// "!=", a multi-clause range, an unparseable release — returns false, so an
// uncertain case is never reported as a conflict. uv remains the real resolver;
// this only surfaces the provable clashes as an advisory.
func rangesDisjoint(userSpec, envSpec string) bool {
	a, aok := parseClause(userSpec)
	b, bok := parseClause(envSpec)
	if !aok || !bok {
		return false
	}
	ai, aok := a.interval()
	bi, bok := b.interval()
	if !aok || !bok {
		return false
	}
	return !ai.overlaps(bi)
}

// interval is the contiguous release range a clause admits, with each endpoint
// marked inclusive or exclusive. A nil endpoint is unbounded in that direction.
//
// Modeling each clause as an interval decides every pair of the operators we parse
// — including the opposite-direction bounds (">=20" vs "<19") and "~=" vs "~="
// pairs the published constraint artifacts actually use — with one endpoint
// comparison instead of a per-operator-pair table.
//
// Inclusivity is tracked explicitly rather than normalizing to a half-open range,
// because releases have no well-defined successor: appending a component to make
// ">3.12" into ">=3.12.1" would wrongly exclude 3.12.0.5, which does satisfy
// ">3.12" (compareRelease zero-pads, so 3.12.0.5 > 3.12). Excluding a real member
// shrinks the interval and would report an overlapping pair as disjoint.
type interval struct {
	lo     []int
	loIncl bool
	hi     []int
	hiIncl bool
}

// interval converts a clause to its release range. ok is false for an operator
// whose satisfying set is not contiguous ("!=", the complement of a point), since
// disjointness cannot then be decided by comparing endpoints.
func (c clause) interval() (interval, bool) {
	switch c.op {
	case "==":
		// A single point. compareRelease zero-pads, so this also covers the spellings
		// that denote the same release (3.12 and 3.12.0).
		return interval{lo: c.rel, loIncl: true, hi: c.rel, hiIncl: true}, true
	case ">=":
		return interval{lo: c.rel, loIncl: true}, true
	case ">":
		return interval{lo: c.rel}, true
	case "<=":
		return interval{hi: c.rel, hiIncl: true}, true
	case "<":
		return interval{hi: c.rel}, true
	case "~=":
		// PEP 440: ~=X.Y expands to ">=X.Y, ==X.*" — the floor is the base, and the
		// exclusive ceiling increments the second-to-last segment. parseClause
		// guarantees at least two segments here.
		ceil := make([]int, len(c.rel)-1)
		copy(ceil, c.rel[:len(c.rel)-1])
		ceil[len(ceil)-1]++
		return interval{lo: c.rel, loIncl: true, hi: ceil}, true
	}
	return interval{}, false
}

// overlaps reports whether two intervals share any release. They intersect iff
// each one starts at or before the other ends; an endpoint touching at equal
// versions counts only when both sides include it.
func (a interval) overlaps(b interval) bool {
	return startsBeforeEnd(a.lo, a.loIncl, b.hi, b.hiIncl) &&
		startsBeforeEnd(b.lo, b.loIncl, a.hi, a.hiIncl)
}

// startsBeforeEnd reports whether a range starting at lo does not begin after a
// range ending at hi. A nil endpoint is unbounded, so the condition holds.
func startsBeforeEnd(lo []int, loIncl bool, hi []int, hiIncl bool) bool {
	if lo == nil || hi == nil {
		return true
	}
	switch cmp := compareRelease(lo, hi); {
	case cmp < 0:
		return true
	case cmp > 0:
		return false
	default:
		// Equal endpoints meet at exactly one version, which belongs to both ranges
		// only if neither excludes it.
		return loIncl && hiIncl
	}
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

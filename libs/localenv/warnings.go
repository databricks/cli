package localenv

import (
	"fmt"
	"maps"
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
// release. A trailing "==X.Y.*" prefix wildcard is modeled by parseClause; pre/post/dev
// suffixes are not — a clause we cannot parse this simply is treated as "unknown" and
// never yields a conflict.
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
// file may legitimately carry shapes a stricter struct rejects — a PEP 735 table
// entry such as {include-group = "test"} beside requirement strings, a PEP 621
// dependency table, or a group declared as a sub-table ([dependency-groups.docs],
// the PDM/mkdocs style). BurntSushi reports a type mismatch on any single key as a
// whole-document error, so one such shape anywhere would drop every warning,
// including the ones that do not read the offending key. The container fields are
// therefore typed as []any / map[string]any and narrowed at the point of use.
// requires-python stays a string because PEP 621 specifies it as one; a non-string
// there is malformed and uv rejects the file outright.
type userPyprojectTOML struct {
	Project struct {
		RequiresPython string `toml:"requires-python"`
		Dependencies   []any  `toml:"dependencies"`
		// uv resolves extras alongside the base dependencies, so a pin here is
		// subject to constraint-dependencies exactly like a [project] one.
		OptionalDependencies map[string]any `toml:"optional-dependencies"`
	} `toml:"project"`
	// Every group is decoded, not just dev: uv locks all declared groups, so a pin in
	// any of them is subject to constraint-dependencies (see resolutionRequirements).
	DependencyGroups map[string]any `toml:"dependency-groups"`
	// Tool.Databricks.Environment.EnvironmentVersion is the serverless version the
	// merge manages; it is read here only to warn when it is left stale on a cluster
	// target. A non-string value here is malformed and, like any decode error, yields
	// no warnings rather than a crash.
	Tool struct {
		Databricks struct {
			Environment struct {
				EnvironmentVersion string `toml:"environment_version"`
			} `toml:"environment"`
		} `toml:"databricks"`
	} `toml:"tool"`
}

// devGroup is the dependency group whose databricks-connect pin the merge manages.
const devGroup = "dev"

// stringEntries returns the requirement strings in a decoded dependency array,
// skipping entries of any other shape (a PEP 735 include-group table, a PEP 621
// dependency table). A value that is not an array at all yields nothing.
func stringEntries(v any) []string {
	entries, ok := v.([]any)
	if !ok {
		return nil
	}
	var out []string
	for _, e := range entries {
		if s, ok := e.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// retainedRequirements returns every requirement uv resolves that the merge leaves
// in place: all of resolutionRequirements minus the single entry the merge rewrites,
// identified by replacedPin.
//
// Deriving it by subtraction rather than by re-deciding which groups and spellings
// the merge edits keeps the two from disagreeing: whatever the merge does not report
// as replaced is, by definition, still in the file next to the managed pin. Only the
// first occurrence is dropped, because the merge rewrites only the first element —
// a second identical pin genuinely survives.
func retainedRequirements(p userPyprojectTOML, replacedPin string) []string {
	all := resolutionRequirements(p)
	if replacedPin == "" {
		return all
	}
	out := make([]string, 0, len(all))
	dropped := false
	for _, r := range all {
		if !dropped && strings.TrimSpace(r) == strings.TrimSpace(replacedPin) {
			dropped = true
			continue
		}
		out = append(out, r)
	}
	return out
}

// resolutionRequirements returns every requirement string uv considers when it locks
// the project, in a deterministic order: [project].dependencies, then each extra in
// [project.optional-dependencies], then each dependency group — all of them, not
// just dev.
//
// Scanning every group is what makes the conflict warning match uv's behaviour:
// constraint-dependencies applies to the whole resolution, so a pin in any declared
// group fails `uv sync` with "requirements are unsatisfiable" just as a [project]
// one does. Restricting the scan to dev would stay silent on exactly that failure.
// Every group is visited on its own, so a PEP 735 {include-group = "..."} reference
// needs no traversal: the group it names is enumerated here regardless, and an
// include-group cycle cannot cause recursion.
func resolutionRequirements(p userPyprojectTOML) []string {
	out := stringEntries(p.Project.Dependencies)
	for _, extra := range slices.Sorted(maps.Keys(p.Project.OptionalDependencies)) {
		out = append(out, stringEntries(p.Project.OptionalDependencies[extra])...)
	}
	for _, g := range slices.Sorted(maps.Keys(p.DependencyGroups)) {
		out = append(out, stringEntries(p.DependencyGroups[g])...)
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
// pyprojectTOML, so no shape a real pyproject.toml may legitimately carry can
// suppress the checks that do not depend on it. Greenfield projects (no pre-existing
// content) produce nothing — there is nothing of the user's to override. Warnings are
// deterministic and ordered (requires-python, then databricks-connect, then the
// standalone-pyspark collision, then constraint conflicts in the order uv would
// encounter them) so goldens are stable.
// opts suppresses the warnings for whichever axes are left unmanaged: a skipped
// axis is not written, so there is nothing on it to warn about.
func detectMergeWarnings(userPyproject []byte, c Constraints, plan dbconnectPlan, opts MergeOptions) []Warning {
	if len(userPyproject) == 0 {
		return nil
	}
	// A decode error is not fatal: userPyprojectTOML types its containers loosely so
	// the shapes a user's file legitimately carries all decode, leaving genuine syntax
	// errors as the only failure — and those yield no fields to compare anyway.
	var p userPyprojectTOML
	if err := toml.Unmarshal(userPyproject, &p); err != nil {
		return nil
	}

	var warnings []Warning

	// The user pinned a requires-python that differs from the env's pin; the merge
	// replaces it with the managed value. Skipped under --no-constraints, where the
	// merge leaves the user's requires-python untouched.
	if up := strings.TrimSpace(p.Project.RequiresPython); !opts.SkipConstraints && up != "" && c.RequiresPython != "" && up != strings.TrimSpace(c.RequiresPython) {
		warnings = append(warnings, Warning{
			Code:    WarnRequiresPythonOverridden,
			Message: fmt.Sprintf("requires-python %q is replaced by the environment's %q", up, c.RequiresPython),
		})
	}

	// The requirements uv still resolves after the merge: everything minus the dev pin
	// the merge rewrites (replacedDevPin) and the strays the consolidation pass removes.
	// A conflict against an entry the merge discards describes a state that does not
	// survive it, so excluding both keeps the conflict scan honest.
	survivors := removeDBConnectPins(retainedRequirements(p, plan.replacedDevPin), plan.removed)

	// The user's databricks-connect pins differ from the env's. Only meaningful when
	// databricks-connect is managed: skipped under --no-dbconnect / --constraints-only,
	// and a no-op when the artifact carries no pin.
	managesDBConnect := !opts.SkipDBConnect && c.DatabricksConnect != ""
	if managesDBConnect {
		warnings = append(warnings, dbconnectWarnings(plan, survivors, c.DatabricksConnect)...)
	}

	// A standalone pyspark collides with databricks-connect's vendored pyspark whenever
	// databricks-connect ends up in the environment — whether the env manages it (a pin
	// is managed) or the user's own pyproject pins it (kept as-is when databricks-connect
	// is skipped). Gate on its presence by either route, not on the mode, so this agrees
	// with the validate hard-fail, which keys on the installed venv rather than the mode.
	if managesDBConnect || len(dbconnectPins(survivors)) > 0 {
		warnings = append(warnings, standalonePysparkWarnings(survivors)...)
	}

	// Skipped under --no-constraints: no constraint block is written, so the user's
	// requirements cannot conflict with a managed one.
	if !opts.SkipConstraints {
		warnings = append(warnings, constraintConflicts(survivors, c.ConstraintDeps)...)
	}

	// A cluster target leaves c.EnvironmentVersion empty and does not manage the
	// serverless environment section, so an environment_version left over from an
	// earlier serverless run is neither refreshed nor removed. Warn that it is now
	// stale rather than let it silently misdescribe the target to a downstream reader.
	if c.EnvironmentVersion == "" {
		if ev := strings.TrimSpace(p.Tool.Databricks.Environment.EnvironmentVersion); ev != "" {
			warnings = append(warnings, Warning{
				Code:    WarnStaleEnvironmentVersion,
				Message: fmt.Sprintf("[tool.databricks.environment] environment_version %q is left from a serverless target but the current target is a cluster; it is not updated", ev),
			})
		}
	}
	return warnings
}

// removeDBConnectPins drops from reqs the requirements the consolidation pass deletes,
// one occurrence per removed pin (mirroring retainedRequirements' subtraction of the
// rewritten dev pin). The strays are removed by MergeManaged, so a warning about them
// as if they survived would describe a state the merge does not produce.
func removeDBConnectPins(reqs []string, removed []removedDBConnect) []string {
	if len(removed) == 0 {
		return reqs
	}
	counts := make(map[string]int, len(removed))
	for _, r := range removed {
		counts[strings.TrimSpace(r.pin)]++
	}
	out := make([]string, 0, len(reqs))
	for _, r := range reqs {
		t := strings.TrimSpace(r)
		if counts[t] > 0 {
			counts[t]--
			continue
		}
		out = append(out, r)
	}
	return out
}

// dbconnectWarnings reports how the merge treats the user's databricks-connect pins,
// distinguishing three outcomes that need different user actions. What the merge does
// is not re-derived here — it comes from plan (see planDBConnect):
//
//   - replacedDevPin: the pin sitting directly in the dev group, rewritten in place to
//     the env's version (WarnDBConnectPinOverridden).
//   - removed: strays the consolidation pass deletes from [project].dependencies, an
//     optional-dependency extra, or another group because their range is disjoint from
//     the env pin — they would otherwise make uv unsatisfiable (WarnDBConnectConsolidated).
//   - a databricks-connect pin the line-based merge cannot delete but uv still resolves
//     — a spelling the passes do not reach (single-quoted element, quoted TOML key,
//     inline-table or dotted sub-table form) — left beside the managed pin. Where its
//     range is disjoint from the env's, uv cannot resolve at all
//     (WarnDBConnectPinDuplicated), a distinct failure the user must fix by hand.
//
// The conditions are reported independently rather than as a first-match choice,
// because a run can override the dev pin, consolidate strays, and still leave an
// unresolvable survivor all at once.
func dbconnectWarnings(plan dbconnectPlan, survivors []string, envPin string) []Warning {
	var warnings []Warning

	envPin = strings.TrimSpace(envPin)
	if plan.replacedDevPin != "" && strings.TrimSpace(plan.replacedDevPin) != envPin {
		warnings = append(warnings, Warning{
			Code:    WarnDBConnectPinOverridden,
			Message: fmt.Sprintf("databricks-connect %q is replaced by the environment's %q", strings.TrimSpace(plan.replacedDevPin), envPin),
		})
	}

	// Every removed pin is disjoint from envPin (removeStrayDatabricksConnect only deletes
	// those), so each is a real conflict the user should see — there is no equal-pin case
	// to skip here.
	for _, r := range plan.removed {
		warnings = append(warnings, Warning{
			Code: WarnDBConnectConsolidated,
			Message: fmt.Sprintf("databricks-connect %q in %s conflicts with the environment's %q and is removed; it is managed in %q",
				strings.TrimSpace(r.pin), r.location, envPin, devGroup),
		})
	}

	_, envSpec, envOK := splitDepSpec(envPin)
	for _, pin := range dbconnectPins(survivors) {
		if pin == envPin {
			// An identical pin needs no reconciliation: uv sees one requirement twice.
			continue
		}
		// Two pins are only a problem when nothing satisfies both. ">=16" beside
		// "~=17.2.0" resolves at 17.2.x, so there is nothing for the user to do, and
		// claiming otherwise would send them after a non-existent conflict.
		_, pinSpec, pinOK := splitDepSpec(pin)
		if !envOK || !pinOK || !rangesDisjoint(pinSpec, envSpec) {
			continue
		}
		warnings = append(warnings, Warning{
			Code: WarnDBConnectPinDuplicated,
			Message: fmt.Sprintf("databricks-connect %q is not rewritten by the merge; the environment's %q sits in %q alongside it, and no version satisfies both",
				pin, envPin, devGroup),
		})
	}
	return warnings
}

// dbconnectPins returns the trimmed databricks-connect requirements among entries,
// in order.
func dbconnectPins(entries []string) []string {
	var out []string
	for _, e := range entries {
		if isDatabricksConnectDep(e) {
			out = append(out, strings.TrimSpace(e))
		}
	}
	return out
}

// standalonePysparkWarnings flags a standalone pyspark requirement among reqs. It is a
// coexistence conflict, not a version one: databricks-connect vendors its own pyspark,
// so any separately declared pyspark overwrites it in a shared environment regardless
// of the version pinned. It is therefore reported without inspecting the specifier, and
// once however many groups declare it — one collision to fix, and repeating it would
// inflate the code histogram consumers build from warnings[]. The caller gates this on
// databricks-connect being present in the resolved environment (by either route).
func standalonePysparkWarnings(reqs []string) []Warning {
	for _, r := range reqs {
		if isPysparkDep(r) {
			return []Warning{{
				Code: WarnStandalonePysparkConflict,
				Message: fmt.Sprintf("dependency %q collides with the pyspark bundled in databricks-connect; the two cannot share one environment — remove it, or install it in a separate environment for local Spark",
					strings.TrimSpace(r)),
			}}
		}
	}
	return nil
}

// constraintConflicts flags each user dependency pin that the env's
// constraint-dependencies also constrains to a provably non-overlapping version.
// It is deliberately conservative — it only fires when the two ranges are provably
// disjoint (see rangesDisjoint); an ambiguous pair yields nothing, so a false
// "conflict" is never reported. Warnings come out in userDeps order, and one
// requirement reported once however many places declare it: the same pin listed in
// [project].dependencies and in a group is a single conflict to fix, and repeating it
// would inflate the code histogram consumers build from warnings[].
func constraintConflicts(userDeps, envConstraints []string) []Warning {
	if len(userDeps) == 0 || len(envConstraints) == 0 {
		return nil
	}
	// Index the env constraints by normalized package name. Multiple entries for one
	// package compose as a conjunction, so they are joined with "," rather than
	// letting the last one win, which would decide against an arbitrary subset of the
	// clauses depending on artifact ordering. specInterval intersects the clauses, so
	// the joined form is evaluated as the conjunction it is.
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
	reported := make(map[string]bool)
	for _, ud := range userDeps {
		name, userSpec, ok := splitDepSpec(ud)
		if !ok || userSpec == "" {
			continue
		}
		envSpec, ok := envByName[name]
		if !ok {
			continue
		}
		if !rangesDisjoint(userSpec, envSpec) {
			continue
		}
		// Keyed on the normalized name and the spec, so two spellings of one pin
		// ("pyarrow==21" and "PyArrow ==21") collapse, while genuinely different
		// conflicting pins for one package are each reported.
		key := name + userSpec
		if reported[key] {
			continue
		}
		reported[key] = true
		warnings = append(warnings, Warning{
			Code:    WarnUserConstraintConflict,
			Message: fmt.Sprintf("dependency %q conflicts with the environment constraint %q", strings.TrimSpace(ud), name+envSpec),
		})
	}
	return warnings
}

// clause is a parsed single version clause: an operator and a numeric release.
// wildcard marks a "==X.Y.*" prefix match, whose range is the segment held fixed
// (see clause.interval).
type clause struct {
	op       string
	rel      []int
	wildcard bool
}

// parseClause parses one "<op><release>" clause. ok is false when the clause cannot
// be parsed simply, which is treated as "unknown range" and never yields a conflict.
func parseClause(spec string) (clause, bool) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return clause{}, false
	}
	m := singleClauseRe.FindStringSubmatch(spec)
	if m == nil {
		return clause{}, false
	}
	op := m[1]
	if op == "" {
		op = "==" // a bare version in dependencies means an exact pin
	}
	// A trailing ".*" is a PEP 440 prefix-match wildcard. It is only defined for "=="
	// (and "!="); "==15.1.*" pins the 15.1 series. For any other operator ".*" is
	// malformed, and "!=X.*" excludes a whole series — a non-contiguous set the
	// interval model cannot represent — so both stay unknown. Any other trailing text
	// (pre/post/dev tags) is out of scope and also refused, to avoid a false conflict.
	wildcard := false
	switch rest := strings.TrimSpace(spec[len(m[0]):]); {
	case rest == "":
	case rest == ".*" && op == "==":
		wildcard = true
	default:
		return clause{}, false
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
	return clause{op: op, rel: rel, wildcard: wildcard}, true
}

// rangesDisjoint reports whether two version specifiers provably share no
// satisfying version. It is intentionally partial: any pair it cannot decide — a
// "!=", an unparseable release — returns false, so an uncertain case is never
// reported as a conflict. uv remains the real resolver; this only surfaces the
// provable clashes as an advisory.
func rangesDisjoint(userSpec, envSpec string) bool {
	a, aok := specInterval(userSpec)
	b, bok := specInterval(envSpec)
	if !aok || !bok {
		return false
	}
	return !a.overlaps(b)
}

// specInterval reduces a whole PEP 440 specifier to the single release range it
// admits. A comma-separated specifier is a conjunction, so the clauses are
// intersected: ">=21,<22" is [21, 22). Compound ranges are the ordinary way to pin a
// dependency, and treating the comma as undecidable would miss the conflicts they
// cause — including the ones this detector creates for itself when it conjoins
// duplicate constraint entries for one package.
//
// ok is false when any clause is undecidable, which keeps the whole specifier
// undecidable rather than letting a partial reading of it decide a conflict.
func specInterval(spec string) (interval, bool) {
	if strings.TrimSpace(spec) == "" {
		return interval{}, false
	}
	out := interval{}
	for clauseSpec := range strings.SplitSeq(spec, ",") {
		c, ok := parseClause(clauseSpec)
		if !ok {
			return interval{}, false
		}
		i, ok := c.interval()
		if !ok {
			return interval{}, false
		}
		out = out.intersect(i)
	}
	return out, true
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
		if c.wildcard {
			// "==X.Y.*" admits the whole X.Y series: floor X.Y inclusive, exclusive
			// ceiling incrementing the last held segment ("==15.1.*" -> [15.1, 15.2)).
			ceil := slices.Clone(c.rel)
			ceil[len(ceil)-1]++
			return interval{lo: c.rel, loIncl: true, hi: ceil}, true
		}
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

// intersect returns the range satisfying both a and b: the higher floor and the
// lower ceiling, keeping an endpoint inclusive only where both sides do. The zero
// interval is unbounded in both directions and so acts as the identity, which lets a
// conjunction be folded over its clauses.
//
// An empty result (a floor above the ceiling) is not normalized to any canonical
// form: overlaps compares endpoints, so an empty interval is already disjoint from
// every interval including itself, which is the correct reading of a specifier no
// version satisfies (">=22,<21").
func (a interval) intersect(b interval) interval {
	out := a
	if a.lo == nil || (b.lo != nil && !startsBeforeEnd(b.lo, true, a.lo, true)) {
		// b's floor is strictly higher, so it wins outright.
		out.lo, out.loIncl = b.lo, b.loIncl
	} else if b.lo != nil && compareRelease(a.lo, b.lo) == 0 {
		// Equal floors: the stricter exclusivity applies.
		out.loIncl = a.loIncl && b.loIncl
	}
	if a.hi == nil || (b.hi != nil && !startsBeforeEnd(a.hi, true, b.hi, true)) {
		out.hi, out.hiIncl = b.hi, b.hiIncl
	} else if b.hi != nil && compareRelease(a.hi, b.hi) == 0 {
		out.hiIncl = a.hiIncl && b.hiIncl
	}
	return out
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

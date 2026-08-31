package tests

import (
	"cmp"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/testdiff"
)

// verdict classifies what happened to one field transition. The suite never fails a
// subtest on a bad verdict: the point is to catalog which fields the engine handles,
// so every outcome is recorded and the golden file is the report.
type verdict string

const (
	// The engine planned a change, applied it, and the next plan was clean.
	verdictOK verdict = "OK"
	// Same, but the engine replaced the resource instead of updating it.
	verdictRecreate verdict = "OK_RECREATE"

	// The planner saw the field change and deliberately dropped it. The detail carries
	// the engine's own reason, which is the interesting part.
	verdictSuppressed verdict = "SUPPRESSED"
	// The plan has no entry for the field at all: the two values are indistinguishable
	// in the state the engine sends, so there is nothing to observe. An unset bool and
	// an explicit false are the common case.
	verdictNotObservable verdict = "NOT_OBSERVABLE"
	// The config changed, the field diff exists, but the plan produced no action at all.
	verdictNoPlan verdict = "NO_PLAN"
	// The field drifts with no config change at all: right after the base deploy, the plan
	// already wants to update it. Usually the read does not echo the field, so state and
	// remote never agree. Recorded once per config against the field itself -- otherwise it
	// makes every other field's post-deploy plan dirty and gets blamed on all of them.
	verdictBaselineDrift verdict = "BASELINE_DRIFT"
	// The apply succeeded and the engine sent the write, but the field's remote value is
	// exactly what it was beforehand, on two consecutive reads: the backend accepted the
	// request and ignored this field. The most common shape is a field being cleared, where
	// the update request type is omitempty so the field never makes it into the body.
	//
	// Two reads, not one, is what separates this from STALE_READ.
	verdictUpdateIgnored verdict = "UPDATE_IGNORED"
	// The write did land, but the read straight after the apply did not show it and the one
	// after that did. Not a field-support gap, yet worth seeing: a user planning right
	// after a deploy is shown a change that does not exist.
	verdictStaleRead verdict = "STALE_READ"
	// The apply succeeded, but the plan taken straight afterwards still proposes a change
	// to this field and the remote moved to something else again, so deploying never
	// converges for a reason other than a plainly ignored write.
	verdictDrift verdict = "POST_DEPLOY_DRIFT"
	// The resource declares that it ignores local changes to this field, and it does: the
	// change came back suppressed with exactly the declared reason. Recorded so the claim is
	// verified rather than assumed.
	verdictInertConfirmed verdict = "OK_INERT"
	// The resource declares that it ignores local changes to this field, and it does not.
	// Either the declaration is wrong or the engine stopped honouring it.
	verdictInertViolated verdict = "INERT_NOT_HONOURED"
	// The field under test converged, but updating it left some *other* field of the same
	// resource drifting. The detail names that other field, which is where the fix belongs:
	// one such field makes every deploy of the resource dirty regardless of what changed.
	verdictCollateral verdict = "COLLATERAL_DRIFT"
	// The field itself converged, but some other node of the resource is still drifted
	// post-deploy. Sub-resources are stripped before planning, so nothing should reach
	// this: it guards against blaming a field for drift that is not its own.
	verdictDriftChild verdict = "POST_DEPLOY_DRIFT_CHILD"

	// The backend rejected the value. Usually the value library needs a valid
	// value for this field rather than the generic per-kind default.
	verdictBackendError verdict = "BACKEND_ERROR"
	// Apply failed for a reason that is not an API rejection.
	verdictDeployError verdict = "DEPLOY_ERROR"
	// The operation did not finish inside the per-operation deadline. Recorded rather
	// than waited out: an app rename, for one, blocks on the old name leaving DELETING.
	verdictTimeout verdict = "TIMEOUT"
	// Planning itself failed.
	verdictPlanError verdict = "PLAN_ERROR"

	// The value could not be written into the config at all: the parent object is
	// absent from this base config, or the type does not accept the value.
	verdictUnsettable verdict = "UNSETTABLE"
	// Left out on purpose by the resource's value library, with a reason.
	verdictSkipped verdict = "SKIPPED"
	// The transition's starting value could not be deployed, so the transition
	// itself was never observed.
	verdictBaseError verdict = "BASE_ERROR"
	// The starting value deployed without error but the field did not end up holding it, so
	// the transition under test could not be set up. A field the API refuses to clear is the
	// usual case: nothing can start from absent once the field has ever been set, and a fresh
	// resource whose base config declares the field cannot start from absent either. Reported
	// rather than tested from the wrong starting point, which would label the result a move
	// that never happened.
	verdictStartNotReached verdict = "START_NOT_REACHED"
)

// leavesResourceUsable reports whether the deployed resource is still in a known-good
// state after this outcome. When it is not, the next transition must start from a fresh
// resource, or one real failure cascades into a run of unrelated ones.
func (v verdict) leavesResourceUsable() bool {
	switch v {
	case verdictOK, verdictRecreate, verdictSuppressed, verdictNotObservable, verdictUnsettable,
		verdictSkipped, verdictInertConfirmed, verdictInertViolated:
		return true
	case verdictDrift, verdictCollateral, verdictDriftChild, verdictUpdateIgnored,
		verdictStaleRead, verdictBaselineDrift, verdictStartNotReached:
		// The resource exists and still matches what the engine last sent; only the one
		// field did not stick. The next transition re-deploys the field anyway.
		//
		// START_NOT_REACHED is here for a second reason: the caller has already retried it on
		// a fresh resource, so rebuilding again would buy nothing.
		return true
	default:
		return false
	}
}

// benignSuppressions are the planner's own reasons for dropping a change where nothing the
// user asked for is lost, so the outcome reads like OK and belongs in the full report only.
//
//   - empty: old and new are both empty -- an unset bool against an explicit false
//   - remote_already_set: the remote already holds the value the config asks for
//   - backend_default: the config asks for nothing and the backend chose, by design
//
// Every other reason means a change the user expressed was dropped, which is a finding
// however defensible it is -- terraform_compat and no_update_api being the clearest.
var benignSuppressions = map[string]bool{
	"empty":              true,
	"remote_already_set": true,
	"backend_default":    true,
}

// isProblem reports whether a result is something a person should look at. The committed
// report lists only these; the .full.txt companion lists everything.
func (r result) isProblem() bool {
	switch r.verdict {
	case verdictOK, verdictRecreate, verdictNotObservable, verdictSkipped, verdictInertConfirmed:
		return false
	case verdictSuppressed:
		return !benignSuppressions[r.detail]
	default:
		return true
	}
}

// result is one line of the report.
type result struct {
	config   string
	field    string
	from, to string
	verdict  verdict
	detail   string

	// evidence is the raw material behind the verdict -- the post-deploy plan for drift,
	// the whole API error for a rejection. Printed only into the full report, indented
	// under its line: it is what you would otherwise re-run the case to see, and it is
	// far too long and too full of generated ids to belong in a committed golden.
	evidence string
}

// report accumulates results for one resource type. Resource types run in parallel but
// each has its own report, and a type's configs run in sequence, so the lock only guards
// against a future change to that.
type report struct {
	resourceType string

	mu       sync.Mutex
	results  []result
	wildcard map[string]bool
	covered  map[string]bool
}

func (r *report) add(res result) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.results = append(r.results, res)
}

// addCoverage records what one config reached and what it could not. A field is only
// reported as not covered when *no* config of the type reached it: several configs of one
// resource declare different blocks, so a union of the per-config gaps would list fields
// that are in fact tested -- jobs has eight configs and only some declare tasks.
func (r *report) addCoverage(covered []field, notCovered []string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.wildcard == nil {
		r.wildcard = map[string]bool{}
	}
	if r.covered == nil {
		r.covered = map[string]bool{}
	}
	for _, p := range notCovered {
		r.wildcard[p] = true
	}
	for _, f := range covered {
		r.covered[patternOf(f.path)] = true
	}
}

// patternOf turns a concrete path back into the pattern it came from, so a field reached
// through one config's slice cancels the same pattern reported by a config without it.
func patternOf(path string) string {
	var sb strings.Builder
	for i := 0; i < len(path); i++ {
		if path[i] != '[' {
			sb.WriteByte(path[i])
			continue
		}
		end := strings.IndexByte(path[i:], ']')
		if end < 0 {
			sb.WriteByte(path[i])
			continue
		}
		inner := path[i+1 : i+end]
		if strings.HasPrefix(inner, "'") {
			// A map key: the walk pattern for a map is ".*", not "[*]".
			sb.WriteString(".*")
		} else {
			sb.WriteString("[*]")
		}
		i += end
	}
	return sb.String()
}

// uncoveredPaths returns the patterns no config of this type reached. Caller holds the lock.
func (r *report) uncoveredPaths() []string {
	var out []string
	for path := range r.wildcard {
		if !r.covered[path] {
			out = append(out, path)
		}
	}
	slices.Sort(out)
	return out
}

// render produces a report body. Problems-only is the committed form: one line per finding,
// plus the count of every verdict -- including the passing ones, which no line names. Those
// counts are what makes a change in *passing* behaviour visible: a field that starts being
// recreated instead of updated moves one OK to OK_RECREATE and nothing else would show it,
// on either the fake server or a real workspace. The full form adds every row, the evidence
// behind each finding, and what was not covered.
func (r *report) render(problemsOnly bool) string {
	r.mu.Lock()
	defer r.mu.Unlock()

	var sb strings.Builder

	slices.SortStableFunc(r.results, func(a, b result) int {
		return cmp.Or(
			strings.Compare(a.config, b.config),
			strings.Compare(a.field, b.field),
			strings.Compare(a.from, b.from),
			strings.Compare(a.to, b.to),
		)
	})

	counts := map[verdict]int{}
	config := ""
	for _, res := range r.results {
		counts[res.verdict]++
		if problemsOnly && !res.isProblem() {
			continue
		}
		if res.config != config {
			if config != "" {
				sb.WriteString("\n")
			}
			fmt.Fprintf(&sb, "=== %s\n", res.config)
			config = res.config
		}
		line := fmt.Sprintf("%-44s %-10s %-10s %s", res.field, res.from, res.to, res.verdict)
		if res.detail != "" {
			line += "  " + res.detail
		}
		sb.WriteString(strings.TrimRight(line, " ") + "\n")

		if !problemsOnly && res.evidence != "" {
			// Indented, but with nothing prefixed: leading whitespace is insignificant to
			// JSON, so the block can be copied straight out and parsed.
			for evidenceLine := range strings.SplitSeq(strings.TrimRight(res.evidence, "\n"), "\n") {
				sb.WriteString("    " + evidenceLine + "\n")
			}
		}
	}

	sb.WriteString("\n=== summary\n")
	for _, v := range []verdict{
		verdictOK, verdictRecreate, verdictInertConfirmed, verdictSuppressed,
		verdictNotObservable, verdictNoPlan, verdictInertViolated,
		verdictBaselineDrift, verdictStaleRead, verdictUpdateIgnored, verdictDrift,
		verdictCollateral, verdictDriftChild,
		verdictBackendError, verdictDeployError, verdictTimeout, verdictPlanError,
		verdictUnsettable, verdictSkipped, verdictBaseError, verdictStartNotReached,
	} {
		if counts[v] > 0 {
			fmt.Fprintf(&sb, "%-18s %d\n", v, counts[v])
		}
	}

	if problemsOnly {
		return sb.String()
	}

	if gaps := r.uncoveredPaths(); len(gaps) > 0 {
		paths := gaps
		fmt.Fprintf(&sb, "\n=== not covered: %d fields no config of this type declares\n", len(paths))
		for _, p := range paths {
			fmt.Fprintf(&sb, "%s\n", p)
		}
	}

	return sb.String()
}

// write compares the report against its golden file, or rewrites it under -update.
// Cloud runs go to a separate, uncommitted file: which values a real backend accepts
// depends on the workspace, so those results are for reading, not for diffing.
// write compares the findings against the committed golden. There is one golden, and both
// the fake server and a real workspace are held to it: a divergence means the fake server
// does not model the API faithfully, which is worth failing over rather than filing away in
// a per-cloud file nobody diffs.
//
// -update rewrites it, so a cloud run can be used to correct a golden the fake server got
// wrong. The full form is never compared -- it is an aid for reading one run, and it moves
// whenever a passing row moves.
func (r *report) write(t testutil.TestingT) {
	name := reportPath(r.resourceType + ".txt")
	body := r.render(true)

	fullName := r.resourceType + ".full.txt"
	if isCloud() {
		// Keep both around locally when comparing a cloud run against the fake server.
		fullName = r.resourceType + "." + cloudName() + ".full.txt"
	}
	writeReport(t, reportPath(fullName), r.render(false))

	if testdiff.OverwriteMode {
		writeReport(t, name, body)
		return
	}

	// Not tolerated as missing: a report with no golden would silently pass, so adding a
	// resource type without generating its report, or deleting one, would go unnoticed.
	expected, err := os.ReadFile(name)
	if err != nil {
		t.Errorf("reading %s: %s (run ./task test-update-fields)", name, err)
		return
	}
	testdiff.AssertEqualTexts(t, name, name, testdiff.NormalizeNewlines(string(expected)), body)
}

func writeReport(t testutil.TestingT, name, body string) {
	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		t.Errorf("creating %s: %s", filepath.Dir(name), err)
		return
	}
	if err := os.WriteFile(name, []byte(body), 0o644); err != nil {
		t.Errorf("writing %s: %s", name, err)
	}
}

// writeCorpusReport records which configs the suite drives and why each of the others
// is left out, so the catalog accounts for the whole corpus rather than just the part
// it happens to cover.
func writeCorpusReport(t testutil.TestingT, usable []testConfig, skipped map[string]string) {
	var sb strings.Builder
	sb.WriteString("=== covered\n")
	for _, c := range usable {
		fmt.Fprintf(&sb, "%-48s %s\n", c.name, c.resourceType)
	}

	names := make([]string, 0, len(skipped))
	for name := range skipped {
		names = append(names, name)
	}
	slices.Sort(names)
	fmt.Fprintf(&sb, "\n=== not covered\n")
	for _, name := range names {
		fmt.Fprintf(&sb, "%-48s %s\n", name, skipped[name])
	}

	name := reportPath("configs.txt")
	body := sb.String()
	if isCloud() {
		return
	}
	if testdiff.OverwriteMode {
		writeReport(t, name, body)
		return
	}
	expected, err := os.ReadFile(name)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("reading %s: %s", name, err)
		return
	}
	testdiff.AssertEqualTexts(t, name, name, testdiff.NormalizeNewlines(string(expected)), body)
}

// outputDir holds the committed reports. Kept out of the package directory so the Go
// files stay readable next to two dozen goldens.
const outputDir = "output"

func reportPath(name string) string {
	return filepath.Join(outputDir, name)
}

func cloudName() string {
	switch env := os.Getenv("CLOUD_ENV"); env {
	case "ucws":
		return "aws"
	case "gcp-ucws":
		return "gcp"
	default:
		return env
	}
}

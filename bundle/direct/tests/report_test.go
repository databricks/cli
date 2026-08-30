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
	// The apply succeeded, but the plan taken straight afterwards still proposes the same
	// change to this field, so deploying never converges.
	verdictDrift verdict = "POST_DEPLOY_DRIFT"
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
)

// leavesResourceUsable reports whether the deployed resource is still in a known-good
// state after this outcome. When it is not, the next transition must start from a fresh
// resource, or one real failure cascades into a run of unrelated ones.
func (v verdict) leavesResourceUsable() bool {
	switch v {
	case verdictOK, verdictRecreate, verdictSuppressed, verdictNotObservable, verdictUnsettable, verdictSkipped:
		return true
	case verdictDrift, verdictDriftChild:
		// The resource exists and still matches what the engine last sent; only the one
		// field did not stick. The next transition re-deploys the field anyway.
		return true
	default:
		return false
	}
}

// reasonEmpty is the planner's own reason for dropping a change whose old and new values
// are both empty -- an unset bool against an explicit false, say. Nothing the user
// expressed is lost, so it reads like OK and belongs in the full report only.
const reasonEmpty = "empty"

// isProblem reports whether a result is something a person should look at. The committed
// report lists only these; the .full.txt companion lists everything.
func (r result) isProblem() bool {
	switch r.verdict {
	case verdictOK, verdictRecreate, verdictNotObservable, verdictSkipped:
		return false
	case verdictSuppressed:
		return r.detail != reasonEmpty
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
}

// report accumulates results for one resource type.
type report struct {
	resourceType string
	results      []result
	wildcard     []string
}

func (r *report) add(res result) { r.results = append(r.results, res) }

// render produces a report body. Problems-only is the committed form: one line per
// finding and nothing else, so a diff is always a change in behaviour. The full form
// adds the passing rows, the summary, and what was not covered.
func (r *report) render(problemsOnly bool) string {
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
	}

	if problemsOnly {
		return sb.String()
	}

	sb.WriteString("\n=== summary\n")
	for _, v := range []verdict{
		verdictOK, verdictRecreate, verdictSuppressed, verdictNotObservable, verdictNoPlan,
		verdictDrift, verdictDriftChild,
		verdictBackendError, verdictDeployError, verdictTimeout, verdictPlanError,
		verdictUnsettable, verdictSkipped, verdictBaseError,
	} {
		if counts[v] > 0 {
			fmt.Fprintf(&sb, "%-14s %d\n", v, counts[v])
		}
	}

	if len(r.wildcard) > 0 {
		slices.Sort(r.wildcard)
		fmt.Fprintf(&sb, "\n=== not covered: %d fields under a slice or map\n", len(r.wildcard))
		for _, p := range r.wildcard {
			fmt.Fprintf(&sb, "%s\n", p)
		}
	}

	return sb.String()
}

// write compares the report against its golden file, or rewrites it under -update.
// Cloud runs go to a separate, uncommitted file: which values a real backend accepts
// depends on the workspace, so those results are for reading, not for diffing.
func (r *report) write(t testutil.TestingT) {
	suffix := ".txt"
	if isCloud() {
		suffix = "." + cloudName() + ".txt"
	}
	name := reportPath(r.resourceType + suffix)
	body := r.render(true)

	// The full form is never compared: it is a local aid for reading a run, and it moves
	// whenever a passing row moves.
	writeReport(t, reportPath(r.resourceType+strings.TrimSuffix(suffix, ".txt")+".full.txt"), r.render(false))

	if isCloud() {
		writeReport(t, name, body)
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

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
	// The apply succeeded but a later plan still proposes the same change to this
	// field, so deploying never converges.
	verdictDrift verdict = "DRIFT"
	// The field itself converged, but some other node of the resource is still drifted.
	// Sub-resources are stripped before planning, so nothing should reach this: it is a
	// guard against silently blaming a field for drift that is not its own.
	verdictDriftChild verdict = "DRIFT_CHILD"

	// The backend rejected the value. Usually the value library needs a valid
	// value for this field rather than the generic per-kind default.
	verdictBackendError verdict = "BACKEND_ERROR"
	// Apply failed for a reason that is not an API rejection.
	verdictDeployError verdict = "DEPLOY_ERROR"
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

// result is one line of the report.
type result struct {
	config     string
	field      string
	transition string
	verdict    verdict
	detail     string
}

// report accumulates results for one resource type.
type report struct {
	resourceType string
	results      []result
	wildcard     []string
}

func (r *report) add(res result) { r.results = append(r.results, res) }

// render produces the golden body: one line per transition, grouped by config and
// field, plus a trailing summary of what was not covered.
func (r *report) render() string {
	var sb strings.Builder

	slices.SortStableFunc(r.results, func(a, b result) int {
		return cmp.Or(
			strings.Compare(a.config, b.config),
			strings.Compare(a.field, b.field),
			strings.Compare(a.transition, b.transition),
		)
	})

	counts := map[verdict]int{}
	config := ""
	for _, res := range r.results {
		if res.config != config {
			if config != "" {
				sb.WriteString("\n")
			}
			fmt.Fprintf(&sb, "=== %s\n", res.config)
			config = res.config
		}
		line := fmt.Sprintf("%-44s %-18s %s", res.field, res.transition, res.verdict)
		if res.detail != "" {
			line += "  " + res.detail
		}
		sb.WriteString(strings.TrimRight(line, " ") + "\n")
		counts[res.verdict]++
	}

	sb.WriteString("\n=== summary\n")
	for _, v := range []verdict{
		verdictOK, verdictRecreate, verdictSuppressed, verdictNotObservable, verdictNoPlan,
		verdictDrift, verdictDriftChild,
		verdictBackendError, verdictDeployError, verdictPlanError,
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
	name := reportPath(r.resourceType + ".txt")
	body := r.render()

	if isCloud() {
		writeReport(t, reportPath(r.resourceType+"."+cloudName()+".txt"), body)
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

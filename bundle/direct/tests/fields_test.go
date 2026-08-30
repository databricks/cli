// Package tests exercises the direct engine end to end -- plan, apply, and the plan
// that follows -- rather than any one resource implementation. It lives beside the
// engine for that reason, not under dresources.
//
// TestFields catalogs how the engine handles a change to every field a user can set on
// a bundle resource.
//
// For each resource it deploys a base config once, then walks the resource's input
// struct and, per field, moves the field through every ordered pair of a small set of
// values -- "absent" included, so adding and removing a field are just the pairs with
// absent on one side. Each move is observed three ways: did the plan propose
// anything, did the apply succeed, and was the next plan clean.
//
// Nothing here shells out to the CLI. A `bundle deploy` per permutation would be
// dominated by bundle-file sync, and there are thousands of permutations.
//
// A bad outcome does not fail the test. The suite is a catalog: every result lands in
// output/<resource_type>.txt, and that golden is the report. A regression shows up
// as a diff in it -- for instance after an SDK bump moves a field's type.
//
// Permissions and grants are out of scope, and are stripped from every config before
// anything is planned: they are separate plan nodes describing an ACL rather than the
// resource, and left in place they make every recreate report a drifted child node
// against whichever field happened to trigger the recreate.
//
// Also out of scope for now, tracked as follow-ups:
//   - remote drift (a change made outside the bundle); this suite only edits config
//   - configs with more than one resource, or with an -init.sh
//   - fields under a slice or map (listed at the end of each report)
package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"maps"
	"os/exec"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/stretchr/testify/require"
)

func TestFields(t *testing.T) {
	usable, skipped := discoverConfigs(t)

	// The order fields are tested in, and the order each field's values are visited in,
	// come from this seed. It is the commit rather than the clock so that a finding cannot
	// be retried away: one commit always produces one order, and a new commit explores a
	// different one.
	runSeed := commitSeed(t)

	// One config per resource type: the simplest one, which is the shortest name -- a
	// "<type>.yml.tmpl" sorts before its "<type>_<variant>.yml.tmpl" siblings. The variants
	// exist to exercise specific engine behaviour, not extra fields, and anything a variant
	// declares that the plain config does not is better seeded through testdata/fields, where
	// it applies to every run rather than only where a corpus config happens to have it.
	//
	// It also removes the per-config workspace problem outright: a field's values are the same
	// for every config, so two configs of one type would rename their resource to the same
	// value and delete each other's.
	byType := map[string]testConfig{}
	for _, c := range usable {
		if existing, ok := byType[c.resourceType]; ok && simplerConfig(existing.name, c.name) {
			continue
		}
		byType[c.resourceType] = c
	}

	writeCorpusReport(t, usable, skipped)

	for resourceType, cfg := range byType {
		t.Run(resourceType, func(t *testing.T) {
			// One resource type per goroutine: the types are independent, and the
			// slowest type then sets the wall time instead of their sum.
			t.Parallel()

			fv, err := loadFieldValues(resourceType)
			require.NoError(t, err)
			// A testserver per type keeps the parallel runs from sharing workspace
			// state. On cloud this is the same real workspace either way.
			client := newClient(t)
			user := workspaceUser(t, client)

			// Outlives the field-level subtests that rebuild harnesses.
			ctx := t.Context()

			rep := &report{resourceType: resourceType} //exhaustruct:ignore
			t.Run(cfg.name, func(t *testing.T) {
				runConfig(t, ctx, client, user, cfg, fv, rep, runSeed)
			})
			rep.write(t)
		})
	}
}

func runConfig(t *testing.T, ctx context.Context, client *databricks.WorkspaceClient, user *iam.User, cfg testConfig, fv *fieldValues, rep *report, runSeed uint64) {
	adapter, err := dresources.NewAdapter(dresources.SupportedResources[cfg.resourceType], cfg.resourceType, client)
	require.NoError(t, err)

	// The base deploy establishes that the config itself is deployable. If it is not,
	// nothing below can be attributed to a field.
	h, err := newBaseline(t, ctx, client, user, cfg, fv)
	if err != nil {
		rep.add(result{cfg.name, "(base config)", "", "create", verdictBaseError, oneLine(err.Error()), err.Error()})
		return
	}

	// Enumerated from the deployed config, not just the type: which slice and map entries
	// exist decides which fields inside them can be reached at all.
	base := h.fieldSnapshot()
	fields, wildcard, inert := enumerateFields(cfg.resourceType, adapter.InputConfigType(), fv, base, runSeed, declaredUnsettable(adapter))
	rep.addCoverage(fields, wildcard)

	// Fields the resource declares a user cannot meaningfully set. Recorded with the
	// resource's own reason rather than tested: every transition would come back SUPPRESSED
	// with exactly that reason.
	for _, path := range slices.Sorted(maps.Keys(inert)) {
		rep.add(result{cfg.name, path, "", "", verdictSkipped, "backend output: " + inert[path], ""})
	}

	// Fields the resource says it ignores local changes to. Not skipped: the declaration is
	// a claim about behaviour, and a transition either bears it out or does not.
	ignoredLocally := declaredIgnoredLocally(adapter)

	for path, reason := range fv.skip {
		rep.add(result{cfg.name, path, "", "", verdictSkipped, reason, ""})
	}

	// Some resources do not converge even with nothing changed: a field the read never
	// echoes diffs forever. Measure that once, against the field that causes it, and leave
	// it out of every transition's drift -- otherwise it dirties every post-deploy plan and
	// gets blamed on whichever field happened to be under test.
	baseline := baselineDrift(h)
	for _, path := range slices.Sorted(maps.Keys(baseline)) {
		rep.add(result{cfg.name, path, "", "", verdictBaselineDrift, "drifts with no config change", ""})
	}

	for _, f := range fields {
		t.Run(f.path, func(t *testing.T) {
			// What the field is currently deployed as, when that is known. Transitions are
			// generated in runs that share a starting value, so this skips about half of
			// the setup deploys -- the single biggest cost in the suite.
			deployed, deployedKnown := dyn.InvalidValue, false

			for _, tr := range f.transitions() {
				t.Run(tr.label(), func(t *testing.T) {
					reuse := deployedKnown && valueLabel(deployed) == valueLabel(tr.from)
					res := runTransition(t, h, cfg.name, f.path, tr, reuse, baseline)
					res = checkDeclaredInert(res, ignoredLocally)
					rep.add(res)

					// Only a clean apply leaves the field provably at "to".
					deployed, deployedKnown = tr.to, res.verdict == verdictOK || res.verdict == verdictRecreate

					if res.verdict.leavesResourceUsable() {
						return
					}
					deployedKnown = false
					// The resource is in an unknown state; carrying it into the next
					// transition would turn one failure into a run of them.
					rebuilt, err := rebuild(t, ctx, client, user, cfg, fv, h)
					if err != nil {
						t.Skipf("could not rebuild baseline: %s", err)
					}
					h = rebuilt
					base = h.fieldSnapshot()
				})
			}
		})
		// Put the field back and confirm the resource converged. A field the API cannot
		// clear leaves it drifted for good, which would otherwise be blamed on every
		// field tested afterwards -- so start over on a fresh resource. A new name is
		// what makes that cheap: reusing the old one waits out an asynchronous delete.
		h.restoreField(base, f.path)
		if !h.converged() {
			rebuilt, err := rebuild(t, ctx, client, user, cfg, fv, h)
			if err != nil {
				rep.add(result{cfg.name, f.path, "", "", verdictBaseError, oneLine(err.Error()), err.Error()})
				return
			}
			h = rebuilt
			base = h.fieldSnapshot()
		}
	}
}

// runTransition moves one field from one value to another and reports what happened.
// startsDeployed says the field already holds tr.from, so the setup deploy can be skipped.
func runTransition(t *testing.T, h *bundleHarness, config, path string, tr transition, startsDeployed bool, baseline map[string]bool) result {
	from, to := tr.from, tr.to
	res := result{config: config, field: path, from: valueLabel(tr.from), to: valueLabel(tr.to)} //exhaustruct:ignore

	// Reach the starting value.
	if err := h.setField(path, from); err != nil {
		res.verdict = verdictUnsettable
		res.detail = err.Error()
		return res
	}
	if !startsDeployed {
		if _, diags := h.deploy(); diags.HasError() {
			res.verdict = verdictBaseError
			res.detail = firstError(diags)
			res.evidence = withContext("error from the deploy:", allErrors(diags))
			return res
		}
	}

	// Now the change under test.
	if err := h.setField(path, to); err != nil {
		res.verdict = verdictUnsettable
		res.detail = err.Error()
		return res
	}

	pending, plan, diags := h.plan()
	if diags.HasError() {
		res.verdict = verdictPlanError
		res.detail = firstError(diags)
		return res
	}
	action := nodeAction(plan, h.node)
	if action == deployplan.Skip {
		res.verdict, res.detail = explainSkip(plan, h.node, path)
		return res
	}

	if diags := h.apply(pending, plan); diags.HasError() {
		switch {
		case isTimeout(diags):
			res.verdict = verdictTimeout
		case isAPIError(diags):
			res.verdict = verdictBackendError
		default:
			res.verdict = verdictDeployError
		}
		res.detail = firstError(diags)
		res.evidence = withContext("error from the deploy:", allErrors(diags))
		return res
	}

	_, after, diags := h.plan()
	if diags.HasError() {
		res.verdict = verdictPlanError
		res.detail = firstError(diags)
		return res
	}
	// A write the backend accepted and then ignored leaves the field's remote value exactly
	// as it was before the apply. That is worth separating from other drift: the engine did
	// everything right and the request simply had no effect.
	//
	// One read cannot tell that apart from a read that was merely stale, so take a second
	// one. If the value has appeared by then the write did land and the first read was
	// behind; if it still has not, the field really was ignored.
	if !baseline[path] && wasIgnored(plan, after, h.node, path) {
		_, second, diags := h.plan()
		if diags.HasError() {
			res.verdict = verdictPlanError
			res.detail = firstError(diags)
			return res
		}
		if !wasIgnored(plan, second, h.node, path) {
			res.verdict = verdictStaleRead
			res.detail = path
			res.evidence = withContext("plan taken right after the deploy:", planJSON(after))
			return res
		}
		res.verdict = verdictUpdateIgnored
		res.detail = path
		res.evidence = withContext("plan taken twice after the deploy, still unchanged:", planJSON(second))
		t.Logf("the write was accepted but the remote value did not move on two reads:\n%s", res.evidence)
		return res
	}

	if own, child := driftDetail(after, h.node, baseline); own != "" || child != "" {
		// Whether the field under test is among the drifting ones decides who is to blame.
		drifted := strings.Split(own, ",")
		// The plan that still wants a change is the whole evidence for this verdict, so
		// keep it: -v prints it, and it goes into the full report.
		res.evidence = withContext("plan taken right after the deploy:", planJSON(after))
		t.Logf("post-deploy plan still proposes a change:\n%s", res.evidence)
		switch {
		case own == "":
			// A child node left behind by a recreate: a different problem, different fix.
			res.verdict, res.detail = verdictDriftChild, child
		case slices.Contains(drifted, path):
			res.verdict, res.detail = verdictDrift, own
		default:
			res.verdict, res.detail = verdictCollateral, own
		}
		return res
	}

	res.verdict = verdictOK
	if action == deployplan.Recreate {
		res.verdict = verdictRecreate
	}
	return res
}

// newBaseline builds a harness and deploys the config as written.
func newBaseline(t *testing.T, ctx context.Context, client *databricks.WorkspaceClient, user *iam.User, cfg testConfig, fv *fieldValues) (*bundleHarness, error) {
	h, err := newHarness(t, ctx, client, user, cfg.name, uniqueName(), fv.base)
	if err != nil {
		return nil, err
	}
	t.Cleanup(func() { _ = h.destroy() })

	action, diags := h.deploy()
	if diags.HasError() {
		return nil, errors.New(firstError(diags))
	}
	if action != deployplan.Create {
		return nil, fmt.Errorf("expected create, got %s", action)
	}
	return h, nil
}

// rebuild starts over on a fresh resource, leaving the old one to be destroyed at the
// end of the test. A new name is what makes this cheap: reusing the old one can mean
// waiting out an asynchronous delete.
func rebuild(t *testing.T, ctx context.Context, client *databricks.WorkspaceClient, user *iam.User, cfg testConfig, fv *fieldValues, old *bundleHarness) (*bundleHarness, error) {
	t.Cleanup(func() { _ = old.destroy() })
	return newBaseline(t, ctx, client, user, cfg, fv)
}

// explainSkip says why a plan came back with nothing to do for the field under test.
// The distinction matters: a field the planner never diffed is not a finding (an unset
// bool and an explicit false are the same on the wire), whereas a field it diffed and
// then dropped is, and the engine records its own reason for that.
func explainSkip(plan *deployplan.Plan, node, path string) (verdict, string) {
	entry, ok := plan.Plan[node]
	if !ok {
		return verdictNoPlan, "no plan entry"
	}
	change, ok := entry.Changes[path]
	if !ok {
		return verdictNotObservable, ""
	}
	if change.Reason != "" {
		return verdictSuppressed, change.Reason
	}
	return verdictSuppressed, string(change.Action)
}

// valueLabel renders a value for the subtest name and the report. A slice or map is
// labelled by size, since the point of testing one is how many entries it has.
func valueLabel(v dyn.Value) string {
	if !v.IsValid() {
		return "absent"
	}
	if items, ok := v.AsSequence(); ok {
		return "len" + strconv.Itoa(len(items))
	}
	if m, ok := v.AsMap(); ok {
		return "keys" + strconv.Itoa(m.Len())
	}
	switch s := fmt.Sprintf("%v", v.AsAny()); s {
	case "":
		return "empty"
	default:
		return strings.ReplaceAll(s, " ", "_")
	}
}

// allErrors returns every error diagnostic in full -- status, code, message, endpoint --
// the way the CLI would print it. firstError's one-line form is for the report column.
func allErrors(diags diag.Diagnostics) string {
	var out []string
	for _, d := range diags {
		if d.Severity == diag.Error {
			out = append(out, idPattern.ReplaceAllString(d.Summary, "[UNIQUE_NAME]"))
		}
	}
	return strings.Join(out, "\n")
}

func firstError(diags diag.Diagnostics) string {
	for _, d := range diags {
		if d.Severity == diag.Error {
			return oneLine(d.Summary)
		}
	}
	return ""
}

// apiErrorCodes are the error codes that mean the backend rejected the request rather
// than the CLI failing. Diagnostics carry a rendered message, not the error value, so
// there is nothing to match with errors.As by the time we see them.
var apiErrorCodes = []string{
	"INVALID_PARAMETER_VALUE",
	"BAD_REQUEST",
	"RESOURCE_DOES_NOT_EXIST",
	"RESOURCE_ALREADY_EXISTS",
	"PERMISSION_DENIED",
	"REQUEST_LIMIT_EXCEEDED",
	"FEATURE_DISABLED",
	"NOT_FOUND",
}

func isTimeout(diags diag.Diagnostics) bool {
	for _, d := range diags {
		if strings.Contains(d.Summary, context.DeadlineExceeded.Error()) {
			return true
		}
	}
	return false
}

func isAPIError(diags diag.Diagnostics) bool {
	for _, d := range diags {
		for _, code := range apiErrorCodes {
			if strings.Contains(d.Summary, code) {
				return true
			}
		}
	}
	return false
}

// checkDeclaredInert re-reads a result for a field the resource claims to ignore locally.
// Suppressed with the declared reason confirms the claim; anything else means a change the
// resource said it would drop was not dropped.
func checkDeclaredInert(res result, rules []dresources.FieldRule) result {
	reason, declared := ruleReason(rules, res.field)
	if !declared {
		return res
	}

	switch res.verdict {
	case verdictSuppressed, verdictNotObservable:
		// Both mean the change had no effect, which is what inert claims. Which of the two
		// appears depends on whether the planner suppresses the change or drops the entry
		// outright, and that is an implementation detail of the rule, not of the field.
		res.verdict, res.detail = verdictInertConfirmed, reason
	case verdictOK, verdictRecreate:
		// The change took effect, so the field is not inert after all.
		res.detail = fmt.Sprintf("declared inert (%s) but the change was applied", reason)
		res.verdict = verdictInertViolated
	default:
		// An error or a drift says something more specific than the declaration does; leave
		// the verdict alone and note that the field was expected to be inert.
		res.detail = fmt.Sprintf("%s (declared inert: %s)", res.detail, reason)
	}
	return res
}

// baselineDrift returns the fields the resource already wants to change with the config
// exactly as deployed.
func baselineDrift(h *bundleHarness) map[string]bool {
	_, plan, diags := h.plan()
	if diags.HasError() || plan == nil {
		return nil
	}
	entry, ok := plan.Plan[h.node]
	if !ok {
		return nil
	}
	out := map[string]bool{}
	for path, change := range entry.Changes {
		if change.Action != deployplan.Skip {
			out[path] = true
		}
	}
	return out
}

// wasIgnored reports whether the write for one field was accepted and then had no effect:
// the field is still pending in the post-deploy plan, and its remote value is identical to
// what the pre-deploy plan saw.
func wasIgnored(before, after *deployplan.Plan, node, path string) bool {
	beforeChange, ok := planChange(before, node, path)
	if !ok {
		return false
	}
	afterChange, ok := planChange(after, node, path)
	if !ok || afterChange.Action == deployplan.Skip {
		return false
	}
	return jsonEqual(beforeChange.Remote, afterChange.Remote)
}

func planChange(plan *deployplan.Plan, node, path string) (*deployplan.ChangeDesc, bool) {
	if plan == nil {
		return nil, false
	}
	entry, ok := plan.Plan[node]
	if !ok {
		return nil, false
	}
	change, ok := entry.Changes[path]
	return change, ok
}

// jsonEqual compares two plan values, which are decoded as any and so cannot be compared
// directly.
func jsonEqual(a, b any) bool {
	left, err := json.Marshal(a)
	if err != nil {
		return false
	}
	right, err := json.Marshal(b)
	if err != nil {
		return false
	}
	return bytes.Equal(left, right)
}

// driftDetail describes what a post-deploy plan still wants to change, split into drift
// on the node under test (own: the field paths that did not stick) and drift on any
// other node (child: how a recreate orphaning a permissions or grants node shows up).
// Both are "" when the deploy converged.
func driftDetail(plan *deployplan.Plan, node string, baseline map[string]bool) (own, child string) {
	var ownPaths, childNodes []string
	for key, entry := range plan.Plan {
		if entry.Action == deployplan.Skip {
			continue
		}
		if key != node {
			childNodes = append(childNodes, strings.TrimPrefix(key, node+".")+":"+string(entry.Action))
			continue
		}
		for p, ch := range entry.Changes {
			if ch.Action != deployplan.Skip && !baseline[p] {
				ownPaths = append(ownPaths, p)
			}
		}
		if len(entry.Changes) == 0 {
			ownPaths = append(ownPaths, string(entry.Action))
		}
	}
	slices.Sort(ownPaths)
	slices.Sort(childNodes)
	return strings.Join(ownPaths, ","), strings.Join(childNodes, " ")
}

// idPattern matches the unique suffix this suite gives every resource it creates. The
// suffix is random per run, and error messages quote resource names and ids, so it has
// to be redacted or the goldens differ on every run.
var idPattern = regexp.MustCompile(`f[0-9a-f]{20}`)

// simplerConfig reports whether a is the simpler of two config names for one resource type.
// Shorter wins, so "job.yml.tmpl" beats "job_with_depends_on.yml.tmpl"; length ties break
// alphabetically so the choice is stable.
func simplerConfig(a, b string) bool {
	if len(a) != len(b) {
		return len(a) < len(b)
	}
	return a < b
}

// commitSeed derives the run seed from HEAD, ignoring the working tree: a dirty checkout
// still explores the order of the commit it is based on. Falls back to a fixed value where
// there is no git at all, so the suite still runs.
//
// Tying the seed to the commit rather than the clock is deliberate: a finding cannot be
// made to disappear by running the test again.
func commitSeed(t *testing.T) uint64 {
	out, err := exec.Command("git", "rev-parse", "HEAD").Output()
	if err != nil {
		t.Logf("no git HEAD (%s); falling back to a fixed order seed", err)
		return 0
	}

	commit := strings.TrimSpace(string(out))
	h := fnv.New64a()
	_, _ = h.Write([]byte(commit))
	seed := h.Sum64()
	t.Logf("field and value order seeded from HEAD %s (seed %d)", commit, seed)
	return seed
}

// withContext labels an evidence block, so a reader of the full report knows what they are
// looking at. The label is its own line, leaving the JSON below it copy-pasteable.
func withContext(label, body string) string {
	return label + "\n" + body
}

// planJSON renders a plan for a -v dump.
func planJSON(plan *deployplan.Plan) string {
	body, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		return err.Error()
	}
	return string(body)
}

func oneLine(s string) string {
	s = idPattern.ReplaceAllString(s, "[UNIQUE_NAME]")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > 140 {
		s = s[:140] + "..."
	}
	return s
}

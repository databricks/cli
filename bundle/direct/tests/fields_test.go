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
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/stretchr/testify/require"
)

func TestFields(t *testing.T) {
	usable, skipped := discoverConfigs(t)

	// The order fields are tested in, and the order each field's values are visited in,
	// come from this seed.
	runSeed := orderSeed
	t.Logf("field and value order seeded from orderSeed %d", runSeed)

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
	// Whatever lost the pick is not covered, and saying so is the point of the report.
	for _, c := range usable {
		if winner := byType[c.resourceType]; winner.name != c.name {
			skipped[c.name] = "a simpler config of this type is driven instead: " + winner.name
		}
	}

	writeCorpusReport(t, byType, skipped)

	for resourceType, cfg := range byType {
		t.Run(resourceType, func(t *testing.T) {
			// One resource type per goroutine: the types are independent, and the
			// slowest type then sets the wall time instead of their sum.
			t.Parallel()

			// A testserver per type keeps the parallel runs from sharing workspace
			// state. On cloud this is the same real workspace either way.
			client := newClient(t)
			user := workspaceUser(t, client)

			// A value library may name the workspace user or a shared test object, so it is
			// rendered with the same variables as a corpus config -- minus UNIQUE_NAME, which
			// belongs to one deploy and would be wrong for a value reused across rebuilds.
			vars := templateVars("", user.UserName)
			delete(vars, "UNIQUE_NAME")
			fv, err := loadFieldValues(resourceType, vars)
			require.NoError(t, err)
			if fv.localOnly != "" && isCloud() {
				t.Skipf("local only: %s", fv.localOnly)
			}

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

// runConfig drives one config. Every resource it creates belongs to this test's lifetime,
// which is what `owner` below carries into the subtests: a resource a subtest asks for is
// reused by the transitions after it, so its cleanup cannot be the subtest's.
func runConfig(t *testing.T, ctx context.Context, client *databricks.WorkspaceClient, user *iam.User, cfg testConfig, fv *fieldValues, rep *report, runSeed uint64) {
	owner := t
	adapter, err := dresources.NewAdapter(dresources.SupportedResources[cfg.resourceType], cfg.resourceType, client)
	require.NoError(t, err)

	// The base deploy establishes that the config itself is deployable. If it is not,
	// nothing below can be attributed to a field.
	h, err := newBaseline(t, ctx, client, user, cfg, fv)
	if err != nil {
		rep.add(result{cfg.name, "(base config)", "", "create", verdictBaseError, oneLine(err.Error()), err.Error()})
		return
	}

	// Enumerated from the deployed resource, not just its type: which slice and map entries
	// exist decides which fields inside them can be reached at all.
	base := h.snapshot()
	resource, err := h.resource()
	require.NoError(t, err)
	fields, uncovered, inert := enumerateFields(cfg.resourceType, adapter.InputConfigType(), fv, resource, runSeed,
		declaredUnsettable(adapter), h.unique)
	rep.addCoverage(fields, uncovered)

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
	baseline, err := baselineDrift(h)
	if err != nil {
		rep.add(result{cfg.name, "(baseline)", "", "", verdictPlanError, oneLine(err.Error()), err.Error()})
		return
	}
	for _, path := range slices.Sorted(maps.Keys(baseline)) {
		rep.add(result{cfg.name, path, "", "", verdictBaselineDrift, "drifts with no config change", ""})
	}

	for _, f := range fields {
		// Set once the resource is in an unknown state and could not be replaced. Every
		// remaining transition would be observing that rather than the field, and so would
		// every later field, so they are recorded as such instead of run.
		var broken error

		t.Run(f.path, func(t *testing.T) {
			// What the field is currently deployed as, when that is known. Transitions are
			// generated in runs that share a starting value, so this skips about half of
			// the setup deploys -- the single biggest cost in the suite.
			deployed, deployedKnown := absent, false

			for _, tr := range f.transitions() {
				t.Run(tr.label(), func(t *testing.T) {
					if broken != nil {
						rep.add(result{cfg.name, f.path, valueLabel(tr.from), valueLabel(tr.to), verdictBaseError, oneLine(broken.Error()), broken.Error()})
						return
					}

					reuse := deployedKnown && valueLabel(deployed) == valueLabel(tr.from)
					res := runTransition(t, h, cfg.name, f.path, tr, reuse, baseline)

					// A starting value the deployed resource will not take -- typically a
					// field the API refuses to clear -- is not a dead end: a fresh resource
					// created without the field genuinely starts absent. Rebuild and try the
					// transition once more, so it is observed rather than written off.
					if res.verdict == verdictStartNotReached {
						rebuilt, err := rebuild(owner, ctx, client, user, cfg, fv, h)
						if err != nil {
							// Nothing was observed and there is no resource left to observe
							// it on, which is a different statement from the field refusing
							// its starting value.
							res.verdict = verdictBaseError
							res.detail = oneLine(err.Error())
							res.evidence = err.Error()
							broken = err
						} else {
							h, base = rebuilt, rebuilt.snapshot()
							res = runTransition(t, h, cfg.name, f.path, tr, false, baseline)
						}
					}

					res = checkDeclaredInert(res, ignoredLocally)
					rep.add(res)

					// Only a clean apply leaves the field provably at "to".
					deployed, deployedKnown = tr.to, res.verdict == verdictOK || res.verdict == verdictRecreate

					if broken != nil || res.verdict.leavesResourceUsable() {
						return
					}
					deployedKnown = false
					// The resource is in an unknown state; carrying it into the next
					// transition would turn one failure into a run of them.
					rebuilt, err := rebuild(owner, ctx, client, user, cfg, fv, h)
					if err != nil {
						broken = err
						return
					}
					h = rebuilt
					base = h.snapshot()
				})
			}
		})
		if broken != nil {
			// The type has no usable resource left, so no later field can be observed either.
			rep.add(result{cfg.name, "(rebuild)", "", "", verdictBaseError, oneLine(broken.Error()), broken.Error()})
			return
		}
		// Put the field back and confirm the resource converged. A field the API cannot
		// clear leaves it drifted for good, which would otherwise be blamed on every
		// field tested afterwards -- so start over on a fresh resource. A new name is
		// what makes that cheap: reusing the old one waits out an asynchronous delete.
		h.restore(base)
		if !h.converged() {
			rebuilt, err := rebuild(owner, ctx, client, user, cfg, fv, h)
			if err != nil {
				rep.add(result{cfg.name, f.path, "", "", verdictBaseError, oneLine(err.Error()), err.Error()})
				return
			}
			h = rebuilt
			base = h.snapshot()
		}
	}
}

// sameField reports whether a drifting path is the field under test. The planner may report
// a change against an ancestor rather than the leaf -- a whole "config" block instead of
// "config.auto_capture_config.catalog_name" -- and that is still the field's own drift, not
// some other field's.
func sameField(drifting, path string) bool {
	return drifting == path ||
		strings.HasPrefix(path, drifting+".") || strings.HasPrefix(path, drifting+"[") ||
		strings.HasPrefix(drifting, path+".") || strings.HasPrefix(drifting, path+"[")
}

// relatedChange returns the plan's change for the field, or for whatever ancestor or
// descendant of it the planner actually recorded -- it diffs at its own granularity, so a
// change to "tasks[0].description" can be reported against "tasks". A pending change is
// preferred over a skipped one, since that is the one that explains what will happen.
func relatedChange(plan *deployplan.Plan, node, path string) (*deployplan.ChangeDesc, bool) {
	_, change, ok := relatedChangeKey(plan, node, path)
	return change, ok
}

// relatedChangeKey is relatedChange, also returning the key the change was recorded under so
// a caller comparing two plans can look the same key up in both.
func relatedChangeKey(plan *deployplan.Plan, node, path string) (string, *deployplan.ChangeDesc, bool) {
	if plan == nil {
		return "", nil, false
	}
	entry, ok := plan.Plan[node]
	if !ok {
		return "", nil, false
	}

	var foundKey string
	var found *deployplan.ChangeDesc
	for _, key := range slices.Sorted(maps.Keys(entry.Changes)) {
		change := entry.Changes[key]
		if !sameField(key, path) {
			continue
		}
		if change.Action != deployplan.Skip {
			return key, change, true
		}
		if found == nil {
			foundKey, found = key, change
		}
	}
	return foundKey, found, found != nil
}

// changeAt looks a change up by its exact key.
func changeAt(plan *deployplan.Plan, node, key string) (*deployplan.ChangeDesc, bool) {
	if plan == nil {
		return nil, false
	}
	entry, ok := plan.Plan[node]
	if !ok {
		return nil, false
	}
	change, ok := entry.Changes[key]
	return change, ok
}

// fieldWasDropped reports whether the plan mentions the field and yet will not act on it.
// The planner records a change at whatever granularity it diffed, which is not always the
// leaf -- a whole "tasks" entry rather than "tasks[0].description" -- so every entry naming
// the field, an ancestor of it, or something under it counts. A field the plan does not
// mention at all is not "dropped": there is nothing to conclude from its absence, and the
// node's own action decides.
func fieldWasDropped(plan *deployplan.Plan, node, path string) bool {
	change, ok := relatedChange(plan, node, path)
	return ok && change.Action == deployplan.Skip
}

// baselineDrifts reports whether the field was already drifting before anything was changed.
// Matched by relation, the same way a plan's changes are: the baseline may be recorded against
// an ancestor -- a whole "config" block -- and treating that as this field's failure to reach
// its starting value would blame the ancestor's drift on every field beneath it.
func baselineDrifts(baseline map[string]bool, path string) bool {
	for drifting := range baseline {
		if sameField(drifting, path) {
			return true
		}
	}
	return false
}

// converged reports whether a plan no longer wants to change the field, at whatever
// granularity the planner recorded it.
func converged(plan *deployplan.Plan, node, path string) bool {
	change, ok := relatedChange(plan, node, path)
	return !ok || change.Action == deployplan.Skip
}

// reachedValue reports whether a plan entry for the field means the remote now holds what
// the config asked for. A pending change means it does not; so does a change the planner
// dropped for a reason other than the value already being there -- an app whose compute is
// stopped suppresses a command with "no active deployment", and the command never landed.
func reachedValue(change *deployplan.ChangeDesc) bool {
	return change.Action == deployplan.Skip && benignSuppressions[change.Reason]
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
		// A deploy the backend accepted does not mean the field now holds "from": a write it
		// ignores leaves the old value there, and the transition below would then be reported
		// under a label that is not what happened -- "absent to 168" while the remote still
		// holds 720. Cheaper to check than to reason about afterwards.
		if plan, diags := h.readPlan(); !diags.HasError() {
			if change, ok := relatedChange(plan, h.node, path); ok && !reachedValue(change) && !baselineDrifts(baseline, path) {
				res.verdict = verdictStartNotReached
				res.detail = path
				res.evidence = withContext("plan after deploying the starting value:", planJSON(plan))
				return res
			}
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
		pending.cancel()
		res.verdict = verdictPlanError
		res.detail = firstError(diags)
		return res
	}
	action := nodeAction(plan, h.node)
	// The field's own entries, not just the node's action: another field already drifting keeps
	// the node at "update", and reading only that would let a change to *this* field be
	// dropped and still come out as OK once the other field is filtered from the post-deploy
	// plan.
	if action == deployplan.Skip || fieldWasDropped(plan, h.node, path) {
		pending.cancel()
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

	after, diags := h.readPlan()
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
	if !baselineDrifts(baseline, path) && wasIgnored(plan, after, h.node, path) {
		second, diags := h.readPlan()
		if diags.HasError() {
			res.verdict = verdictPlanError
			res.detail = firstError(diags)
			return res
		}
		switch {
		case wasIgnored(plan, second, h.node, path):
			res.verdict = verdictUpdateIgnored
			res.detail = path
			res.evidence = withContext("plan taken twice after the deploy, still unchanged:", planJSON(second))
			t.Logf("the write was accepted but the remote value did not move on two reads:\n%s", res.evidence)
			return res

		case converged(second, h.node, path):
			// The remote moved and the plan is now clean, so the first read was behind.
			res.verdict = verdictStaleRead
			res.detail = path
			res.evidence = withContext("plan taken right after the deploy:", planJSON(after))
			return res

		default:
			// The remote moved, but to something other than what was asked for, so this is
			// drift rather than a stale read. Classified below, against the later plan.
			after = second
		}
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
		case slices.ContainsFunc(drifted, func(p string) bool { return sameField(p, path) }):
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
//
// owner is the test whose lifetime the resource belongs to, which is the one running the
// config -- not the subtest that happened to ask for the rebuild. Registering the cleanup on
// the subtest would destroy the resource as soon as that transition ended, and every
// transition after it would silently be creating a new one.
func newBaseline(owner *testing.T, ctx context.Context, client *databricks.WorkspaceClient, user *iam.User, cfg testConfig, fv *fieldValues) (*bundleHarness, error) {
	h, err := newHarness(owner, ctx, client, user, cfg.name, uniqueName(), fv.base)
	if err != nil {
		return nil, err
	}
	owner.Cleanup(func() { _ = h.destroy() })

	action, diags := h.deploy()
	if diags.HasError() {
		// The whole diagnostics, not firstError's one-line form: the caller records this as
		// evidence in the full report, where the point is to read the backend's own words.
		return nil, errors.New(allErrors(diags))
	}
	if action != deployplan.Create {
		return nil, fmt.Errorf("expected create, got %s", action)
	}
	return h, nil
}

// rebuild starts over on a fresh resource, leaving the old one to be destroyed at the
// end of the test. A new name is what makes this cheap: reusing the old one can mean
// waiting out an asynchronous delete.
func rebuild(owner *testing.T, ctx context.Context, client *databricks.WorkspaceClient, user *iam.User, cfg testConfig, fv *fieldValues, old *bundleHarness) (*bundleHarness, error) {
	owner.Cleanup(func() { _ = old.destroy() })
	return newBaseline(owner, ctx, client, user, cfg, fv)
}

// explainSkip says why a plan came back with nothing to do for the field under test.
// The distinction matters: a field the planner never diffed is not a finding (an unset
// bool and an explicit false are the same on the wire), whereas a field it diffed and
// then dropped is, and the engine records its own reason for that.
func explainSkip(plan *deployplan.Plan, node, path string) (verdict, string) {
	if _, ok := plan.Plan[node]; !ok {
		return verdictNoPlan, "no plan entry"
	}
	// Whichever entry names the field, which is not always the leaf: the planner records a
	// change at the granularity it diffed at, so the reason for dropping "tasks[0].foo" may
	// be attached to "tasks".
	change, ok := relatedChange(plan, node, path)
	if !ok {
		return verdictNotObservable, ""
	}
	if change.Reason != "" {
		return verdictSuppressed, change.Reason
	}
	return verdictSuppressed, string(change.Action)
}

// valueLabel renders a value for the subtest name and the report. A slice or map is
// labelled by size, since the point of testing one is how many entries it has. An identity
// field's value carries the run's own suffix, which is redacted here so the label -- and so
// the golden -- is the same on every run.
func valueLabel(v any) string {
	if v == nil {
		return "absent"
	}
	switch value := reflect.ValueOf(v); value.Kind() {
	case reflect.Slice, reflect.Array:
		return "len" + strconv.Itoa(value.Len())
	case reflect.Map:
		return "keys" + strconv.Itoa(value.Len())
	default:
	}
	switch s := oneLine(fmt.Sprintf("%v", v)); s {
	case "":
		return "empty"
	default:
		return shortLabel(s)
	}
}

// labelSafe matches the characters a label keeps verbatim: everything else would need
// quoting to pass a label back to `go test -run`.
var labelSafe = regexp.MustCompile(`[^A-Za-z0-9._@/:=-]+`)

// shortLabel renders a value as a token that is readable, stable, and safe to paste into a
// -run filter. A value that survives as-is is left alone; anything long or punctuated -- a
// cluster policy definition is a whole JSON document -- is trimmed and given a short digest,
// so two values that share a prefix still get different labels.
func shortLabel(s string) string {
	const maxLabel = 24

	clean := labelSafe.ReplaceAllString(s, "_")
	if clean == s && len(clean) <= maxLabel {
		return clean
	}

	digest := fnv.New32a()
	_, _ = digest.Write([]byte(s))
	return strings.Trim(clean[:min(len(clean), maxLabel-5)], "_") + "~" + strconv.FormatUint(uint64(digest.Sum32())%0x10000, 16)
}

// allErrors returns every error diagnostic in full -- status, code, message, endpoint --
// the way the CLI would print it. firstError's one-line form is for the report column.
func allErrors(diags diag.Diagnostics) string {
	var out []string
	for _, d := range diags {
		if d.Severity == diag.Error {
			out = append(out, redactIDs(d.Summary))
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
func baselineDrift(h *bundleHarness) (map[string]bool, error) {
	plan, diags := h.readPlan()
	if diags.HasError() {
		// Returning no drift would leave every later verdict measured against an unknown
		// baseline: a field that was already drifting would be blamed on whichever
		// transition happened to run.
		return nil, errors.New(firstError(diags))
	}
	if plan == nil {
		return nil, errors.New("no plan")
	}
	entry, ok := plan.Plan[h.node]
	if !ok {
		return nil, nil
	}
	out := map[string]bool{}
	for path, change := range entry.Changes {
		if change.Action != deployplan.Skip {
			out[path] = true
		}
	}
	return out, nil
}

// wasIgnored reports whether the write for one field was accepted and then had no effect:
// the field is still pending in the post-deploy plan, and its remote value is identical to
// what the pre-deploy plan saw.
func wasIgnored(before, after *deployplan.Plan, node, path string) bool {
	// Matched by relation, since the planner records a change at the granularity it diffed at:
	// a write to "libraries[1].pypi.repo" can be reported against "libraries[1].pypi". But the
	// two reads must be compared at the *same* key -- the granularity can differ between them,
	// and comparing a leaf's value against its parent object never matches, which would turn
	// every ignored write into drift.
	beforeKey, beforeChange, ok := relatedChangeKey(before, node, path)
	if !ok {
		return false
	}
	afterChange, ok := changeAt(after, node, beforeKey)
	if !ok || afterChange.Action == deployplan.Skip {
		return false
	}
	return jsonEqual(beforeChange.Remote, afterChange.Remote)
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
			if ch.Action != deployplan.Skip && !baselineDrifts(baseline, p) {
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

// generatedIDs are the shapes of value that differ between runs, all of which turn up inside
// quoted resource names and ids in error messages -- so they have to be redacted or no golden
// would ever be stable. Ordered: the unique suffix first, since it is the most specific.
var generatedIDs = []struct {
	pattern     *regexp.Regexp
	replacement string
}{
	// The suffix this suite gives every resource it creates.
	{regexp.MustCompile(`f[0-9a-f]{20}`), "[UNIQUE_NAME]"},
	// The placeholder an identity field's values carry. Only ever seen in a label, where it
	// reads better without brackets -- an error message carries the substituted suffix, which
	// the rule above catches.
	{regexp.MustCompile(regexp.QuoteMeta(uniqueMarker)), "UNIQUE"},
	// A backend-assigned id. Before the UUID rule, and covering both shapes: the fake server
	// hands out UUIDs where a real workspace hands out hex, and the two have to redact to the
	// same placeholder or every error message naming one would differ between them.
	{regexp.MustCompile("id=[0-9A-Fa-f-]{6,}"), "id=[ID]"},
	{regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`), "[UUID]"},
}

// redactIDs replaces every run-specific id in a message with a stable placeholder.
func redactIDs(s string) string {
	for _, id := range generatedIDs {
		s = id.pattern.ReplaceAllString(s, id.replacement)
	}
	return s
}

// simplerConfig reports whether a is the simpler of two config names for one resource type.
// Shorter wins, so "job.yml.tmpl" beats "job_with_depends_on.yml.tmpl"; length ties break
// alphabetically so the choice is stable.
func simplerConfig(a, b string) bool {
	if len(a) != len(b) {
		return len(a) < len(b)
	}
	return a < b
}

// orderSeed fixes the order fields are tested in and the order each field's values are
// visited in. Bump it to explore a different order, and regenerate the reports.
//
// It is a constant rather than something derived -- HEAD was the first attempt -- because
// the reports are committed and some verdicts depend on the order. A field the API cannot
// clear leaves the remote holding an old value, and whether the next transition then reads
// that as an ignored write or as drift depends on what ran before it. A seed taken from the
// tree or the clock moves under the golden: committing the report changes HEAD, so the
// report would never validate at the commit that contains it.
const orderSeed uint64 = 1

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
	s = redactIDs(s)
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > 140 {
		s = s[:140] + "..."
	}
	return s
}

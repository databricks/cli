// Package autotest exercises the direct engine end to end -- plan, apply, and the plan
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
package autotest

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"hash/fnv"
	"maps"
	"math/rand/v2"
	"os"
	"os/exec"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/folders"
	"github.com/databricks/cli/libs/git"
	"github.com/databricks/cli/libs/testdiff"
	"github.com/databricks/cli/libs/vfs"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/stretchr/testify/require"
)

func TestFields(t *testing.T) {
	driven := drivenTypes(t)

	sample := sampleSize(t)

	// -update writes the report as the whole truth for a type, so it cannot come from a run
	// that tested a few fields: the missing ones would read as removed.
	if sample > 0 && testdiff.OverwriteMode {
		t.Fatal("-sample cannot be combined with -update: a sampled run would truncate the reports")
	}

	// The order fields are tested in, and the order each field's values are visited in,
	// come from this seed.
	runSeed := orderSeed
	t.Logf("field and value order seeded from orderSeed %d", runSeed)

	var sampleFields uint64
	if sample > 0 {
		sampleFields = sampleSeed(t)
		t.Logf("testing %d fields per resource type, seeded from HEAD (%d)", sample, sampleFields)
	}

	for _, resourceType := range driven {
		t.Run(resourceType, func(t *testing.T) {
			// One resource type per goroutine: the types are independent, and the
			// slowest type then sets the wall time instead of their sum.
			t.Parallel()

			// A testserver per type keeps the parallel runs from sharing workspace
			// state. On cloud this is the same real workspace either way.
			client := newClient(t)
			user := workspaceUser(t, client)

			// A value library may name the workspace user or a shared test object, so the same
			// variables a bundle gets are expanded here too. UNIQUE_NAME is the exception and
			// is left to renderBundle, which knows the deploy it belongs to.
			fv, err := loadFieldValues(resourceType, templateVars("", user.UserName))
			require.NoError(t, err)
			if fv.localOnly != "" && isCloud() {
				t.Skipf("local only: %s", fv.localOnly)
			}
			if len(fv.clouds) > 0 && isCloud() && !slices.Contains(fv.clouds, cloudName()) {
				t.Skipf("not available on %s: the service exists on %s", cloudName(), strings.Join(fv.clouds, ", "))
			}

			// Outlives the field-level subtests that rebuild harnesses.
			ctx := t.Context()

			rep := &report{resourceType: resourceType, started: time.Now()} //exhaustruct:ignore
			runType(t, ctx, client, user, resourceType, fv, rep, runSeed, sampleFields, sample)
			rep.write(t)
		})
	}
}

// runType drives one resource type. Every resource it creates belongs to this test's lifetime,
// which is what `owner` below carries into the subtests: a resource a subtest asks for is
// reused by the transitions after it, so its cleanup cannot be the subtest's.
func runType(t *testing.T, ctx context.Context, client *databricks.WorkspaceClient, user *iam.User, resourceType string, fv *fieldValues, rep *report, runSeed, sampleSeedValue uint64, sample int) {
	owner := t
	adapter, err := dresources.NewAdapter(dresources.SupportedResources[resourceType], resourceType, client)
	require.NoError(t, err)

	// The base deploy establishes that the config itself is deployable. If it is not,
	// nothing below can be attributed to a field.
	h, err := newBaseline(t, ctx, client, user, resourceType, fv)
	if err != nil {
		rep.add(result{"(base config)", "", "create", verdictBaseError, oneLine(err.Error()), err.Error()})
		return
	}

	// Enumerated from the deployed resource, not just its type: which slice and map entries
	// exist decides which fields inside them can be reached at all.
	base := h.snapshot()
	resource, err := h.resource()
	require.NoError(t, err)
	fields, uncovered, inert := enumerateFields(resourceType, adapter.InputConfigType(), fv, resource, runSeed,
		declaredUnsettable(adapter), h.unique)
	rep.addCoverage(fields, uncovered)
	fields = pickSample(fields, sampleSeedValue, sample, rep)

	// A label that does not say what the value is needs the legend to: a container's says only how
	// many entries it has, and an alias says nothing at all.
	for _, f := range fields {
		for _, value := range f.values {
			if label := f.label(value); isContainer(f.kind) || label != valueLabel(value) {
				rep.addLegend(f.path, label, value)
			}
		}
	}

	// Fields the resource declares a user cannot meaningfully set. Recorded with the
	// resource's own reason rather than tested: every transition would come back SUPPRESSED
	// with exactly that reason.
	for _, path := range slices.Sorted(maps.Keys(inert)) {
		rep.add(result{path, "", "", verdictSkipped, "backend output: " + inert[path], ""})
	}

	// Fields the resource says it ignores local changes to. Not skipped: the declaration is
	// a claim about behaviour, and a transition either bears it out or does not.
	ignoredLocally := declaredIgnoredLocally(adapter)
	// Every rule the resource declares about deliberately not acting, which is what makes a
	// skipped change the expected outcome rather than a failure to reach a value.
	decl := declarations{deliberate: declaredDeliberate(adapter), idFields: declaredIDFields(adapter)}

	for path, reason := range fv.skip {
		rep.add(result{path, "", "", verdictSkipped, reason, ""})
	}

	// Some resources do not converge even with nothing changed: a field the read never
	// echoes diffs forever. Measure that once, against the field that causes it, and leave
	// it out of every transition's drift -- otherwise it dirties every post-deploy plan and
	// gets blamed on whichever field happened to be under test.
	baseline, basePlan, err := baselineDrift(h)
	if err != nil {
		rep.add(result{"(baseline)", "", "", verdictPlanError, oneLine(err.Error()), err.Error()})
		return
	}
	for _, path := range slices.Sorted(maps.Keys(baseline)) {
		// The plan is the whole diagnosis: a field drifting straight after the create is usually
		// one the backend did not apply, and only its remote_state says so. Without it the row
		// states the symptom and nothing else, which is not enough to tell a write the API
		// ignored from a read that does not echo the field.
		rep.add(result{path, "", "", verdictBaselineDrift, "drifts with no config change", withContext("plan right after the base deploy:", basePlan)})
	}

	for _, f := range fields {
		// Set once the resource is in an unknown state and could not be replaced. Every
		// remaining transition would be observing that rather than the field, and so would
		// every later field, so they are recorded as such instead of run.
		var broken error

		t.Run(subtestName(f.path), func(t *testing.T) {
			// What the field is currently deployed as, when that is known. Transitions are
			// generated in runs that share a starting value, so this skips about half of
			// the setup deploys -- the single biggest cost in the suite.
			deployed, deployedKnown := absent, false

			for _, tr := range f.transitions() {
				t.Run(tr.label(), func(t *testing.T) {
					if broken != nil {
						rep.add(result{f.path, tr.fromLabel, tr.toLabel, verdictBaseError, oneLine(broken.Error()), broken.Error()})
						return
					}

					reuse := deployedKnown && valueLabel(deployed) == valueLabel(tr.from)
					res := runTransition(t, h, f.path, tr, reuse, baseline, decl)

					// A starting value the deployed resource will not take -- typically a
					// field the API refuses to clear -- is not a dead end: a fresh resource
					// created without the field genuinely starts absent. Rebuild and try the
					// transition once more, so it is observed rather than written off.
					if res.verdict == verdictStartNotReached {
						// With the starting value written into the config before the create, so
						// the new resource is built holding it: a field the API will not move to
						// a value can still be created with it.
						rebuilt, err := rebuild(owner, ctx, client, user, resourceType, fv, h, preset{f.path, tr.from})
						uncreatable := err != nil
						if uncreatable {
							// The resource cannot even be created holding that value, which is the
							// answer for this transition: a dashboard has to have a display name,
							// so nothing starts from it being absent. Recorded, and a plain
							// rebuild follows so the remaining transitions have a resource.
							res.detail = "cannot create the resource with this starting value: " + oneLine(err.Error())
							res.evidence = withContext("error creating the resource with the starting value:", err.Error())
							rebuilt, err = rebuild(owner, ctx, client, user, resourceType, fv, h)
						}
						if err != nil {
							// No resource left to observe anything on, which is a different
							// statement from the field refusing its starting value.
							res.verdict = verdictBaseError
							res.detail = oneLine(err.Error())
							res.evidence = err.Error()
							broken = err
						} else {
							h, base, baseline = rebuilt, rebuilt.snapshot(), remeasure(rebuilt, baseline, rep)
							if !uncreatable {
								res = runTransition(t, h, f.path, tr, false, baseline, decl)
							}
						}
					}

					res = checkDeclaredInert(res, ignoredLocally)
					rep.add(res)

					// Only a clean apply leaves the field provably at "to".
					deployed, deployedKnown = tr.to, res.verdict == verdictOK || res.verdict == verdictRecreate

					if res.verdict == verdictRecreate {
						// A recreate replaced the resource without going through rebuild, so the
						// baseline belongs to one that no longer exists.
						baseline = remeasure(h, baseline, rep)
					}

					if broken != nil || res.verdict.leavesResourceUsable() {
						return
					}
					deployedKnown = false
					// The resource is in an unknown state; carrying it into the next
					// transition would turn one failure into a run of them.
					rebuilt, err := rebuild(owner, ctx, client, user, resourceType, fv, h)
					if err != nil {
						broken = err
						return
					}
					h, base, baseline = rebuilt, rebuilt.snapshot(), remeasure(rebuilt, baseline, rep)
				})
			}
		})
		if broken != nil {
			// The type has no usable resource left, so no later field can be observed either.
			rep.add(result{"(rebuild)", "", "", verdictBaseError, oneLine(broken.Error()), broken.Error()})
			return
		}
		// Put the field back and confirm the resource converged. A field the API cannot
		// clear leaves it drifted for good, which would otherwise be blamed on every
		// field tested afterwards -- so start over on a fresh resource. A new name is
		// what makes that cheap: reusing the old one waits out an asynchronous delete.
		h.restore(base)
		if !h.converged() {
			rebuilt, err := rebuild(owner, ctx, client, user, resourceType, fv, h)
			if err != nil {
				rep.add(result{f.path, "", "", verdictBaseError, oneLine(err.Error()), err.Error()})
				return
			}
			h, base, baseline = rebuilt, rebuilt.snapshot(), remeasure(rebuilt, baseline, rep)
		}
	}
}

// sample cuts the field list down to -sample entries, drawn without replacement. Sorting first
// makes the draw depend only on the seed, not on the order enumeration happened to produce.
// The chosen paths go to the report, which needs them to compare a partial run against a
// golden written by a full one.
func pickSample(fields []field, seed uint64, sample int, rep *report) []field {
	// A type with no more fields than the sample size is covered in full, so its report stays
	// comparable whole -- rep.sampled is left nil.
	if sample <= 0 || len(fields) <= sample {
		return fields
	}
	ordered := slices.Clone(fields)
	slices.SortFunc(ordered, func(a, b field) int { return strings.Compare(a.path, b.path) })
	rng := rand.New(rand.NewPCG(seed, 2))
	rng.Shuffle(len(ordered), func(i, j int) { ordered[i], ordered[j] = ordered[j], ordered[i] })
	ordered = ordered[:sample]
	for _, f := range ordered {
		rep.addSampled(f.path)
	}
	return ordered
}

// sameField reports whether a drifting path is the field under test. The planner may report
// a change against an ancestor rather than the leaf -- a whole "config" block instead of
// "config.auto_capture_config.catalog_name" -- and that is still the field's own drift, not
// some other field's.
func sameField(drifting, path string) bool {
	drifting, path = normalizeSelectors(drifting), normalizeSelectors(path)
	return drifting == path ||
		strings.HasPrefix(path, drifting+".") || strings.HasPrefix(path, drifting+"[") ||
		strings.HasPrefix(drifting, path+".") || strings.HasPrefix(drifting, path+"[")
}

// selectorPattern matches the key-value selector the planner uses to name a slice element it
// can identify -- "tasks[task_key='seeded']" for what this suite calls "tasks[0]".
var selectorPattern = regexp.MustCompile(`\[(?:[^'=\]]+='[^']*'|[0-9]+)\]`)

// normalizeSelectors reduces both ways of naming a slice element to one form, so a change the
// planner recorded against "tasks[task_key='seeded'].run_if" is recognised as the same field as
// "tasks[0].run_if". Which element it is does not need to be resolved: this suite drives one
// element at a time, so at most one is in play.
func normalizeSelectors(path string) string {
	return selectorPattern.ReplaceAllString(path, "[]")
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
func fieldWillChange(plan *deployplan.Plan, node, path string) bool {
	// A create or a recreate replaces the resource and records no per-field detail, so there is
	// nothing to consult and the node's own action is the answer.
	if !hasFieldChanges(plan, node) {
		return true
	}
	change, ok := relatedChange(plan, node, path)
	return ok && change.Action != deployplan.Skip
}

// hasFieldChanges reports whether the plan records per-field detail for the node.
func hasFieldChanges(plan *deployplan.Plan, node string) bool {
	if plan == nil {
		return false
	}
	entry, ok := plan.Plan[node]
	return ok && len(entry.Changes) > 0
}

// remeasure takes the resource's own baseline drift, for a resource that has just replaced the
// one measured before: carrying the old measurement over would either hide drift the new one
// has or blame it on whichever field is under test.
//
// A plan that fails here leaves the previous measurement in place, which is closer than
// nothing, and says so: every verdict after it is measured against a baseline that may not be
// this resource's.
func remeasure(h *bundleHarness, previous map[string]bool, rep *report) map[string]bool {
	measured, _, err := baselineDrift(h)
	if err == nil {
		return measured
	}
	// Recorded, not just logged: every verdict after this is measured against a baseline that
	// belongs to a resource which no longer exists, and a reader of the report has to know.
	rep.add(result{"(baseline)", "", "", verdictPlanError, oneLine(err.Error()), err.Error()})
	return previous
}

// baselineCovers reports whether the drift this field would be blamed for is drift that was
// already there. The comparison is against the key the plan resolved the field to, not the leaf
// under test and not any relation of it: existing drift at "config" explains a change recorded
// against "config", but says nothing about whether a write to "config.foo" landed, and treating
// it as an excuse would let an ignored write pass unnoticed.
func baselineCovers(baseline map[string]bool, plan *deployplan.Plan, node, path string) bool {
	if baseline[path] {
		return true
	}
	key, _, ok := relatedChangeKey(plan, node, path)
	return ok && baseline[key]
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
func reachedValue(change *deployplan.ChangeDesc, path string, deliberate []dresources.FieldRule) bool {
	if change.Action != deployplan.Skip {
		return false
	}
	if benignSuppressions[change.Reason] {
		return true
	}
	// A field the remote type has no place for -- a write-only input like purge_on_delete, which
	// the bundle acts on at destroy and the API never returns. The engine's own definition of the
	// reason is that the remote is always nil, so there is nothing for the value to be reached
	// in; it is in state and that is all there is.
	if change.Reason == deployplan.ReasonMissingInRemote {
		return true
	}
	// A skip for a reason the resource declares about this field is the engine doing what it
	// says it does, so the config's value is as reached as it will ever be. Two shapes: a field
	// whose local changes are dropped never holds what the config asks for (asking whether the
	// starting value landed has no answer, and treating "no" as failure buried the useful
	// OK_INERT verdict), and a field whose remote value is not compared -- an input_only alias --
	// has the value in state and on the wire, just not visible on a read.
	reason, declared := ruleReason(deliberate, path)
	return declared && reason == change.Reason
}

// runTransition moves one field from one value to another and reports what happened.
// startsDeployed says the field already holds tr.from, so the setup deploy can be skipped.
// declarations are the resource's own claims about its fields, read from resources.yml. Two
// slices of the same type as separate parameters would let a caller swap them unnoticed.
type declarations struct {
	// deliberate: rules saying the engine does not act on a field, so a skipped change is the
	// declared behaviour rather than a value that failed to land.
	deliberate []dresources.FieldRule
	// idFields: rules naming the fields that compose the resource's ID.
	idFields []dresources.FieldRule
}

func runTransition(t *testing.T, h *bundleHarness, path string, tr transition, startsDeployed bool, baseline map[string]bool, decl declarations) result {
	from, to := tr.from, tr.to
	res := result{field: path, from: tr.fromLabel, to: tr.toLabel} //exhaustruct:ignore

	// Reach the starting value.
	if err := h.setField(path, from); err != nil {
		res.verdict = verdictUnsettable
		res.detail = err.Error()
		return res
	}
	if !startsDeployed {
		if _, diags := h.deploy(); diags.HasError() {
			res.verdict = idFieldRequired(from, path, decl, diags, verdictBaseError)
			res.detail = firstError(diags)
			res.evidence = withContext("error from the deploy:", allErrors(diags))
			return res
		}
		// A deploy the backend accepted does not mean the field now holds "from": a write it
		// ignores leaves the old value there, and the transition below would then be reported
		// under a label that is not what happened -- "absent to 168" while the remote still
		// holds 720. Cheaper to check than to reason about afterwards.
		plan, diags := h.readPlan()
		if diags.HasError() {
			// Without this plan there is no way to know the field reached "from", and going ahead
			// would label whatever happens next as a move that may never have started.
			res.verdict = verdictPlanError
			res.detail = firstError(diags)
			return res
		}
		if change, ok := relatedChange(plan, h.node, path); ok && !reachedValue(change, path, decl.deliberate) && !baselineCovers(baseline, plan, h.node, path) {
			res.verdict = verdictStartNotReached
			// The planner's own reason for not acting, when it gave one -- "the app has no active
			// deployment" is the whole explanation. The field is already the row's first column,
			// so the detail carries only what that column cannot say.
			res.detail = change.Reason
			res.evidence = withContext("plan after deploying the starting value:", planJSON(plan))
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
	// A field the plan does not act on, however the node's action reads. Another field already
	// drifting keeps the node at "update", and going ahead on that would apply nothing for this
	// field and then report OK once the other field is filtered out as known baseline drift.
	if action == deployplan.Skip || !fieldWillChange(plan, h.node, path) {
		pending.cancel()
		res.verdict, res.detail = explainSkip(plan, h.node, path)
		return res
	}

	if diags := h.apply(pending, plan); diags.HasError() {
		switch {
		case isTimeout(diags):
			res.verdict = verdictTimeout
		case isAPIError(diags):
			res.verdict = idFieldRequired(to, path, decl, diags, verdictBackendError)
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
	if wasIgnored(plan, after, h.node, path) {
		second, diags := h.readPlan()
		if diags.HasError() {
			res.verdict = verdictPlanError
			res.detail = firstError(diags)
			return res
		}
		switch {
		case wasIgnored(plan, second, h.node, path):
			res.verdict = verdictUpdateIgnored
			res.evidence = withContext("plan taken twice after the deploy, still unchanged:", planJSON(second))
			t.Logf("the write was accepted but the remote value did not move on two reads:\n%s", res.evidence)
			return res

		case converged(second, h.node, path):
			// The remote moved and the plan is now clean, so the first read was behind.
			res.verdict = verdictStaleRead
			res.evidence = withContext("plan taken right after the deploy:", planJSON(after))
			return res

		default:
			// The remote moved, but to something other than what was asked for, so this is
			// drift rather than a stale read. Classified below, against the later plan.
			after = second
		}
	}

	if own, child, bare := driftDetail(after, h.node, baseline); own != "" || child != "" || bare != "" {
		// Whether the field under test is among the drifting ones decides who is to blame.
		drifted := strings.Split(own, ",")
		// The plan that still wants a change is the whole evidence for this verdict, so
		// keep it: -v prints it, and it goes into the full report.
		res.evidence = withContext("plan taken right after the deploy:", planJSON(after))
		t.Logf("post-deploy plan still proposes a change:\n%s", res.evidence)
		switch {
		case bare != "":
			// The whole resource is being replaced again, with no field named.
			res.verdict, res.detail = verdictDrift, bare
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
func newBaseline(owner *testing.T, ctx context.Context, client *databricks.WorkspaceClient, user *iam.User, resourceType string, fv *fieldValues, presets ...preset) (*bundleHarness, error) {
	h, err := newHarness(owner, ctx, client, user, resourceType, uniqueName(), fv.base, fv.deps, fv.variables)
	if err != nil {
		return nil, err
	}
	owner.Cleanup(func() { _ = h.destroy() })

	// Applied before the first deploy, so the create carries the value rather than an update
	// having to reach it. That is the only way to start from a value the API will not move a
	// field to -- most often absent, for a field it refuses to clear.
	for _, p := range presets {
		if err := h.setField(p.path, p.value); err != nil {
			return nil, fmt.Errorf("presetting %s: %w", p.path, err)
		}
	}

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

// preset is a field value to write into the config before the resource is first deployed.
type preset struct {
	path  string
	value any
}

// rebuild starts over on a fresh resource, leaving the old one to be destroyed at the
// end of the test. A new name is what makes this cheap: reusing the old one can mean
// waiting out an asynchronous delete.
func rebuild(owner *testing.T, ctx context.Context, client *databricks.WorkspaceClient, user *iam.User, resourceType string, fv *fieldValues, old *bundleHarness, presets ...preset) (*bundleHarness, error) {
	owner.Cleanup(func() { _ = old.destroy() })
	return newBaseline(owner, ctx, client, user, resourceType, fv, presets...)
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

// subtestName turns a field path into a subtest name. An index is written "tasks_0", not
// "tasks[0]": brackets are a character class to the -run regex, so a filter copied from the test
// output would quietly mean something else.
func subtestName(path string) string {
	return indexPattern.ReplaceAllString(path, "_$1")
}

// indexPattern matches a slice index in a field path.
var indexPattern = regexp.MustCompile(`\[([0-9]+)\]`)

// labelSafe matches the characters a label keeps verbatim: everything else would need
// quoting to pass a label back to `go test -run`.
var labelSafe = regexp.MustCompile(`[^A-Za-z0-9._@/:=-]+`)

// shortLabel renders a value as a token that is readable, stable, and safe to paste into a
// -run filter. A value that survives as-is is left alone; anything long or punctuated -- a
// cluster policy definition is a whole JSON document -- is trimmed and given a short digest,
// so two values that share a prefix still get different labels.
func shortLabel(s string) string {
	if clean := cleanLabel(s); clean != "" {
		return clean
	}

	// No alias was assigned for this value -- it is not one of a field's declared values, so
	// nothing holds a legend entry for it. A digest keeps the label short and unique.
	digest := fnv.New32a()
	_, _ = digest.Write([]byte(s))
	safe := labelSafe.ReplaceAllString(s, "_")
	return strings.Trim(safe[:min(len(safe), maxLabelLength-5)], "_") + "~" + strconv.FormatUint(uint64(digest.Sum32())%0x10000, 16)
}

// maxLabelLength is how long a value's own label may be before it is aliased instead.
const maxLabelLength = 24

// cleanLabel returns the value unchanged when it reads as a label already -- short, and free of
// characters that would need quoting in a test filter -- and "" when it does not.
func cleanLabel(s string) string {
	if len(s) <= maxLabelLength && labelSafe.ReplaceAllString(s, "_") == s {
		return s
	}
	return ""
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
// invalidRequestCodes are the subset of apiErrorCodes that say the request itself was wrong, as
// opposed to the caller lacking permission or asking too often.
var invalidRequestCodes = []string{
	"INVALID_PARAMETER_VALUE",
	"BAD_REQUEST",
}

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

// idFieldRequired downgrades a rejected deploy to OK_ID_FIELD_REQUIRED when the value it was
// carrying was absent on a field the resource declares as part of its ID. Every input is a
// declaration or a fact about the run -- the field is named in provided_id_fields, the value is
// absent, the backend refused -- and none is the error's wording, which cannot be attributed
// to a single field because a recreate carries the whole resource.
//
// The backend having refused still has to hold: an internal CLI failure on the same deploy is a
// real defect and keeps its own verdict.
func idFieldRequired(value any, path string, decl declarations, diags diag.Diagnostics, otherwise verdict) verdict {
	// Same nil test valueLabel uses to print "absent", so the verdict and the row's own column
	// can never disagree about which value this was.
	//
	// Not any API error: a rejection is only evidence about the request when it says the request
	// was malformed. PERMISSION_DENIED and REQUEST_LIMIT_EXCEEDED are answers about the caller
	// and the moment, and would have come back whatever the field held.
	if value != nil || !hasErrorCode(diags, invalidRequestCodes) {
		return otherwise
	}
	if _, isID := ruleReason(decl.idFields, path); !isID {
		return otherwise
	}
	return verdictIDFieldRequired
}

func isAPIError(diags diag.Diagnostics) bool {
	return hasErrorCode(diags, apiErrorCodes)
}

func hasErrorCode(diags diag.Diagnostics, codes []string) bool {
	for _, d := range diags {
		for _, code := range codes {
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
func baselineDrift(h *bundleHarness) (map[string]bool, string, error) {
	plan, diags := h.readPlan()
	if diags.HasError() {
		// Returning no drift would leave every later verdict measured against an unknown
		// baseline: a field that was already drifting would be blamed on whichever
		// transition happened to run.
		return nil, "", errors.New(firstError(diags))
	}
	if plan == nil {
		return nil, "", errors.New("no plan")
	}
	entry, ok := plan.Plan[h.node]
	if !ok {
		return nil, "", nil
	}
	out := map[string]bool{}
	for path, change := range entry.Changes {
		if change.Action != deployplan.Skip {
			out[path] = true
		}
	}
	return out, planJSON(plan), nil
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
func driftDetail(plan *deployplan.Plan, node string, baseline map[string]bool) (own, child, bare string) {
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
			// Exact, not by relation: a baseline recorded against an ancestor must not hide new
			// drift reported against something under it. Over-reporting is the safer error for a
			// catalog -- a spurious row is visible and can be investigated, a hidden one cannot.
			if ch.Action != deployplan.Skip && !baseline[p] {
				ownPaths = append(ownPaths, p)
			}
		}
		if len(entry.Changes) == 0 {
			// A recreate carries no per-field detail, so there is no path to name: the whole
			// resource is being replaced again. Reported as the field's own non-convergence,
			// since attributing it to a sibling would name an action where a field belongs.
			bare = string(entry.Action)
		}
	}
	slices.Sort(ownPaths)
	slices.Sort(childNodes)
	return strings.Join(ownPaths, ","), strings.Join(childNodes, " "), bare
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

// workspaceUserName holds the identity this run deploys as, so it can be redacted: it is
// tester@databricks.com against the fake server and a service principal's UUID on a real
// workspace, and it turns up inside seeded values and error messages alike. Written once with
// the same value by every harness, so the mutex only guards the race, not a decision.
var (
	workspaceUserMu   sync.Mutex
	workspaceUserName string
)

func rememberWorkspaceUser(name string) {
	workspaceUserMu.Lock()
	defer workspaceUserMu.Unlock()
	workspaceUserName = name
}

// redactIDs replaces every run-specific id in a message with a stable placeholder.
func redactIDs(s string) string {
	for _, id := range generatedIDs {
		s = id.pattern.ReplaceAllString(s, id.replacement)
	}

	workspaceUserMu.Lock()
	user := workspaceUserName
	workspaceUserMu.Unlock()
	if user != "" {
		s = strings.ReplaceAll(s, user, "[USERNAME]")
	}
	return s
}

// sampleSize limits how many of a type's fields are tested, for a run that has to be cheap
// rather than exhaustive -- a PR check against a real workspace. Which fields are picked comes
// from the commit, so successive commits cover different ground, and a whole run's picks are
// reproducible from its SHA alone.
var sampleSizeFlag = flag.Int("sample", -1, "test only N fields per resource type (0 tests every field; -1 decides from the environment)")

// sampleSize is how many fields per type this run tests, 0 meaning all of them.
//
// A cloud run costs minutes per type and hours in total, which is too much for every PR, so on cloud
// it samples unless the commit asks for everything. That is what AUTOTEST_ALL in a commit title means:
// put it there and the PR's integration run drives every field against a real workspace.
//
// Locally a full run is 11 seconds, so there is nothing to save and the default is all of them --
// which also keeps the committed goldens compared in full by ./task test.
func sampleSize(t *testing.T) int {
	if *sampleSizeFlag >= 0 {
		return *sampleSizeFlag
	}
	if !isCloud() {
		return 0
	}
	for _, subject := range commitSubjects() {
		if strings.Contains(subject, autotestAllMarker) {
			t.Logf("commit title contains %s, so every field runs against the workspace", autotestAllMarker)
			return 0
		}
	}
	t.Logf("cloud run without %s in the commit title, so %d fields per type are sampled; "+
		"put %s in a commit title to run them all", autotestAllMarker, defaultCloudSample, autotestAllMarker)
	return defaultCloudSample
}

// autotestAllMarker in a commit title asks a cloud run for every field rather than a sample.
const autotestAllMarker = "AUTOTEST_ALL"

// defaultCloudSample is how many fields per type a cloud run tests when the commit does not ask for
// all of them. Two is enough to establish that every resource type still deploys and that the fields
// picked still behave as the goldens record.
const defaultCloudSample = 2

// commitSubjects returns the subject of HEAD and, when HEAD is a merge, of its parents. Both are
// needed because CI may check out either the commit itself or the pull request's merge ref, and in
// the second case HEAD is a synthetic "Merge <sha> into <base>" whose subject carries nothing the
// author wrote. Only a merge's parents are consulted, not the branch's history, so a marker used
// once does not keep triggering full runs on the commits after it.
//
// A subprocess because libs/git exposes the commit id but not its message; it runs once per run.
func commitSubjects() []string {
	subject, err := gitOutput("log", "-1", "--format=%s")
	if err != nil {
		return nil
	}
	subjects := []string{subject}

	parents, err := gitOutput("log", "-1", "--format=%P")
	if err != nil {
		return subjects
	}
	fields := strings.Fields(parents)
	if len(fields) < 2 {
		// Not a merge: HEAD is the commit its author wrote, and its subject is the whole answer.
		return subjects
	}
	for _, parent := range fields {
		if parentSubject, err := gitOutput("log", "-1", "--format=%s", parent); err == nil {
			subjects = append(subjects, parentSubject)
		}
	}
	return subjects
}

func gitOutput(args ...string) (string, error) {
	out, err := exec.Command("git", args...).Output()
	return strings.TrimSpace(string(out)), err
}

// sampleSeed derives the sample's seed from HEAD. Unlike orderSeed it may move freely: a
// sampled run compares each field's rows against the same golden as a full run, and never
// rewrites it, so the seed cannot invalidate anything committed.
func sampleSeed(t *testing.T) uint64 {
	wd, err := os.Getwd()
	require.NoError(t, err)
	root, err := folders.FindDirWithLeaf(wd, git.GitDirectoryName)
	require.NoError(t, err)
	repo, err := git.NewRepository(t.Context(), vfs.MustNew(root))
	require.NoError(t, err)
	commit, err := repo.LatestCommit()
	// No fallback: a run that cannot name its commit would pick fields no one can reproduce,
	// which is the one property sampling has to keep.
	require.NoError(t, err, "sampling needs the commit to seed from")
	require.NotEmpty(t, commit, "sampling needs the commit to seed from")
	sum := sha256.Sum256([]byte(commit))
	return binary.BigEndian.Uint64(sum[:8])
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
	// A long URL in the middle pushes out the part that says what went wrong -- a request that
	// never completed reads as `Get "https://…": read tcp …: operation timed out`, and truncating
	// at 140 characters kept the host and dropped the cause. The URL identifies nothing a reader
	// of the report can use: the resource is already the row's own field, and the id is redacted.
	s = urlPattern.ReplaceAllString(s, `"…"`)
	// The engine's own framing -- "cannot update resources.jobs.foo: updating id=...:" -- says only
	// what the row already says: the resource is the report's own filename and the operation follows
	// from the transition. Dropping it leaves the room for what only the backend can say, which was
	// the half being cut: 49 rows were elided mid-sentence with the API's reason on the far side.
	s = enginePrefixPattern.ReplaceAllString(s, "")
	if len(s) > maxDetailLength {
		// Elided in the middle, keeping both ends: the front says what failed, the back says what the
		// backend or the network answered. Cutting at the front dropped the answer.
		//
		// By runes, not bytes: an API message can carry any character, and the ellipsis substituted
		// for a URL above is itself multi-byte, so a byte index can split one and emit invalid UTF-8.
		runes := []rune(s)
		s = string(runes[:maxDetailLength-50]) + "..." + string(runes[len(runes)-47:])
	}
	return s
}

// maxDetailLength caps a row's detail: wide enough that the median backend message survives whole once
// the engine's prefix is gone, narrow enough that a row still fits a terminal.
const maxDetailLength = 200

// enginePrefixPattern matches the engine's framing around a backend error.
var enginePrefixPattern = regexp.MustCompile(`^cannot [a-z]+ resources\.[a-z_]+\.[a-zA-Z0-9_-]+: (?:(?:updating|creating|deleting old|waiting after [a-z]+) id=\S*: )?`)

// urlPattern matches a quoted http(s) URL, which carries a workspace host and a long path and
// crowds out the error that follows it.
var urlPattern = regexp.MustCompile(`"https?://[^"]*"`)

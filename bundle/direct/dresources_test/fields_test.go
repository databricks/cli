// Package dresources_test catalogs how the direct engine handles a change to every
// field a user can set on a bundle resource.
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
// out.<resource_type>.fields.txt, and that golden is the report. A regression shows up
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
package dresources_test

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
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

	byType := map[string][]testConfig{}
	for _, c := range usable {
		byType[c.resourceType] = append(byType[c.resourceType], c)
	}

	writeCorpusReport(t, usable, skipped)

	for resourceType, configs := range byType {
		t.Run(resourceType, func(t *testing.T) {
			// One resource type per goroutine: the types are independent, and the
			// slowest type then sets the wall time instead of their sum.
			t.Parallel()

			fv, err := loadFieldValues(resourceType)
			require.NoError(t, err)
			if isCloud() && !fv.cloud {
				t.Skipf("not enabled for cloud runs; set cloud: true in fields/%s.yml", resourceType)
			}

			// A testserver per type keeps the parallel runs from sharing workspace
			// state. On cloud this is the same real workspace either way.
			client := newClient(t)
			user := workspaceUser(t, client)
			// Outlives the field-level subtests that rebuild harnesses.
			ctx := t.Context()

			rep := &report{resourceType: resourceType} //exhaustruct:ignore
			for _, cfg := range configs {
				t.Run(cfg.name, func(t *testing.T) {
					runConfig(t, ctx, client, user, cfg, fv, rep)
				})
			}
			rep.write(t)
		})
	}
}

func runConfig(t *testing.T, ctx context.Context, client *databricks.WorkspaceClient, user *iam.User, cfg testConfig, fv *fieldValues, rep *report) {
	adapter, err := dresources.NewAdapter(dresources.SupportedResources[cfg.resourceType], cfg.resourceType, client)
	require.NoError(t, err)

	fields, wildcard := enumerateFields(cfg.resourceType, adapter.InputConfigType(), fv)
	rep.wildcard = wildcard

	// The base deploy establishes that the config itself is deployable. If it is not,
	// nothing below can be attributed to a field.
	h, err := newBaseline(t, ctx, client, user, cfg)
	if err != nil {
		rep.add(result{cfg.name, "(base config)", "create", verdictBaseError, oneLine(err.Error())})
		return
	}

	base := h.fieldSnapshot()

	for _, f := range fields {
		if reason, ok := fv.skipReason(f.path); ok {
			rep.add(result{cfg.name, f.path, "-", verdictSkipped, reason})
			continue
		}
		t.Run(f.path, func(t *testing.T) {
			for _, tr := range f.transitions() {
				t.Run(tr.label(), func(t *testing.T) {
					res := runTransition(h, cfg.name, f.path, tr)
					rep.add(res)
					if res.verdict.leavesResourceUsable() {
						return
					}
					// The resource is in an unknown state; carrying it into the next
					// transition would turn one failure into a run of them.
					rebuilt, err := rebuild(t, ctx, client, user, cfg, h)
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
			rebuilt, err := rebuild(t, ctx, client, user, cfg, h)
			if err != nil {
				rep.add(result{cfg.name, f.path, "-", verdictBaseError, oneLine(err.Error())})
				return
			}
			h = rebuilt
			base = h.fieldSnapshot()
		}
	}
}

// runTransition moves one field from one value to another and reports what happened.
func runTransition(h *bundleHarness, config, path string, tr transition) result {
	from, to := tr.from, tr.to
	res := result{config: config, field: path, transition: tr.label()} //exhaustruct:ignore

	// Reach the starting value.
	if err := h.setField(path, from); err != nil {
		res.verdict = verdictUnsettable
		res.detail = err.Error()
		return res
	}
	if _, diags := h.deploy(); diags.HasError() {
		res.verdict = verdictBaseError
		res.detail = firstError(diags)
		return res
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
		res.verdict = verdictDeployError
		if isAPIError(diags) {
			res.verdict = verdictBackendError
		}
		res.detail = firstError(diags)
		return res
	}

	_, after, diags := h.plan()
	if diags.HasError() {
		res.verdict = verdictPlanError
		res.detail = firstError(diags)
		return res
	}
	if own, child := driftDetail(after, h.node); own != "" || child != "" {
		// Attribute to the field only when the field's own node is still drifted; a
		// child left behind by a recreate is a different problem with a different fix.
		res.verdict, res.detail = verdictDrift, own
		if own == "" {
			res.verdict, res.detail = verdictDriftChild, child
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
func newBaseline(t *testing.T, ctx context.Context, client *databricks.WorkspaceClient, user *iam.User, cfg testConfig) (*bundleHarness, error) {
	h, err := newHarness(t, ctx, client, user, cfg.name, uniqueName())
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
func rebuild(t *testing.T, ctx context.Context, client *databricks.WorkspaceClient, user *iam.User, cfg testConfig, old *bundleHarness) (*bundleHarness, error) {
	t.Cleanup(func() { _ = old.destroy() })
	return newBaseline(t, ctx, client, user, cfg)
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

// valueLabel renders a value for the subtest name and the report.
func valueLabel(v dyn.Value) string {
	if !v.IsValid() {
		return "absent"
	}
	switch s := fmt.Sprintf("%v", v.AsAny()); s {
	case "":
		return "empty"
	default:
		return strings.ReplaceAll(s, " ", "_")
	}
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

// driftDetail describes what a post-deploy plan still wants to change, split into drift
// on the node under test (own: the field paths that did not stick) and drift on any
// other node (child: how a recreate orphaning a permissions or grants node shows up).
// Both are "" when the deploy converged.
func driftDetail(plan *deployplan.Plan, node string) (own, child string) {
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
			if ch.Action != deployplan.Skip {
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

func oneLine(s string) string {
	s = idPattern.ReplaceAllString(s, "[UNIQUE_NAME]")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > 140 {
		s = s[:140] + "..."
	}
	return s
}

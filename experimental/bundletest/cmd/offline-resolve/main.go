// Command offline-resolve loads a Declarative Automation Bundle and resolves its
// configuration entirely offline — no workspace client, no auth — then prints the
// resolved config as JSON on stdout.
//
// bundletest's local (DuckDB) backend subprocesses this instead of reimplementing
// variable/include/target resolution in Python. It reuses the CLI's own mutators, so
// includes, target overrides, presets, and ${var.*}/${bundle.*} references resolve
// exactly as `databricks bundle validate` would.
//
// Deliberately offline: it applies only the mutators that need no workspace. In
// particular it never runs PopulateCurrentUser (the first auth call) or
// ResolveLookupVariables (needs the workspace). References only the workspace can
// resolve are preserved for the caller to reject loudly at the use site: ${workspace.*}
// stays literal, and a lookup or unset variable comes through as a sentinel marker (see
// SentinelFormat).
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/loader"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/validate"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/logdiag"
)

// SentinelFormat encodes a variable that can't be resolved offline. The two %s are the
// reason ("lookup" or "unset") and the variable name. The caller (bundletest's local
// backend) matches this marker in the resolved output and rejects the referencing
// resource loudly, since resolving it needs a workspace.
const SentinelFormat = "__bundletest_unresolved__%s__%s__"

// seedOfflineUnresolvable gives each variable that has no offline value a sentinel
// default and drops any lookup, so the real resolver can complete instead of aborting.
//
// Two things would otherwise fail offline: SetVariables errors on a required variable
// with no value (set_variables.go), and dynvar errors resolving a reference to a lookup
// variable whose value only a workspace can supply. Assigning a sentinel *default* (not a
// value, so a BUNDLE_VAR_* / variable-file override still wins) and clearing lookup (so
// SetVariables uses the default rather than deferring to ResolveLookupVariables) lets
// ${var.<name>} resolve to the sentinel, which the caller detects. Must run before SetVariables.
type seedOfflineUnresolvable struct{}

func (seedOfflineUnresolvable) Name() string { return "seedOfflineUnresolvable" }

func (seedOfflineUnresolvable) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	err := b.Config.Mutate(func(root dyn.Value) (dyn.Value, error) {
		for name, v := range b.Config.Variables {
			var reason string
			switch {
			case v.Lookup != nil:
				reason = "lookup"
			case !v.HasValue() && !v.HasDefault():
				reason = "unset"
			default:
				continue
			}
			sentinel := fmt.Sprintf(SentinelFormat, reason, name)
			var err error
			root, err = dyn.Set(root, "variables."+name+".default", dyn.V(sentinel))
			if err != nil {
				return dyn.InvalidValue, err
			}
		}
		// Drop every variables.*.lookup: the only ones present are on vars we just seeded.
		return dyn.Walk(root, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			if len(p) == 3 && p[0] == dyn.Key("variables") && p[2] == dyn.Key("lookup") {
				return v, dyn.ErrDrop
			}
			return v, nil
		})
	})
	return diag.FromErr(err)
}

// offlinePrefixes are the variable-reference prefixes that resolve without a workspace.
// "workspace" is deliberately excluded: workspace fields are unset offline and would
// otherwise resolve to empty strings, hiding references the caller must reject.
var offlinePrefixes = []string{"bundle", "variables"}

func run(ctx context.Context, path, target string) error {
	b, err := bundle.Load(ctx, path)
	if err != nil {
		return err
	}

	selectTarget := mutator.SelectDefaultTarget()
	if target != "" {
		selectTarget = mutator.SelectTarget(target)
	}

	ctx = logdiag.InitContext(ctx)
	logdiag.SetCollect(ctx, true)
	bundle.ApplySeqContext(ctx, b,
		// --- load phase (offline subset of mutator.DefaultMutators) ---
		loader.EntryPoint(),
		loader.ProcessRootIncludes(),
		mutator.EnvironmentsToTargets(),
		mutator.ComputeIdToClusterId(),
		mutator.InitializeVariables(),
		mutator.DefineDefaultTarget(),
		selectTarget,
		bundle.Mutator(seedOfflineUnresolvable{}),

		// --- initialize phase, everything before the first auth call
		// (PopulateCurrentUser) that is safe offline ---
		mutator.RejectInternalResources(),
		validate.AllResourcesHaveValues(),
		validate.ValidateEngine(),
		validate.Scripts(),
		mutator.RewriteSyncPaths(),
		mutator.SyncDefaultPath(),
		mutator.SyncInferRoot(),
		mutator.InitializeCache(),

		// Variable resolution. SetVariables assigns values from BUNDLE_VAR_*, variable
		// files, and defaults. The two ResolveVariableReferences* mutators are the real
		// engine, restricted to offline prefixes so workspace/lookup/unset refs stay literal.
		mutator.SetVariables(),
		mutator.ResolveVariableReferencesInLookup(),
		mutator.ResolveVariableReferencesWithoutResources(offlinePrefixes...),
		mutator.ResolveVariableReferencesOnlyResources(offlinePrefixes...),
	)

	diags := logdiag.FlushCollected(ctx)
	if diags.HasError() {
		return fmt.Errorf("offline resolution failed: %w", diags.Error())
	}

	converted, err := convert.FromTyped(&b.Config, b.Config.Value())
	if err != nil {
		return err
	}
	buf, err := json.Marshal(converted.AsAny())
	if err != nil {
		return err
	}
	os.Stdout.Write(buf)
	return nil
}

func main() {
	target := flag.String("target", "", "bundle target to select (default target if empty)")
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: offline-resolve [--target NAME] <bundle-path>")
		os.Exit(1)
	}

	if err := run(context.Background(), flag.Arg(0), *target); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

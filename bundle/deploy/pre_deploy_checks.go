package deploy

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"sync"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/diag"
)

type preDeployCheckRegistration struct {
	resourceKey string
	newCheck    func(isPlan bool) bundle.Mutator
}

var preDeployCheckRegistrations []preDeployCheckRegistration

// registerPreDeployCheck registers newCheck to create the check that "bundle plan" and
// "bundle deploy" run for the resource type under resourceKey before the deployment
// state is pulled. isPlan is true for "bundle plan", where a check may report a finding
// that blocks a deploy as a warning instead. A check must not modify the configuration,
// because "bundle migrate" does not run it. Call it from init() in the file of that
// resource type.
//
// A type registers at most one check. Checks run in the canonical order of their types
// (the field order of [config.Resources]).
func registerPreDeployCheck(resourceKey string, newCheck func(isPlan bool) bundle.Mutator) {
	preDeployCheckRegistrations = append(preDeployCheckRegistrations, preDeployCheckRegistration{resourceKey: resourceKey, newCheck: newCheck})
}

var orderedPreDeployChecks = sync.OnceValue(func() []func(isPlan bool) bundle.Mutator {
	var resourceKeys []string
	for _, rt := range config.AllResourceTypes() {
		resourceKeys = append(resourceKeys, rt.Key)
	}
	return orderPreDeployChecks(preDeployCheckRegistrations, resourceKeys)
})

// orderPreDeployChecks orders registrations as [registerPreDeployCheck] documents, with
// resource types in the order of resourceKeys. It panics on a registration for a key that
// is not in resourceKeys and on a key that is registered twice, because each of these is
// a mistake in the code that registers.
func orderPreDeployChecks(registrations []preDeployCheckRegistration, resourceKeys []string) []func(isPlan bool) bundle.Mutator {
	typeOrder := map[string]int{}
	for i, key := range resourceKeys {
		typeOrder[key] = i
	}

	regs := slices.Clone(registrations)
	seen := map[string]bool{}
	for _, r := range regs {
		if _, ok := typeOrder[r.resourceKey]; !ok {
			panic(fmt.Sprintf("pre-deploy check registered for unknown resource type %q", r.resourceKey))
		}
		if seen[r.resourceKey] {
			panic(fmt.Sprintf("pre-deploy check registered more than once for resource type %q", r.resourceKey))
		}
		seen[r.resourceKey] = true
	}
	slices.SortFunc(regs, func(a, b preDeployCheckRegistration) int {
		return cmp.Compare(typeOrder[a.resourceKey], typeOrder[b.resourceKey])
	})

	checks := make([]func(isPlan bool) bundle.Mutator, 0, len(regs))
	for _, r := range regs {
		checks = append(checks, r.newCheck)
	}
	return checks
}

type preDeployResourceChecks struct {
	isPlan bool
}

// PreDeployResourceChecks returns a mutator that applies the checks registered with
// registerPreDeployCheck in sequence. Like [bundle.ApplySeqContext], it stops at the first
// error.
func PreDeployResourceChecks(isPlan bool) bundle.Mutator {
	return &preDeployResourceChecks{isPlan: isPlan}
}

func (m *preDeployResourceChecks) Name() string {
	return "PreDeployResourceChecks"
}

func (m *preDeployResourceChecks) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	var checks []bundle.Mutator
	for _, newCheck := range orderedPreDeployChecks() {
		checks = append(checks, newCheck(m.isPlan))
	}
	bundle.ApplySeqContext(ctx, b, checks...)
	return nil
}

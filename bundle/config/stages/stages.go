package stages

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

// Stage is a point in the bundle pipeline at which mutators that each handle a single
// resource type run. The pipeline applies a stage with [ApplyStage] between its generic
// steps; a mutator registers itself for a stage from the file of the type it handles.
type Stage string

const (
	// StageBeforeVariables runs early in the initialize phase, before variables are set
	// and references are resolved, so mutators here see references as written.
	StageBeforeVariables Stage = "before_variables"

	// StageBeforePresets runs when resources are initialized, after run_as is applied and
	// before presets and defaults are applied.
	StageBeforePresets Stage = "before_presets"

	// StageAfterDefaults runs when resources are initialized, after presets and defaults
	// are applied and before bundle permissions are.
	StageAfterDefaults Stage = "after_defaults"

	// StageAfterResourceVariables runs when resources are normalized, after variable
	// references in resources are resolved.
	StageAfterResourceVariables Stage = "after_resource_variables"

	// StageAfterTargetMerge runs when resources are normalized, right after
	// [StageAfterResourceVariables]. Merging the selected target into the root can leave
	// several elements with the same key in a sequence; mutators here merge them. They
	// must not depend on resolved variables, because code that loads a configuration
	// without the initialize phase runs this stage on its own.
	StageAfterTargetMerge Stage = "after_target_merge"

	// StageAfterUCDependencies runs last when resources are normalized, after grants are
	// merged and implicit Unity Catalog references are made explicit.
	StageAfterUCDependencies Stage = "after_uc_dependencies"

	// StageAfterPython runs in the initialize phase after Python code has loaded and
	// mutated resources. Mutators here see the resources Python code defined, and their
	// changes are not visible to Python code.
	StageAfterPython Stage = "after_python"

	// StageAfterFieldValidation runs in the initialize phase after required fields and
	// enum values are validated.
	StageAfterFieldValidation Stage = "after_field_validation"

	// StageBeforePlan runs before the configuration is planned or deployed, and before
	// existing Terraform state is migrated to the direct engine.
	StageBeforePlan Stage = "before_plan"

	// StageAfterBuild runs at the point after which building artifacts and the predeploy
	// script would run, also for commands that run neither of them. Mutators here see the
	// files those steps produce.
	StageAfterBuild Stage = "after_build"
)

type registration struct {
	stage       Stage
	resourceKey string
	newMutators []func() bundle.Mutator
}

var registrations []registration

// Register registers newMutators to run in stage, in the given order, for the resource
// type under resourceKey. Call it from init() in the file of that resource type.
//
// Within a stage, resource types run in their canonical order (the field order of
// [config.Resources]). Several calls for the same type and stage run in the order of the
// name of the first mutator of each call, so that the order depends neither on file
// names nor on the order in which Go runs init functions. Mutators that must run in a
// given order are therefore registered in one call.
func Register(stage Stage, resourceKey string, newMutators ...func() bundle.Mutator) {
	registrations = append(registrations, registration{stage: stage, resourceKey: resourceKey, newMutators: newMutators})
}

var mutatorsByStage = sync.OnceValue(func() map[Stage][]func() bundle.Mutator {
	var resourceKeys []string
	for _, rt := range config.AllResourceTypes() {
		resourceKeys = append(resourceKeys, rt.Key)
	}
	return groupByStage(registrations, resourceKeys)
})

// groupByStage orders registrations as [Register] documents, with resource types in the
// order of resourceKeys, and returns their mutators per stage. It panics on a
// registration for a key that is not in resourceKeys, on a registration without
// mutators, and on a mutator name that is registered twice for one stage, because each
// of these is a mistake in the code that registers.
func groupByStage(registrations []registration, resourceKeys []string) map[Stage][]func() bundle.Mutator {
	typeOrder := map[string]int{}
	for i, key := range resourceKeys {
		typeOrder[key] = i
	}

	type ordered struct {
		registration
		typeIndex int
		firstName string
	}
	var regs []ordered
	for _, r := range registrations {
		typeIndex, ok := typeOrder[r.resourceKey]
		if !ok {
			panic(fmt.Sprintf("mutator registered for unknown resource type %q", r.resourceKey))
		}
		if len(r.newMutators) == 0 {
			panic(fmt.Sprintf("no mutators registered for resource type %q in stage %q", r.resourceKey, r.stage))
		}
		regs = append(regs, ordered{registration: r, typeIndex: typeIndex, firstName: r.newMutators[0]().Name()})
	}
	slices.SortStableFunc(regs, func(a, b ordered) int {
		return cmp.Or(cmp.Compare(a.typeIndex, b.typeIndex), cmp.Compare(a.firstName, b.firstName))
	})

	byStage := map[Stage][]func() bundle.Mutator{}
	seen := map[Stage]map[string]bool{}
	for _, r := range regs {
		if seen[r.stage] == nil {
			seen[r.stage] = map[string]bool{}
		}
		for _, newMutator := range r.newMutators {
			name := newMutator().Name()
			if seen[r.stage][name] {
				panic(fmt.Sprintf("mutator %q registered more than once for stage %q", name, r.stage))
			}
			seen[r.stage][name] = true
			byStage[r.stage] = append(byStage[r.stage], newMutator)
		}
	}
	return byStage
}

type applyStage struct {
	stage Stage
}

// ApplyStage returns a mutator that applies the mutators registered for stage in sequence.
// Like [bundle.ApplySeqContext], it stops at the first error.
func ApplyStage(stage Stage) bundle.Mutator {
	return &applyStage{stage: stage}
}

func (m *applyStage) Name() string {
	return fmt.Sprintf("ApplyStage(%s)", m.stage)
}

func (m *applyStage) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	var mutators []bundle.Mutator
	for _, newMutator := range mutatorsByStage()[m.stage] {
		mutators = append(mutators, newMutator())
	}
	bundle.ApplySeqContext(ctx, b, mutators...)
	return nil
}

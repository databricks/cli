package resourcemutator

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

	// StageBeforeDirectEngine runs before the configuration is planned or deployed by the
	// direct engine, and before state is migrated to it. Mutators here adapt the
	// configuration to what the direct engine expects.
	StageBeforeDirectEngine Stage = "before_direct_engine"

	// StageAfterBuild runs after the steps that can produce files a deployment syncs
	// (building artifacts and the predeploy script), for the commands that run them.
	StageAfterBuild Stage = "after_build"
)

type stageMutator struct {
	stage       Stage
	resourceKey string
	newMutator  func() bundle.Mutator
}

var stageMutators []stageMutator

// registerStageMutator registers newMutator to run in stage for the resource type under
// resourceKey. Call it from init() in the file of that resource type.
func registerStageMutator(stage Stage, resourceKey string, newMutator func() bundle.Mutator) {
	stageMutators = append(stageMutators, stageMutator{stage: stage, resourceKey: resourceKey, newMutator: newMutator})
}

// stageMutatorsByStage groups the registered mutators by stage. Within a stage, resource
// types run in their canonical order (the field order of [config.Resources]), so that the
// order does not depend on file names. The mutators of one type keep their registration
// order: the order of calls within a file, then the order in which Go runs init functions
// of different files. Mutators of one type that depend on each other must therefore be
// registered from one file, in the order they must run.
var stageMutatorsByStage = sync.OnceValue(func() map[Stage][]stageMutator {
	typeOrder := map[string]int{}
	for key := range config.ResourceTypesImplementing[config.ConfigResource]() {
		typeOrder[key] = len(typeOrder)
	}

	byStage := map[Stage][]stageMutator{}
	for _, m := range stageMutators {
		if _, ok := typeOrder[m.resourceKey]; !ok {
			panic(fmt.Sprintf("mutator registered for unknown resource type %q", m.resourceKey))
		}
		byStage[m.stage] = append(byStage[m.stage], m)
	}
	for _, ms := range byStage {
		slices.SortStableFunc(ms, func(a, b stageMutator) int {
			return cmp.Compare(typeOrder[a.resourceKey], typeOrder[b.resourceKey])
		})
	}
	return byStage
})

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
	for _, sm := range stageMutatorsByStage()[m.stage] {
		mutators = append(mutators, sm.newMutator())
	}
	bundle.ApplySeqContext(ctx, b, mutators...)
	return nil
}

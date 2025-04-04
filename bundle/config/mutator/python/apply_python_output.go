package python

import (
	"fmt"

	"github.com/databricks/cli/bundle/config/mutator/resourcemutator"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/merge"
)

// applyPythonOutputResult contains which resources where added, updated, or deleted by Python mutator.
type applyPythonOutputResult struct {
	AddedResources   resourcemutator.ResourceKeySet
	UpdatedResources resourcemutator.ResourceKeySet
	DeletedResources resourcemutator.ResourceKeySet
}

// applyPythonOutput applies output of Python mutator to bundle configuration before Python mutator.
//
// It records applyPythonOutputResult containing which resources where added, updated, or deleted
// by Python mutator.
//
// Return value is equivalent to output except for:
// - if property is unchanged in output, it's original location will be preserved
// - if empty sequence/mapping is deleted in output, it's original value will be preserved
func applyPythonOutput(root, output dyn.Value) (dyn.Value, applyPythonOutputResult, error) {
	result, visitor := createOverrideVisitor(root, output)
	merged, err := merge.Override(root, output, visitor)
	if err != nil {
		return dyn.InvalidValue, result, err
	}

	return merged, result, nil
}

func createOverrideVisitor(leftRoot, rightRoot dyn.Value) (applyPythonOutputResult, merge.OverrideVisitor) {
	resourcesPath := dyn.NewPath(dyn.Key("resources"))
	deleted := resourcemutator.NewResourceKeySet()
	updated := resourcemutator.NewResourceKeySet()
	added := resourcemutator.NewResourceKeySet()

	visitor := merge.OverrideVisitor{
		VisitDelete: func(valuePath dyn.Path, left dyn.Value) error {
			if isOmitemptyDelete(left) {
				return merge.ErrOverrideUndoDelete
			}

			if !valuePath.HasPrefix(resourcesPath) {
				return fmt.Errorf("unexpected change at %q (delete)", valuePath.String())
			}

			// use leftRoot below because it contains deleted resources

			if len(valuePath) == 1 {
				// Example:
				//
				// valuePath: "resources"
				// leftRoot:  {"bundle": ..., "resources": ...},
				// rightRoot: {"bundle": ...}

				return deleted.AddPattern(
					dyn.NewPatternFromPath(valuePath).Append(dyn.AnyKey(), dyn.AnyKey()),
					leftRoot,
				)
			} else if len(valuePath) == 2 {
				// Example:
				//
				// valuePath: "resources.jobs"
				// leftRoot:  {"resources": { "jobs": ..., "pipeline": ...}}},
				// rightRoot: {"resources": { "jobs": ...}}},

				return deleted.AddPattern(
					dyn.NewPatternFromPath(valuePath).Append(dyn.AnyKey()),
					leftRoot,
				)
			} else if len(valuePath) == 3 {
				// Example: "resources.jobs.job_0"
				resourceKey := resourcemutator.ResourceKey{Type: valuePath[1].Key(), Name: valuePath[2].Key()}
				deleted.AddResourceKey(resourceKey)

				return nil
			} else {
				// Example: "resources.jobs.job_0.tags"
				resourceKey := resourcemutator.ResourceKey{Type: valuePath[1].Key(), Name: valuePath[2].Key()}
				updated.AddResourceKey(resourceKey)

				return nil
			}
		},
		VisitInsert: func(valuePath dyn.Path, right dyn.Value) (dyn.Value, error) {
			if !valuePath.HasPrefix(resourcesPath) {
				return dyn.InvalidValue, fmt.Errorf("unexpected change at %q (insert)", valuePath.String())
			}

			// use rightRoot below because it contains result

			if len(valuePath) == 1 {
				// Example:
				//
				// valuePath: "resources"
				// leftRoot:  {"bundle": ...,                    }
				// rightRoot: {"bundle": ..., "resources": {...} }

				return right, added.AddPattern(
					dyn.NewPatternFromPath(valuePath).Append(dyn.AnyKey(), dyn.AnyKey()),
					rightRoot,
				)
			} else if len(valuePath) == 2 {
				// Example:
				//
				// valuePath: "resources.jobs"
				// leftRoot:  {"resources": {               }}
				// rightRoot: {"resources": { "jobs": {...} }}

				return right, added.AddPattern(
					dyn.NewPatternFromPath(valuePath).Append(dyn.AnyKey()),
					rightRoot,
				)
			} else if len(valuePath) == 3 {
				// Example:
				//
				// valuePath: "resources.jobs"
				// leftRoot:  {"resources": { "jobs": {               }}}
				// rightRoot: {"resources": { "jobs": {"job_0": {...} }}}
				resourceKey := resourcemutator.ResourceKey{Type: valuePath[1].Key(), Name: valuePath[2].Key()}
				added.AddResourceKey(resourceKey)

				return right, nil
			} else {
				// Example: "resources.jobs.job_0.email_notifications"
				resourceKey := resourcemutator.ResourceKey{Type: valuePath[1].Key(), Name: valuePath[2].Key()}
				updated.AddResourceKey(resourceKey)

				return right, nil
			}
		},
		VisitUpdate: func(valuePath dyn.Path, _, right dyn.Value) (dyn.Value, error) {
			if !valuePath.HasPrefix(resourcesPath) {
				return dyn.InvalidValue, fmt.Errorf("unexpected change at %q (update)", valuePath.String())
			}

			// use rightRoot below because it contains result

			if len(valuePath) == 1 {
				// Example:
				//
				// valuePath: "resources"
				// leftRoot:  {"bundle": ..., "resources": null  }
				// rightRoot: {"bundle": ..., "resources": {...} }

				return right, added.AddPattern(
					dyn.NewPatternFromPath(valuePath).Append(dyn.AnyKey(), dyn.AnyKey()),
					rightRoot,
				)
			} else if len(valuePath) == 2 {
				// Example:
				//
				// valuePath: "resources.jobs"
				// leftRoot:  {"resources": { "jobs": null  }}
				// rightRoot: {"resources": { "jobs": {...} }}

				return right, added.AddPattern(
					dyn.NewPatternFromPath(valuePath).Append(dyn.AnyKey()),
					rightRoot,
				)
			} else if len(valuePath) == 3 {
				// Example:
				//
				// valuePath: "resources.jobs.job_0"
				// leftRoot:  {"resources": { "jobs": {"job_0": null  }}}
				// rightRoot: {"resources": { "jobs": {"job_0": {...} }}}
				resourceKey := resourcemutator.ResourceKey{Type: valuePath[1].Key(), Name: valuePath[2].Key()}
				added.AddResourceKey(resourceKey)

				return right, nil
			} else {
				// Example: "resources.jobs.job_0.name"
				resourceKey := resourcemutator.ResourceKey{Type: valuePath[1].Key(), Name: valuePath[2].Key()}
				updated.AddResourceKey(resourceKey)

				return right, nil
			}
		},
	}

	return applyPythonOutputResult{
		AddedResources:   added,
		UpdatedResources: updated,
		DeletedResources: deleted,
	}, visitor
}

func isOmitemptyDelete(left dyn.Value) bool {
	// Python output can omit empty sequences/mappings, because we don't track them as optional,
	// there is no semantic difference between empty and missing, so we keep them as they were before
	// Python mutator deleted them.

	switch left.Kind() {
	case dyn.KindMap:
		return left.MustMap().Len() == 0

	case dyn.KindSequence:
		return len(left.MustSequence()) == 0

	case dyn.KindNil:
		// map/sequence can be nil, for instance, bad YAML like: `foo:<eof>`
		return true

	default:
		return false
	}
}

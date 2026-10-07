package python

import (
	"fmt"

	"github.com/databricks/cli/bundle/config/mutator/resourcemutator"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
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
func applyPythonOutput(root, output structvar.View) (*structvar.OverridePlan, applyPythonOutputResult, error) {
	result, visitor := createOverrideVisitor(root, output)
	plan, err := structvar.PlanOverride(root, output, visitor)
	if err != nil {
		return nil, result, err
	}

	return plan, result, nil
}

// addResourceKeys adds the keys of the resources in root that match pattern.
func addResourceKeys(set resourcemutator.ResourceKeySet, pattern *structpath.PatternNode, root structvar.View) error {
	return structvar.ForEach(root, pattern, func(np *structpath.PathNode, _ structvar.View) error {
		set.AddResourceKey(resourcemutator.ResourceKey{Type: np.KeyAt(1), Name: np.KeyAt(2)})
		return nil
	})
}

func createOverrideVisitor(leftRoot, rightRoot structvar.View) (applyPythonOutputResult, structvar.OverrideVisitor) {
	resourcesPath := structpath.MustParsePath("resources")
	deleted := resourcemutator.NewResourceKeySet()
	updated := resourcemutator.NewResourceKeySet()
	added := resourcemutator.NewResourceKeySet()

	visitor := structvar.OverrideVisitor{
		VisitDelete: func(np *structpath.PathNode, left structvar.View) error {
			if isOmitemptyDelete(left) {
				return structvar.ErrOverrideUndoDelete
			}

			if !np.HasPrefix(resourcesPath) {
				return fmt.Errorf("unexpected change at %q (delete)", np.String())
			}

			// use leftRoot below because it contains deleted resources

			if np.Len() == 1 {
				// Example:
				//
				// valuePath: "resources"
				// leftRoot:  {"bundle": ..., "resources": ...},
				// rightRoot: {"bundle": ...}

				return addResourceKeys(deleted,
					structpath.MustParsePattern("resources.*.*"),
					leftRoot,
				)
			} else if np.Len() == 2 {
				// Example:
				//
				// valuePath: "resources.jobs"
				// leftRoot:  {"resources": { "jobs": ..., "pipeline": ...}}},
				// rightRoot: {"resources": { "jobs": ...}}},

				return addResourceKeys(deleted,
					structpath.NewPatternDotStar(structpath.NewPatternStringKey(structpath.MustParsePattern("resources"), np.KeyAt(1))),
					leftRoot,
				)
			} else if np.Len() == 3 {
				// Example: "resources.jobs.job_0"
				resourceKey := resourcemutator.ResourceKey{Type: np.KeyAt(1), Name: np.KeyAt(2)}
				deleted.AddResourceKey(resourceKey)

				return nil
			} else {
				// Example: "resources.jobs.job_0.tags"
				resourceKey := resourcemutator.ResourceKey{Type: np.KeyAt(1), Name: np.KeyAt(2)}
				updated.AddResourceKey(resourceKey)

				return nil
			}
		},
		VisitInsert: func(np *structpath.PathNode, right structvar.View) error {
			if !np.HasPrefix(resourcesPath) {
				return fmt.Errorf("unexpected change at %q (insert)", np.String())
			}

			// use rightRoot below because it contains result

			if np.Len() == 1 {
				// Example:
				//
				// valuePath: "resources"
				// leftRoot:  {"bundle": ...,                    }
				// rightRoot: {"bundle": ..., "resources": {...} }

				return addResourceKeys(added,
					structpath.MustParsePattern("resources.*.*"),
					rightRoot,
				)
			} else if np.Len() == 2 {
				// Example:
				//
				// valuePath: "resources.jobs"
				// leftRoot:  {"resources": {               }}
				// rightRoot: {"resources": { "jobs": {...} }}

				return addResourceKeys(added,
					structpath.NewPatternDotStar(structpath.NewPatternStringKey(structpath.MustParsePattern("resources"), np.KeyAt(1))),
					rightRoot,
				)
			} else if np.Len() == 3 {
				// Example:
				//
				// valuePath: "resources.jobs"
				// leftRoot:  {"resources": { "jobs": {               }}}
				// rightRoot: {"resources": { "jobs": {"job_0": {...} }}}
				resourceKey := resourcemutator.ResourceKey{Type: np.KeyAt(1), Name: np.KeyAt(2)}
				added.AddResourceKey(resourceKey)

				return nil
			} else {
				// Example: "resources.jobs.job_0.email_notifications"
				resourceKey := resourcemutator.ResourceKey{Type: np.KeyAt(1), Name: np.KeyAt(2)}
				updated.AddResourceKey(resourceKey)

				return nil
			}
		},
		VisitUpdate: func(np *structpath.PathNode, _, right structvar.View) error {
			if !np.HasPrefix(resourcesPath) {
				return fmt.Errorf("unexpected change at %q (update)", np.String())
			}

			// use rightRoot below because it contains result

			if np.Len() == 1 {
				// Example:
				//
				// valuePath: "resources"
				// leftRoot:  {"bundle": ..., "resources": null  }
				// rightRoot: {"bundle": ..., "resources": {...} }

				return addResourceKeys(added,
					structpath.MustParsePattern("resources.*.*"),
					rightRoot,
				)
			} else if np.Len() == 2 {
				// Example:
				//
				// valuePath: "resources.jobs"
				// leftRoot:  {"resources": { "jobs": null  }}
				// rightRoot: {"resources": { "jobs": {...} }}

				return addResourceKeys(added,
					structpath.NewPatternDotStar(structpath.NewPatternStringKey(structpath.MustParsePattern("resources"), np.KeyAt(1))),
					rightRoot,
				)
			} else if np.Len() == 3 {
				// Example:
				//
				// valuePath: "resources.jobs.job_0"
				// leftRoot:  {"resources": { "jobs": {"job_0": null  }}}
				// rightRoot: {"resources": { "jobs": {"job_0": {...} }}}
				resourceKey := resourcemutator.ResourceKey{Type: np.KeyAt(1), Name: np.KeyAt(2)}
				added.AddResourceKey(resourceKey)

				return nil
			} else {
				// Example: "resources.jobs.job_0.name"
				resourceKey := resourcemutator.ResourceKey{Type: np.KeyAt(1), Name: np.KeyAt(2)}
				updated.AddResourceKey(resourceKey)

				return nil
			}
		},
	}

	return applyPythonOutputResult{
		AddedResources:   added,
		UpdatedResources: updated,
		DeletedResources: deleted,
	}, visitor
}

func isOmitemptyDelete(left structvar.View) bool {
	// Python output can omit empty sequences/mappings, because we don't track them as optional,
	// there is no semantic difference between empty and missing, so we keep them as they were before
	// Python mutator deleted them.

	switch left.Kind() {
	case structvar.KindMap:
		for range left.MapItems() {
			return false
		}
		return true

	case structvar.KindSequence:
		for range left.Sequence() {
			return false
		}
		return true

	case structvar.KindNil:
		// map/sequence can be nil, for instance, bad YAML like: `foo:<eof>`
		return true

	default:
		return false
	}
}

package phases

import (
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/dyn"
)

// checkForPreventDestroy checks if the resource has lifecycle.prevent_destroy set, but the plan calls for this resource to be recreated or destroyed.
// If it does, it returns an error.
func checkForPreventDestroy(b *bundle.Bundle, actions []deployplan.Action, isDestroy bool) error {
	root := b.Config.Value()
	for _, action := range actions {
		if action.ActionType == deployplan.ActionTypeRecreate || (isDestroy && action.ActionType == deployplan.ActionTypeDelete) {
			path := dyn.NewPath(dyn.Key("resources"), dyn.Key(action.Group), dyn.Key(action.Name), dyn.Key("lifecycle"))
			lifecycleV, err := dyn.GetByPath(root, path)
			if err != nil {
				return err
			}
			if lifecycleV.Kind() == dyn.KindMap {
				preventDestroyV := lifecycleV.Get("prevent_destroy")
				preventDestroy, ok := preventDestroyV.AsBool()
				if ok && preventDestroy {
					return fmt.Errorf("resource %s has lifecycle.prevent_destroy set, but the plan calls for this resource to be recreated or destroyed. To avoid this error, disable lifecycle.prevent_destroy", action.Name)
				}
			}
		}
	}
	return nil
}

package tfdyn

import (
	"context"

	"github.com/databricks/cli/libs/dyn"
)

func convertLifecycle(ctx context.Context, vout, vLifecycle dyn.Value) (dyn.Value, error) {
	if !vLifecycle.IsValid() {
		return vout, nil
	}

	// Strip lifecycle.started: it is a DABs-only field not understood by Terraform.
	var err error
	vLifecycle, err = dyn.DropKeys(vLifecycle, []string{"started"})
	if err != nil {
		return dyn.InvalidValue, err
	}

	// If only lifecycle.started was set (now empty), skip setting the lifecycle block.
	if m, ok := vLifecycle.AsMap(); ok && m.Len() == 0 {
		return vout, nil
	}

	vout, err = dyn.Set(vout, "lifecycle", vLifecycle)
	if err != nil {
		return dyn.InvalidValue, err
	}

	return vout, nil
}

package tfdyn

import (
	"context"

	"github.com/databricks/cli/libs/dyn"
)

func convertLifecycle(ctx context.Context, vout, vLifecycle dyn.Value) (dyn.Value, error) {
	if !vLifecycle.IsValid() {
		return vout, nil
	}

	vout, err := dyn.Set(vout, "lifecycle", vLifecycle)
	if err != nil {
		return dyn.InvalidValue, err
	}

	return vout, nil
}

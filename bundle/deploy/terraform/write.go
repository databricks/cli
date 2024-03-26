package terraform

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

type write struct{}

func (w *write) Name() string {
	return "terraform.Write"
}

func (w *write) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	dir, err := Dir(ctx, b)
	if err != nil {
		return diag.FromErr(err)
	}

	var root *schema.Root
	err = b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		root, err = BundleToTerraformWithDynValue(ctx, v)
		return v, err
	})
	if err != nil {
		return diag.FromErr(err)
	}

	f, err := os.Create(filepath.Join(dir, TerraformConfigFileName))
	if err != nil {
		return diag.FromErr(err)
	}

	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	err = enc.Encode(root)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

// Write returns a [bundle.Mutator] that converts resources in a bundle configuration
// to the equivalent Terraform JSON representation and writes the result to a file.
func Write() bundle.Mutator {
	return &write{}
}

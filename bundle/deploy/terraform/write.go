package terraform

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/databricks/bricks/bundle"
)

type write struct{}

func (w *write) Name() string {
	return "terraform.Write"
}

func (w *write) Apply(ctx context.Context, b *bundle.Bundle) ([]bundle.Mutator, error) {
	dir, err := Dir(b)
	if err != nil {
		return nil, err
	}

	root := BundleToTerraform(&b.Config)
	f, err := os.Create(filepath.Join(dir, "bundle.tf.json"))
	if err != nil {
		return nil, err
	}

	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	err = enc.Encode(root)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

// Write returns a [bundle.Mutator] that converts resources in a bundle configuration
// to the equivalent Terraform JSON representation and writes the result to a file.
func Write() bundle.Mutator {
	return &write{}
}

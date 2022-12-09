package terraform

import (
	"os"
	"path/filepath"

	"github.com/databricks/bricks/bundle"
)

// Dir returns the Terraform working directory for a given bundle.
// The working directory is emphemeral and nested under the bundle's cache directory.
func Dir(b *bundle.Bundle) (string, error) {
	path, err := b.CacheDir()
	if err != nil {
		return "", err
	}

	nest := filepath.Join(path, "terraform")
	err = os.MkdirAll(nest, 0700)
	if err != nil {
		return "", err
	}

	return nest, nil
}

// Package schema fetches and loads the Terraform provider schema used by
// the codegen step.
package schema

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	tfjson "github.com/hashicorp/terraform-json"
)

// DatabricksProvider is the fully qualified Terraform provider address.
const DatabricksProvider = "registry.terraform.io/databricks/databricks"

// Schema describes the on-disk location of the provider schema artifacts.
type Schema struct {
	WorkingDir string

	ProviderSchemaFile string
}

// New creates a Schema rooted at ./tmp under the current working directory.
func New() (*Schema, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	tmpdir := filepath.Join(wd, "./tmp")
	err = os.MkdirAll(tmpdir, 0o755)
	if err != nil {
		return nil, err
	}

	return &Schema{
		WorkingDir:         tmpdir,
		ProviderSchemaFile: filepath.Join(tmpdir, "provider.json"),
	}, nil
}

// Load returns the parsed provider schema, fetching and generating it if needed.
func Load(ctx context.Context) (*tfjson.ProviderSchema, error) {
	s, err := New()
	if err != nil {
		return nil, err
	}

	// Generate schema file if it doesn't exist.
	if _, err := os.Stat(s.ProviderSchemaFile); errors.Is(err, fs.ErrNotExist) {
		err = s.Generate(ctx)
		if err != nil {
			return nil, err
		}
	}

	return s.Load(ctx)
}

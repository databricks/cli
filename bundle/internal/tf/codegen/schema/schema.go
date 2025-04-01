package schema

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	tfjson "github.com/hashicorp/terraform-json"
)

const DatabricksProvider = "registry.terraform.io/databricks/databricks"

type Schema struct {
	WorkingDir string

	ProviderSchemaFile string
}

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

package schema

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hc-install/product"
	"github.com/hashicorp/hc-install/releases"
	"github.com/hashicorp/terraform-exec/tfexec"
)

func (s *Schema) writeTerraformBlock(_ context.Context) error {
	body := map[string]any{
		"terraform": map[string]any{
			"required_providers": map[string]any{
				"databricks": map[string]any{
					"source":  "databricks/databricks",
					"version": ProviderVersion,
				},
			},
		},
	}

	buf, err := json.MarshalIndent(body, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(s.WorkingDir, "main.tf.json"), buf, 0o644)
}

func (s *Schema) installTerraform(ctx context.Context) (path string, err error) {
	installDir := filepath.Join(s.WorkingDir, "bin")
	err = os.MkdirAll(installDir, 0o755)
	if err != nil {
		return
	}

	installer := &releases.ExactVersion{
		Product:    product.Terraform,
		Version:    version.Must(version.NewVersion("1.5.5")),
		InstallDir: installDir,
	}

	installer.SetLogger(log.Default())

	path, err = installer.Install(ctx)
	return
}

func (s *Schema) generateSchema(ctx context.Context, execPath string) error {
	tf, err := tfexec.NewTerraform(s.WorkingDir, execPath)
	if err != nil {
		return err
	}

	log.Printf("running `terraform init`")
	err = tf.Init(ctx, tfexec.Upgrade(true))
	if err != nil {
		return err
	}

	log.Printf("acquiring provider schema")
	schemas, err := tf.ProvidersSchema(ctx)
	if err != nil {
		return err
	}

	// Find the databricks provider definition.
	_, ok := schemas.Schemas[DatabricksProvider]
	if !ok {
		return fmt.Errorf("schema file doesn't include schema for %s", DatabricksProvider)
	}

	buf, err := json.MarshalIndent(schemas, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.ProviderSchemaFile, buf, 0o644)
}

func (s *Schema) Generate(ctx context.Context) error {
	var err error

	err = s.writeTerraformBlock(ctx)
	if err != nil {
		return err
	}

	path, err := s.installTerraform(ctx)
	if err != nil {
		return err
	}

	err = s.generateSchema(ctx, path)
	if err != nil {
		return err
	}

	return nil
}

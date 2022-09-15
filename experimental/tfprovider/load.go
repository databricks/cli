package tfprovider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/hashicorp/hc-install/product"
	"github.com/hashicorp/hc-install/releases"
	"github.com/hashicorp/terraform-exec/tfexec"
	tfjson "github.com/hashicorp/terraform-json"
)

const DatabricksProvider = "registry.terraform.io/databricks/databricks"

func writeTfJson(path string) error {
	var body = map[string]interface{}{
		"terraform": map[string]interface{}{
			"required_providers": map[string]interface{}{
				"databricks": map[string]interface{}{
					"source":  "databricks/databricks",
					"version": ">= 1.0.0",
				},
			},
		},
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}

	defer f.Close()

	enc := json.NewEncoder(f)
	err = enc.Encode(body)
	if err != nil {
		return err
	}

	return nil
}

func ProduceProviderSchema() (*tfjson.ProviderSchema, error) {
	dir := "./tmp"
	err := os.Mkdir(dir, 0755)
	if err != nil && !errors.Is(err, os.ErrExist) {
		log.Fatal(err)
	}

	err = writeTfJson(filepath.Join(dir, "main.tf.json"))
	if err != nil {
		log.Fatalf("writing main.tf.json: %s", err)
	}

	installer := &releases.LatestVersion{
		Product: product.Terraform,
	}

	log.Printf("[INFO] Installing Terraform")
	execPath, err := installer.Install(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	tf, err := tfexec.NewTerraform(dir, execPath)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("[INFO] Running `terraform init`")
	err = tf.Init(context.Background(), tfexec.Upgrade(true))
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("[INFO] Acquiring provider schema")
	schemas, err := tf.ProvidersSchema(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	// Find the databricks provider definition.
	schema, ok := schemas.Schemas[DatabricksProvider]
	if !ok {
		return nil, fmt.Errorf("schema file doesn't include schema for %s", DatabricksProvider)
	}

	return schema, nil
}

func LoadSchema(path string) (*tfjson.ProviderSchema, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	var document tfjson.ProviderSchemas
	dec := json.NewDecoder(f)
	err = dec.Decode(&document)
	if err != nil {
		return nil, err
	}

	err = document.Validate()
	if err != nil {
		return nil, fmt.Errorf("invalid schema: %w", err)
	}

	// Find the databricks provider definition.
	schema, ok := document.Schemas[DatabricksProvider]
	if !ok {
		return nil, fmt.Errorf("schema file doesn't include schema for %s", DatabricksProvider)
	}

	return schema, nil
}

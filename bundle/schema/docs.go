package schema

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"reflect"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/jsonschema"
	"github.com/databricks/databricks-sdk-go/openapi"
)

// A subset of Schema struct
type Docs struct {
	Description          string           `json:"description"`
	Properties           map[string]*Docs `json:"properties,omitempty"`
	Items                *Docs            `json:"items,omitempty"`
	AdditionalProperties *Docs            `json:"additionalproperties,omitempty"`
}

//go:embed docs/bundle_descriptions.json
var bundleDocs []byte

func (docs *Docs) refreshTargetsDocs() error {
	targetsDocs, ok := docs.Properties["targets"]
	if !ok || targetsDocs.AdditionalProperties == nil ||
		targetsDocs.AdditionalProperties.Properties == nil {
		return fmt.Errorf("invalid targets descriptions")
	}
	targetProperties := targetsDocs.AdditionalProperties.Properties
	propertiesToCopy := []string{"artifacts", "bundle", "resources", "workspace"}
	for _, p := range propertiesToCopy {
		targetProperties[p] = docs.Properties[p]
	}
	return nil
}

func LoadBundleDescriptions() (*Docs, error) {
	embedded := Docs{}
	err := json.Unmarshal(bundleDocs, &embedded)
	return &embedded, err
}

func UpdateBundleDescriptions(openapiSpecPath string) (*Docs, error) {
	embedded, err := LoadBundleDescriptions()
	if err != nil {
		return nil, err
	}

	// Generate schema from the embedded descriptions, and convert it back to docs.
	// This creates empty descriptions for any properties that were missing in the
	// embedded descriptions.
	schema, err := New(reflect.TypeOf(config.Root{}), embedded)
	if err != nil {
		return nil, err
	}
	docs := schemaToDocs(schema)

	// Load the Databricks OpenAPI spec
	openapiSpec, err := os.ReadFile(openapiSpecPath)
	if err != nil {
		return nil, err
	}
	spec := &openapi.Specification{}
	err = json.Unmarshal(openapiSpec, spec)
	if err != nil {
		return nil, err
	}
	openapiReader := &OpenapiReader{
		OpenapiSpec: spec,
		Memo:        make(map[string]*jsonschema.Schema),
	}

	// Generate descriptions for the "resources" field
	resourcesDocs, err := openapiReader.ResourcesDocs()
	if err != nil {
		return nil, err
	}
	resourceSchema, err := New(reflect.TypeOf(config.Resources{}), resourcesDocs)
	if err != nil {
		return nil, err
	}
	docs.Properties["resources"] = schemaToDocs(resourceSchema)
	docs.refreshTargetsDocs()
	return docs, nil
}

// *Docs are a subset of *Schema, this function selects that subset
func schemaToDocs(jsonSchema *jsonschema.Schema) *Docs {
	// terminate recursion if schema is nil
	if jsonSchema == nil {
		return nil
	}
	docs := &Docs{
		Description: jsonSchema.Description,
	}
	if len(jsonSchema.Properties) > 0 {
		docs.Properties = make(map[string]*Docs)
	}
	for k, v := range jsonSchema.Properties {
		docs.Properties[k] = schemaToDocs(v)
	}
	docs.Items = schemaToDocs(jsonSchema.Items)
	if additionalProperties, ok := jsonSchema.AdditionalProperties.(*jsonschema.Schema); ok {
		docs.AdditionalProperties = schemaToDocs(additionalProperties)
	}
	return docs
}

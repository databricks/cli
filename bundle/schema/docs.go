package schema

import (
	_ "embed"
	"encoding/json"
	"os"
	"reflect"

	"github.com/databricks/bricks/bundle/config"
	"gopkg.in/yaml.v3"
)

type Docs struct {
	Description          string           `json:"description"`
	Properties           map[string]*Docs `json:"properties,omitempty"`
	Items                *Docs            `json:"items,omitempty"`
	AdditionalProperties *Docs            `json:"additionalproperties,omitempty"`
}

func LoadDocs(path string) (*Docs, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	docs := Docs{}
	err = yaml.Unmarshal(bytes, &docs)
	if err != nil {
		return nil, err
	}
	return &docs, nil
}

//go:embed docs/bundle_descriptions.json
var bundleDocs []byte

func BundleDocs(openapiSpecPath string) (*Docs, error) {
	docs, err := initializeBundleDocs()
	if err != nil {
		return nil, err
	}
	if openapiSpecPath != "" {
		openapiSpec, err := os.ReadFile(openapiSpecPath)
		if err != nil {
			return nil, err
		}
		spec := &openapi{}
		err = json.Unmarshal(openapiSpec, spec)
		if err != nil {
			return nil, err
		}
		docs.Properties["resources"], err = spec.ResourcesDocs()
		if err != nil {
			return nil, err
		}
	}
	return docs, nil
}

func initializeBundleDocs() (*Docs, error) {
	// load embedded descriptions
	embedded := Docs{}
	err := yaml.Unmarshal(bundleDocs, &embedded)
	if err != nil {
		return nil, err
	}
	// generate schema with the embedded descriptions
	schema, err := New(reflect.TypeOf(config.Root{}), &embedded)
	if err != nil {
		return nil, err
	}
	// converting the schema back to docs. This creates empty descriptions
	// for any properties that were missing in the embedded descriptions
	docs := schemaToDocs(schema)
	return docs, nil
}

func schemaToDocs(schema *Schema) *Docs {
	if schema == nil {
		return nil
	}
	docs := &Docs{
		Description: schema.Description,
	}
	if len(schema.Properties) > 0 {
		docs.Properties = make(map[string]*Docs)
	}
	for k, v := range schema.Properties {
		docs.Properties[k] = schemaToDocs(v)
	}
	docs.Items = schemaToDocs(schema.Items)
	if additionalProperties, ok := schema.AdditionalProperties.(*Schema); ok {
		docs.AdditionalProperties = schemaToDocs(additionalProperties)
	}
	return docs
}

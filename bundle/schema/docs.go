package schema

import (
	_ "embed"
	"os"
	"reflect"

	"github.com/databricks/bricks/bundle/config"
	"gopkg.in/yaml.v3"
)

type Docs struct {
	Description          string           `json:"description"`
	Properties           map[string]*Docs `json:"properties,omitempty"`
	Items                *Docs            `json:"items,omitempty"`
	AdditionalProperties *Docs            `json:"additionalProperties,omitempty"`
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

//go:embed bundle_config_docs.yml
var bundleDocs []byte

func GetBundleDocs() (*Docs, error) {
	docs := Docs{}
	err := yaml.Unmarshal(bundleDocs, &docs)
	if err != nil {
		return nil, err
	}
	return &docs, nil
}

func InitializeBundleDocs() (*Docs, error) {
	savedDocs := Docs{}
	err := yaml.Unmarshal(bundleDocs, &savedDocs)
	if err != nil {
		return nil, err
	}
	schema, err := New(reflect.TypeOf(config.Root{}), &savedDocs)
	if err != nil {
		return nil, err
	}
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

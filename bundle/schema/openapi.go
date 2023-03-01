package schema

import (
	_ "embed"
	"fmt"
	"strings"
)

type Components struct {
	Schemas map[string]*Schema `json:"schemas"`
}

type openapi struct {
	Components *Components `json:"components"`
}

const SchemaPathPrefix = "#/components/schemas/"

func (spec *openapi) readSchema(path string) (*Schema, error) {
	schemaKey := strings.TrimPrefix(path, SchemaPathPrefix)
	schema, ok := spec.Components.Schemas[schemaKey]
	if !ok {
		return nil, fmt.Errorf("[ERROR] schema with path %s not found in openapi spec", path)
	}
	if schema.Reference != nil {
		return nil, fmt.Errorf("[ERROR] schema with root level references are not supported")
	}
	for k, v := range schema.Properties {
		if v.Reference != nil {
			childSchema, err := spec.readSchema(*v.Reference)
			if err != nil {
				return nil, err
			}
			childSchema.Description = v.Description
			schema.Properties[k] = childSchema
		}
	}
	if schema.Items != nil && schema.Items.Reference != nil {
		childSchema, err := spec.readSchema(*schema.Items.Reference)
		if err != nil {
			return nil, err
		}
		childSchema.Description = schema.Items.Description
		schema.Items = childSchema
	}
	additionalProperties, ok := schema.AdditionalProperties.(*Schema)
	if ok && additionalProperties.Reference != nil {
		childSchema, err := spec.readSchema(*additionalProperties.Reference)
		if err != nil {
			return nil, err
		}
		childSchema.Description = additionalProperties.Description
		schema.AdditionalProperties = childSchema
	}
	return schema, nil
}

// TODO: Add description for id here
func (spec *openapi) jobsDocs() (*Docs, error) {
	jobSettingsSchema, err := spec.readSchema(SchemaPathPrefix + "jobs.JobSettings")
	if err != nil {
		return nil, err
	}
	jobDocs := schemaToDocs(jobSettingsSchema)
	jobDocs.Properties["id"] = &Docs{
		Description: "the canonical identifier of the job",
	}
	jobsDocs := &Docs{
		Description:          "list of job definations",
		AdditionalProperties: jobDocs,
	}
	return jobsDocs, nil
}

func (spec *openapi) ResourcesDocs() (*Docs, error) {
	jobsDocs, err := spec.jobsDocs()
	if err != nil {
		return nil, err
	}

	return &Docs{
		Description: "specification of databricks resources to instantiate",
		Properties: map[string]*Docs{
			"jobs": jobsDocs,
		},
	}, nil
}

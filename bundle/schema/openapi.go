package schema

import (
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
			refSchema, err := spec.readSchema(*v.Reference)
			if err != nil {
				return nil, err
			}
			refSchema.Description = v.Description
			schema.Properties[k] = refSchema
		}
	}
	if schema.Items != nil && schema.Items.Reference != nil {
		refSchema, err := spec.readSchema(*schema.Items.Reference)
		if err != nil {
			return nil, err
		}
		refSchema.Description = schema.Items.Description
		schema.Items = refSchema
	}
	additionalProperties, ok := schema.AdditionalProperties.(*Schema)
	if ok && additionalProperties.Reference != nil {
		refSchema, err := spec.readSchema(*additionalProperties.Reference)
		if err != nil {
			return nil, err
		}
		refSchema.Description = additionalProperties.Description
		schema.AdditionalProperties = refSchema
	}
	return schema, nil
}

func (spec *openapi) jobsDocs() (*Docs, error) {
	jobSettingsSchema, err := spec.readSchema(SchemaPathPrefix + "jobs.JobSettings")
	if err != nil {
		return nil, err
	}
	jobDocs := schemaToDocs(jobSettingsSchema)
	// TODO: add description for id if needed.
	// Tracked in https://databricks.atlassian.net/browse/DECO-558
	jobsDocs := &Docs{
		Description:          "List of job definations",
		AdditionalProperties: jobDocs,
	}
	return jobsDocs, nil
}

func (spec *openapi) pipelinesDocs() (*Docs, error) {
	pipelineSpecSchema, err := spec.readSchema(SchemaPathPrefix + "pipelines.PipelineSpec")
	if err != nil {
		return nil, err
	}
	pipelineDocs := schemaToDocs(pipelineSpecSchema)
	// TODO: Two fields in resources.Pipeline have the json tag id. Clarify the
	// semantics and then add a description if needed. (https://databricks.atlassian.net/browse/DECO-558)
	pipelinesDocs := &Docs{
		Description:          "List of pipeline definations",
		AdditionalProperties: pipelineDocs,
	}
	return pipelinesDocs, nil
}

func (spec *openapi) ResourcesDocs() (*Docs, error) {
	jobsDocs, err := spec.jobsDocs()
	if err != nil {
		return nil, err
	}
	pipelinesDocs, err := spec.pipelinesDocs()
	if err != nil {
		return nil, err
	}

	return &Docs{
		Description: "Specification of databricks resources to instantiate",
		Properties: map[string]*Docs{
			"jobs":      jobsDocs,
			"pipelines": pipelinesDocs,
		},
	}, nil
}

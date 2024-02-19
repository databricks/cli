package schema

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/databricks/cli/libs/jsonschema"
	"github.com/databricks/databricks-sdk-go/openapi"
)

type OpenapiReader struct {
	OpenapiSpec *openapi.Specification
	Memo        map[string]*jsonschema.Schema
}

const SchemaPathPrefix = "#/components/schemas/"

func (reader *OpenapiReader) readOpenapiSchema(path string) (*jsonschema.Schema, error) {
	schemaKey := strings.TrimPrefix(path, SchemaPathPrefix)

	// return early if we already have a computed schema
	memoSchema, ok := reader.Memo[schemaKey]
	if ok {
		return memoSchema, nil
	}

	// check path is present in openapi spec
	openapiSchema, ok := reader.OpenapiSpec.Components.Schemas[schemaKey]
	if !ok {
		return nil, fmt.Errorf("schema with path %s not found in openapi spec", path)
	}

	// convert openapi schema to the native schema struct
	bytes, err := json.Marshal(*openapiSchema)
	if err != nil {
		return nil, err
	}
	jsonSchema := &jsonschema.Schema{}
	err = json.Unmarshal(bytes, jsonSchema)
	if err != nil {
		return nil, err
	}

	// A hack to convert a map[string]interface{} to *Schema
	// We rely on the type of a AdditionalProperties in downstream functions
	// to do reference interpolation
	_, ok = jsonSchema.AdditionalProperties.(map[string]interface{})
	if ok {
		b, err := json.Marshal(jsonSchema.AdditionalProperties)
		if err != nil {
			return nil, err
		}
		additionalProperties := &jsonschema.Schema{}
		err = json.Unmarshal(b, additionalProperties)
		if err != nil {
			return nil, err
		}
		jsonSchema.AdditionalProperties = additionalProperties
	}

	// store read schema into memo
	reader.Memo[schemaKey] = jsonSchema

	return jsonSchema, nil
}

// safe againt loops in refs
func (reader *OpenapiReader) safeResolveRefs(root *jsonschema.Schema, tracker *tracker) (*jsonschema.Schema, error) {
	if root.Reference == nil {
		return reader.traverseSchema(root, tracker)
	}
	key := *root.Reference

	// HACK to unblock CLI release (13th Feb 2024). This is temporary until proper
	// support for recursive types is added to the docs generator. PR: https://github.com/databricks/cli/pull/1204
	if strings.Contains(key, "ForEachTask") {
		return root, nil
	}

	if tracker.hasCycle(key) {
		// self reference loops can be supported however the logic is non-trivial because
		// cross refernce loops are not allowed (see: http://json-schema.org/understanding-json-schema/structuring.html#recursion)
		return nil, fmt.Errorf("references loop detected")
	}
	ref := *root.Reference
	description := root.Description
	tracker.push(ref, ref)

	// Mark reference nil, so we do not traverse this again. This is tracked
	// in the memo
	root.Reference = nil

	// unroll one level of reference
	selfRef, err := reader.readOpenapiSchema(ref)
	if err != nil {
		return nil, err
	}
	root = selfRef
	root.Description = description

	// traverse again to find new references
	root, err = reader.traverseSchema(root, tracker)
	if err != nil {
		return nil, err
	}
	tracker.pop(ref)
	return root, err
}

func (reader *OpenapiReader) traverseSchema(root *jsonschema.Schema, tracker *tracker) (*jsonschema.Schema, error) {
	// case primitive (or invalid)
	if root.Type != jsonschema.ObjectType && root.Type != jsonschema.ArrayType {
		return root, nil
	}
	// only root references are resolved
	if root.Reference != nil {
		return reader.safeResolveRefs(root, tracker)
	}
	// case struct
	if len(root.Properties) > 0 {
		for k, v := range root.Properties {
			childSchema, err := reader.safeResolveRefs(v, tracker)
			if err != nil {
				return nil, err
			}
			root.Properties[k] = childSchema
		}
	}
	// case array
	if root.Items != nil {
		itemsSchema, err := reader.safeResolveRefs(root.Items, tracker)
		if err != nil {
			return nil, err
		}
		root.Items = itemsSchema
	}
	// case map
	additionalProperties, ok := root.AdditionalProperties.(*jsonschema.Schema)
	if ok && additionalProperties != nil {
		valueSchema, err := reader.safeResolveRefs(additionalProperties, tracker)
		if err != nil {
			return nil, err
		}
		root.AdditionalProperties = valueSchema
	}
	return root, nil
}

func (reader *OpenapiReader) readResolvedSchema(path string) (*jsonschema.Schema, error) {
	root, err := reader.readOpenapiSchema(path)
	if err != nil {
		return nil, err
	}
	tracker := newTracker()
	tracker.push(path, path)
	root, err = reader.safeResolveRefs(root, tracker)
	if err != nil {
		return nil, tracker.errWithTrace(err.Error(), "")
	}
	return root, nil
}

func (reader *OpenapiReader) jobsDocs() (*Docs, error) {
	jobSettingsSchema, err := reader.readResolvedSchema(SchemaPathPrefix + "jobs.JobSettings")
	if err != nil {
		return nil, err
	}
	jobDocs := schemaToDocs(jobSettingsSchema)
	// TODO: add description for id if needed.
	// Tracked in https://github.com/databricks/cli/issues/242
	jobsDocs := &Docs{
		Description:          "List of Databricks jobs",
		AdditionalProperties: jobDocs,
	}
	return jobsDocs, nil
}

func (reader *OpenapiReader) pipelinesDocs() (*Docs, error) {
	pipelineSpecSchema, err := reader.readResolvedSchema(SchemaPathPrefix + "pipelines.PipelineSpec")
	if err != nil {
		return nil, err
	}
	pipelineDocs := schemaToDocs(pipelineSpecSchema)
	// TODO: Two fields in resources.Pipeline have the json tag id. Clarify the
	// semantics and then add a description if needed. (https://github.com/databricks/cli/issues/242)
	pipelinesDocs := &Docs{
		Description:          "List of DLT pipelines",
		AdditionalProperties: pipelineDocs,
	}
	return pipelinesDocs, nil
}

func (reader *OpenapiReader) experimentsDocs() (*Docs, error) {
	experimentSpecSchema, err := reader.readResolvedSchema(SchemaPathPrefix + "ml.Experiment")
	if err != nil {
		return nil, err
	}
	experimentDocs := schemaToDocs(experimentSpecSchema)
	experimentsDocs := &Docs{
		Description:          "List of MLflow experiments",
		AdditionalProperties: experimentDocs,
	}
	return experimentsDocs, nil
}

func (reader *OpenapiReader) modelsDocs() (*Docs, error) {
	modelSpecSchema, err := reader.readResolvedSchema(SchemaPathPrefix + "ml.Model")
	if err != nil {
		return nil, err
	}
	modelDocs := schemaToDocs(modelSpecSchema)
	modelsDocs := &Docs{
		Description:          "List of MLflow models",
		AdditionalProperties: modelDocs,
	}
	return modelsDocs, nil
}

func (reader *OpenapiReader) modelServingEndpointsDocs() (*Docs, error) {
	modelServingEndpointsSpecSchema, err := reader.readResolvedSchema(SchemaPathPrefix + "serving.CreateServingEndpoint")
	if err != nil {
		return nil, err
	}
	modelServingEndpointsDocs := schemaToDocs(modelServingEndpointsSpecSchema)
	modelServingEndpointsAllDocs := &Docs{
		Description:          "List of Model Serving Endpoints",
		AdditionalProperties: modelServingEndpointsDocs,
	}
	return modelServingEndpointsAllDocs, nil
}

func (reader *OpenapiReader) registeredModelDocs() (*Docs, error) {
	registeredModelsSpecSchema, err := reader.readResolvedSchema(SchemaPathPrefix + "catalog.CreateRegisteredModelRequest")
	if err != nil {
		return nil, err
	}
	registeredModelsDocs := schemaToDocs(registeredModelsSpecSchema)
	registeredModelsAllDocs := &Docs{
		Description:          "List of Registered Models",
		AdditionalProperties: registeredModelsDocs,
	}
	return registeredModelsAllDocs, nil
}

func (reader *OpenapiReader) ResourcesDocs() (*Docs, error) {
	jobsDocs, err := reader.jobsDocs()
	if err != nil {
		return nil, err
	}
	pipelinesDocs, err := reader.pipelinesDocs()
	if err != nil {
		return nil, err
	}
	experimentsDocs, err := reader.experimentsDocs()
	if err != nil {
		return nil, err
	}
	modelsDocs, err := reader.modelsDocs()
	if err != nil {
		return nil, err
	}
	modelServingEndpointsDocs, err := reader.modelServingEndpointsDocs()
	if err != nil {
		return nil, err
	}
	registeredModelsDocs, err := reader.registeredModelDocs()
	if err != nil {
		return nil, err
	}

	return &Docs{
		Description: "Collection of Databricks resources to deploy.",
		Properties: map[string]*Docs{
			"jobs":                    jobsDocs,
			"pipelines":               pipelinesDocs,
			"experiments":             experimentsDocs,
			"models":                  modelsDocs,
			"model_serving_endpoints": modelServingEndpointsDocs,
			"registered_models":       registeredModelsDocs,
		},
	}, nil
}

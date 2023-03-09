package schema

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/databricks/databricks-sdk-go/openapi"
)

type OpenapiReader struct {
	OpenapiSpec *openapi.Specification
	Memo        map[string]*Schema
}

const SchemaPathPrefix = "#/components/schemas/"

func (reader *OpenapiReader) readOpenapiSchema(path string) (*Schema, error) {
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
	jsonSchema := &Schema{}
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
		additionalProperties := &Schema{}
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
func (reader *OpenapiReader) safeResolveRefs(root *Schema, seenRefs map[string]struct{}) (*Schema, error) {
	if root.Reference == nil {
		return reader.traverseSchema(root, seenRefs)
	}
	key := *root.Reference
	_, ok := seenRefs[key]
	if ok {
		// self reference loops can be supported however the logic is non-trivial because
		// cross refernce loops are not allowed (see: http://json-schema.org/understanding-json-schema/structuring.html#recursion)
		return nil, fmt.Errorf("references loop detected")
	}
	ref := *root.Reference
	description := root.Description
	seenRefs[ref] = struct{}{}

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
	root, err = reader.traverseSchema(root, seenRefs)
	if err != nil {
		return nil, err
	}
	delete(seenRefs, ref)
	return root, err
}

func (reader *OpenapiReader) traverseSchema(root *Schema, seenRefs map[string]struct{}) (*Schema, error) {
	// case primitive (or invalid)
	if root.Type != Object && root.Type != Array {
		return root, nil
	}
	// only root references are resolved
	if root.Reference != nil {
		return reader.safeResolveRefs(root, seenRefs)
	}
	// case struct
	if len(root.Properties) > 0 {
		for k, v := range root.Properties {
			childSchema, err := reader.safeResolveRefs(v, seenRefs)
			if err != nil {
				return nil, err
			}
			root.Properties[k] = childSchema
		}
	}
	// case array
	if root.Items != nil {
		itemsSchema, err := reader.safeResolveRefs(root.Items, seenRefs)
		if err != nil {
			return nil, err
		}
		root.Items = itemsSchema
	}
	// case map
	additionionalProperties, ok := root.AdditionalProperties.(*Schema)
	if ok && additionionalProperties != nil {
		valueSchema, err := reader.safeResolveRefs(additionionalProperties, seenRefs)
		if err != nil {
			return nil, err
		}
		root.AdditionalProperties = valueSchema
	}
	return root, nil
}

func (reader *OpenapiReader) readResolvedSchema(path string) (*Schema, error) {
	root, err := reader.readOpenapiSchema(path)
	if err != nil {
		return nil, err
	}
	seenRefs := make(map[string]struct{})
	seenRefs[path] = struct{}{}
	root, err = reader.safeResolveRefs(root, seenRefs)
	if err != nil {
		trace := ""
		count := 0
		for k := range seenRefs {
			if count == len(seenRefs)-1 {
				trace += k
				break
			}
			trace += k + " -> "
			count++
		}
		return nil, fmt.Errorf("%s. schema ref trace: %s", err, trace)
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
	// Tracked in https://databricks.atlassian.net/browse/DECO-558
	jobsDocs := &Docs{
		Description:          "List of job definations",
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
	// semantics and then add a description if needed. (https://databricks.atlassian.net/browse/DECO-558)
	pipelinesDocs := &Docs{
		Description:          "List of pipeline definations",
		AdditionalProperties: pipelineDocs,
	}
	return pipelinesDocs, nil
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

	return &Docs{
		Description: "Specification of databricks resources to instantiate",
		Properties: map[string]*Docs{
			"jobs":      jobsDocs,
			"pipelines": pipelinesDocs,
		},
	}, nil
}

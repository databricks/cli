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

func (spec *openapi) readShallowSchema(path string) (*Schema, error) {
	schemaKey := strings.TrimPrefix(path, SchemaPathPrefix)
	schema, ok := spec.Components.Schemas[schemaKey]
	if !ok {
		return nil, fmt.Errorf("schema with path %s not found in openapi spec", path)
	}
	return schema, nil
}

// TODO: can use (and maybe need to use) openapi for require (eg. jobs_cluster_key)

// safe againt loops in refs
func (spec *openapi) safeResolveRefs(root *Schema, seenRefs map[string]struct{}) (*Schema, error) {
	if root.Reference == nil {
		return spec.traverseSchema(root, seenRefs)
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
	root.Reference = nil

	// unroll one level of reference
	selfRef, err := spec.readShallowSchema(ref)
	if err != nil {
		return nil, err
	}
	root = selfRef
	root.Description = description

	// traverse again to find new references
	root, err = spec.traverseSchema(root, seenRefs)
	if err != nil {
		return nil, err
	}
	delete(seenRefs, ref)
	return root, err
}

// TODO: add test for error with loop
func (spec *openapi) traverseSchema(root *Schema, seenRefs map[string]struct{}) (*Schema, error) {
	// case primitive (or invalid)
	if root.Type != Object && root.Type != Array {
		return root, nil
	}
	// only root references are resolved
	if root.Reference != nil {
		return spec.safeResolveRefs(root, seenRefs)
	}
	// case struct
	if len(root.Properties) > 0 {
		for k, v := range root.Properties {
			childSchema, err := spec.safeResolveRefs(v, seenRefs)
			if err != nil {
				return nil, err
			}
			root.Properties[k] = childSchema
		}
	}
	// case array
	if root.Items != nil {
		itemsSchema, err := spec.safeResolveRefs(root.Items, seenRefs)
		if err != nil {
			return nil, err
		}
		root.Items = itemsSchema
	}
	// case map
	additionionalProperties, ok := root.AdditionalProperties.(*Schema)
	if ok && additionionalProperties != nil {
		valueSchema, err := spec.safeResolveRefs(additionionalProperties, seenRefs)
		if err != nil {
			return nil, err
		}
		root.AdditionalProperties = valueSchema
	}
	return root, nil
}

func (spec *openapi) readResolvedSchema(path string) (*Schema, error) {
	root, err := spec.readShallowSchema(path)
	if err != nil {
		return nil, err
	}
	seenRefs := make(map[string]struct{})
	seenRefs[path] = struct{}{}
	root, err = spec.safeResolveRefs(root, seenRefs)
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

func (spec *openapi) jobsDocs() (*Docs, error) {
	jobSettingsSchema, err := spec.readResolvedSchema(SchemaPathPrefix + "jobs.JobSettings")
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
	pipelineSpecSchema, err := spec.readResolvedSchema(SchemaPathPrefix + "pipelines.PipelineSpec")
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

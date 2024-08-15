package schema

import "github.com/databricks/cli/libs/jsonschema"

type Specification struct {
	Components *Components `json:"components"`
}

type Components struct {
	Schemas map[string]*jsonschema.Schema `json:"schemas,omitempty"`
}

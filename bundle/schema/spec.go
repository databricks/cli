package schema

import "github.com/databricks/cli/libs/jsonschema"

type Node struct {
	Description string `json:"description,omitempty"`
	Preview     string `json:"x-databricks-preview,omitempty"`
	Ref         string `json:"$ref,omitempty"`

	// Currently it is only defined for top level schemas
	JsonPath string `json:"-"`
}

type Specification struct {
	Node
	Components *Components `json:"components"`
}

type Components struct {
	Node
	Parameters map[string]*Parameter         `json:"parameters,omitempty"`
	Responses  map[string]*Body              `json:"responses,omitempty"`
	Schemas    map[string]*jsonschema.Schema `json:"schemas,omitempty"`
}

type Parameter struct {
	Node
	Required     bool               `json:"required,omitempty"`
	In           string             `json:"in,omitempty"`
	Name         string             `json:"name,omitempty"`
	MultiSegment bool               `json:"x-databricks-multi-segment,omitempty"`
	Schema       *jsonschema.Schema `json:"schema,omitempty"`
}

type MediaType struct {
	Node
	Schema *jsonschema.Schema `json:"schema,omitempty"`
}

type MimeType string

const (
	MimeTypeJson        MimeType = "application/json"
	MimeTypeOctetStream MimeType = "application/octet-stream"
	MimeTypeTextPlain   MimeType = "text/plain"
)

type Body struct {
	Node
	Required bool                  `json:"required,omitempty"`
	Content  map[string]MediaType  `json:"content,omitempty"`
	Headers  map[string]*Parameter `json:"headers,omitempty"`
}

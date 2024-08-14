package schema

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Node struct {
	Description string `json:"description,omitempty"`
	Preview     string `json:"x-databricks-preview,omitempty"`
	Ref         string `json:"$ref,omitempty"`

	// Currently it is only defined for top level schemas
	JsonPath string `json:"-"`
}

type Specification struct {
	Node
	Paths      map[string]Path `json:"paths"`
	Components *Components     `json:"components"`
	Tags       []Tag           `json:"tags"`
}

type PathStyle string

const (
	// PathStyleRpc indicates that the endpoint is an RPC-style endpoint.
	// The endpoint path is an action, and the entity to act on is specified
	// in the request body.
	PathStyleRpc PathStyle = "rpc"

	// PathStyleRest indicates that the endpoint is a REST-style endpoint.
	// The endpoint path is a resource, and the operation to perform on the
	// resource is specified in the HTTP method.
	PathStyleRest PathStyle = "rest"
)

func (r *PathStyle) UnmarshalJSON(data []byte) error {
	var s string
	err := json.Unmarshal(data, &s)
	if err != nil {
		return fmt.Errorf("cannot unmarshal RequestStyle: %w", err)
	}
	switch s {
	case "rpc", "rest":
		*r = PathStyle(s)
	default:
		return fmt.Errorf("invalid RequestStyle: %s", s)
	}
	return nil
}

type Tag struct {
	Node
	Package             string    `json:"x-databricks-package"`
	PathStyle           PathStyle `json:"x-databricks-path-style"`
	Service             string    `json:"x-databricks-service"`
	ParentService       string    `json:"x-databricks-parent-service"`
	ControlPlaneService string    `json:"x-databricks-controlplane"`
	IsAccounts          bool      `json:"x-databricks-is-accounts"`
	Name                string    `json:"name"`
}

type Path struct {
	Node
	Parameters []Parameter `json:"parameters,omitempty"`
	Get        *Operation  `json:"get,omitempty"`
	Head       *Operation  `json:"head,omitempty"`
	Post       *Operation  `json:"post,omitempty"`
	Put        *Operation  `json:"put,omitempty"`
	Patch      *Operation  `json:"patch,omitempty"`
	Delete     *Operation  `json:"delete,omitempty"`
}

type fieldPath []string

func (fp fieldPath) String() string {
	return strings.Join(fp, ".")
}

// Operation is the equivalent of method
type Operation struct {
	Node
	Wait       *Wait       `json:"x-databricks-wait,omitempty"`
	Pagination *Pagination `json:"x-databricks-pagination,omitempty"`
	DataPlane  *DataPlane  `json:"x-databricks-dataplane,omitempty"`
	Shortcut   bool        `json:"x-databricks-shortcut,omitempty"`
	Crud       string      `json:"x-databricks-crud,omitempty"`
	JsonOnly   bool        `json:"x-databricks-cli-json-only,omitempty"`

	// The x-databricks-path-style field indicates whether the operation has a
	// RESTful path style or a RPC style. When specified, this overrides the
	// service-level setting. Valid values are "rest" and "rpc". "rest" means
	// that the operation has a RESTful path style, i.e. the path represents
	// a resource and the HTTP method represents an action on the resource.
	// "rpc" means that the operation has a RPC style, i.e. the path represents
	// an action and the request body represents the resource.
	PathStyle PathStyle `json:"x-databricks-path-style,omitempty"`

	// The x-databricks-request-type-name field defines the name to use for
	// the request type in the generated client. This may be specified only
	// if the operation does NOT have a request body, thus only uses a request
	// type to encapsulate path and query parameters.
	RequestTypeName string `json:"x-databricks-request-type-name,omitempty"`

	// For list APIs, the path to the field in the response entity that contains
	// the resource ID.
	IdField fieldPath `json:"x-databricks-id,omitempty"`

	// For list APIs, the path to the field in the response entity that contains
	// the user-friendly name of the resource.
	NameField fieldPath `json:"x-databricks-name,omitempty"`

	Summary     string           `json:"summary,omitempty"`
	OperationId string           `json:"operationId"`
	Tags        []string         `json:"tags"`
	Parameters  []Parameter      `json:"parameters,omitempty"`
	Responses   map[string]*Body `json:"responses"`
	RequestBody *Body            `json:"requestBody,omitempty"`
}

type Components struct {
	Node
	Parameters map[string]*Parameter `json:"parameters,omitempty"`
	Responses  map[string]*Body      `json:"responses,omitempty"`
	Schemas    map[string]*Schema    `json:"schemas,omitempty"`
}

type Schema struct {
	Node
	IsComputed       bool               `json:"x-databricks-computed,omitempty"`
	IsAny            bool               `json:"x-databricks-any,omitempty"`
	Type             string             `json:"type,omitempty"`
	Enum             []string           `json:"enum,omitempty"`
	AliasEnum        []string           `json:"x-databricks-alias-enum,omitempty"`
	EnumDescriptions map[string]string  `json:"x-databricks-enum-descriptions,omitempty"`
	Default          any                `json:"default,omitempty"`
	Example          any                `json:"example,omitempty"`
	Format           string             `json:"format,omitempty"`
	Required         []string           `json:"required,omitempty"`
	Properties       map[string]*Schema `json:"properties,omitempty"`
	ArrayValue       *Schema            `json:"items,omitempty"`
	MapValue         *Schema            `json:"additionalProperties,omitempty"`
}

type Parameter struct {
	Node
	Required     bool    `json:"required,omitempty"`
	In           string  `json:"in,omitempty"`
	Name         string  `json:"name,omitempty"`
	MultiSegment bool    `json:"x-databricks-multi-segment,omitempty"`
	Schema       *Schema `json:"schema,omitempty"`
}

type MediaType struct {
	Node
	Schema *Schema `json:"schema,omitempty"`
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

// Pagination is the Databricks OpenAPI Extension for retrieving
// lists of entities through multiple API calls
type Pagination struct {
	Offset    string   `json:"offset,omitempty"`
	Limit     string   `json:"limit,omitempty"`
	Results   string   `json:"results,omitempty"`
	Increment int      `json:"increment,omitempty"`
	Inline    bool     `json:"inline,omitempty"`
	Token     *Binding `json:"token,omitempty"`
}

// Wait is the Databricks OpenAPI Extension for long-running result polling
type Wait struct {
	Poll         string             `json:"poll"`
	Bind         string             `json:"bind"`
	BindResponse string             `json:"bindResponse,omitempty"`
	Binding      map[string]Binding `json:"binding,omitempty"`
	Field        []string           `json:"field"`
	Message      []string           `json:"message"`
	Success      []string           `json:"success"`
	Failure      []string           `json:"failure"`
	Timeout      int                `json:"timeout,omitempty"`
}

// Binding is a relationship between request and/or response
type Binding struct {
	Request  string `json:"request,omitempty"`
	Response string `json:"response,omitempty"`
}

// DataPlane is the Databricks OpenAPI Extension for direct access to DataPlane APIs
type DataPlane struct {
	ConfigMethod string   `json:"configMethod"`
	Fields       []string `json:"field"`
}

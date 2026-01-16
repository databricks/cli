package sdkdocs

import (
	"embed"
	"encoding/json"
	"fmt"
)

//go:embed sdk_docs_index.json
var indexFS embed.FS

// SDKDocsIndex represents the complete SDK documentation index.
type SDKDocsIndex struct {
	Version     string                 `json:"version"`
	GeneratedAt string                 `json:"generated_at"`
	Services    map[string]*ServiceDoc `json:"services"`
	Types       map[string]*TypeDoc    `json:"types"`
	Enums       map[string]*EnumDoc    `json:"enums"`
}

// ServiceDoc represents documentation for an API service.
type ServiceDoc struct {
	Name        string                `json:"name"`
	Description string                `json:"description"`
	Package     string                `json:"package"`
	Methods     map[string]*MethodDoc `json:"methods"`
}

// MethodDoc represents documentation for an API method.
type MethodDoc struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Signature   string      `json:"signature"`
	Parameters  []ParamDoc  `json:"parameters"`
	Returns     *ReturnDoc  `json:"returns,omitempty"`
	Example     string      `json:"example,omitempty"`
	HTTPMethod  string      `json:"http_method,omitempty"`
	HTTPPath    string      `json:"http_path,omitempty"`
}

// ParamDoc represents documentation for a method parameter.
type ParamDoc struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

// ReturnDoc represents documentation for a method return type.
type ReturnDoc struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

// TypeDoc represents documentation for a data type.
type TypeDoc struct {
	Name        string               `json:"name"`
	Package     string               `json:"package"`
	Description string               `json:"description"`
	Fields      map[string]*FieldDoc `json:"fields"`
}

// FieldDoc represents documentation for a struct field.
type FieldDoc struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
	OutputOnly  bool   `json:"output_only,omitempty"`
	Deprecated  bool   `json:"deprecated,omitempty"`
}

// EnumDoc represents documentation for an enum type.
type EnumDoc struct {
	Name        string   `json:"name"`
	Package     string   `json:"package"`
	Description string   `json:"description"`
	Values      []string `json:"values"`
}

// LoadIndex loads the embedded SDK documentation index.
func LoadIndex() (*SDKDocsIndex, error) {
	data, err := indexFS.ReadFile("sdk_docs_index.json")
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded SDK docs index: %w", err)
	}

	var index SDKDocsIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("failed to parse SDK docs index: %w", err)
	}

	return &index, nil
}

// GetMethod retrieves a method by its path (e.g., "jobs.Create").
func (idx *SDKDocsIndex) GetMethod(serviceName, methodName string) *MethodDoc {
	service, ok := idx.Services[serviceName]
	if !ok {
		return nil
	}
	return service.Methods[methodName]
}

// GetType retrieves a type by its full path (e.g., "jobs.CreateJob").
func (idx *SDKDocsIndex) GetType(typePath string) *TypeDoc {
	return idx.Types[typePath]
}

// GetEnum retrieves an enum by its full path.
func (idx *SDKDocsIndex) GetEnum(enumPath string) *EnumDoc {
	return idx.Enums[enumPath]
}

// GetService retrieves a service by name.
func (idx *SDKDocsIndex) GetService(serviceName string) *ServiceDoc {
	return idx.Services[serviceName]
}

// ListServices returns all service names.
func (idx *SDKDocsIndex) ListServices() []string {
	names := make([]string, 0, len(idx.Services))
	for name := range idx.Services {
		names = append(names, name)
	}
	return names
}

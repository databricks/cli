package legacytemplates

import (
	"context"
)

// ResourceMetadata defines metadata for a resource type including how it maps to YAML.
type ResourceMetadata struct {
	Type          ResourceType
	SpecKey       ResourceSpecKey
	YamlName      string
	Description   string
	FlagNames     []string
	VariableNames []string
	// BindingLines generates YAML binding lines for the resource.
	// Returns nil for resources without bindings (like UC Volume).
	BindingLines func(values []string) []string
}

// ResourceHandler defines the interface for handling a specific resource type.
type ResourceHandler interface {
	// Metadata returns the metadata for this resource type.
	Metadata() *ResourceMetadata

	// IsRequired checks if the template requires this resource type.
	IsRequired(tmpl *AppTemplateManifest) bool

	// PromptForValues prompts the user for resource values.
	// Returns nil if the resource is not required or no values are needed.
	PromptForValues(ctx context.Context) (*ResourceValue, error)

	// ErrorMessage returns the error message to show when the resource is required but not provided.
	ErrorMessage() string
}

// BaseResourceHandler provides a default implementation of ResourceHandler.
type BaseResourceHandler struct {
	metadata     *ResourceMetadata
	specKey      ResourceSpecKey
	promptFunc   func(context.Context) (string, error)
	errorMessage string
}

// Metadata returns the resource metadata.
func (h *BaseResourceHandler) Metadata() *ResourceMetadata {
	return h.metadata
}

// IsRequired checks if the template requires this resource.
func (h *BaseResourceHandler) IsRequired(tmpl *AppTemplateManifest) bool {
	return hasResourceSpec(tmpl, func(s *resourceSpec) bool {
		return hasSpecKey(s, h.specKey)
	})
}

// PromptForValues prompts for a single resource value.
func (h *BaseResourceHandler) PromptForValues(ctx context.Context) (*ResourceValue, error) {
	value, err := h.promptFunc(ctx)
	if err != nil {
		return nil, err
	}
	if value == "" {
		return nil, nil
	}
	return &ResourceValue{
		Type:   h.metadata.Type,
		Values: []string{value},
	}, nil
}

// ErrorMessage returns the error message for missing resource.
func (h *BaseResourceHandler) ErrorMessage() string {
	return h.errorMessage
}

// hasSpecKey checks if a resourceSpec has a specific spec key set.
func hasSpecKey(spec *resourceSpec, key ResourceSpecKey) bool {
	switch key {
	case SpecKeySQLWarehouse:
		return spec.SQLWarehouse != nil
	case SpecKeyServingEndpoint:
		return spec.ServingEndpoint != nil
	case SpecKeyExperiment:
		return spec.Experiment != nil
	case SpecKeyDatabase:
		return spec.Database != nil
	case SpecKeyUCVolume:
		return spec.UCSecurable != nil
	default:
		return false
	}
}

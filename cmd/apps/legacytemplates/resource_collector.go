package legacytemplates

import (
	"context"
	"errors"
)

// ResourceCollector handles gathering all required resources for a template using the resource registry.
type ResourceCollector struct {
	template      *AppTemplateManifest
	isInteractive bool
	providedFlags map[ResourceType][]string
	registry      *ResourceRegistry
}

// NewResourceCollector creates a new resource collector.
func NewResourceCollector(template *AppTemplateManifest, isInteractive bool, warehouseID, servingEndpoint, experimentID, instanceName, databaseName, ucVolume string) *ResourceCollector {
	// Build provided flags map from legacy parameters
	providedFlags := make(map[ResourceType][]string)
	if warehouseID != "" {
		providedFlags[ResourceTypeSQLWarehouse] = []string{warehouseID}
	}
	if servingEndpoint != "" {
		providedFlags[ResourceTypeServingEndpoint] = []string{servingEndpoint}
	}
	if experimentID != "" {
		providedFlags[ResourceTypeExperiment] = []string{experimentID}
	}
	if instanceName != "" || databaseName != "" {
		providedFlags[ResourceTypeDatabase] = []string{instanceName, databaseName}
	}
	if ucVolume != "" {
		providedFlags[ResourceTypeUCVolume] = []string{ucVolume}
	}

	return &ResourceCollector{
		template:      template,
		isInteractive: isInteractive,
		providedFlags: providedFlags,
		registry:      GetGlobalRegistry(),
	}
}

// CollectAll gathers all required resources for the template.
func (c *ResourceCollector) CollectAll(ctx context.Context) (*ResourceValues, error) {
	resources := NewResourceValues()

	// Iterate through all registered handlers
	for _, handler := range c.registry.All() {
		// Check if resource is required by template
		if !handler.IsRequired(c.template) {
			continue
		}

		metadata := handler.Metadata()

		// Check if value was provided via flags
		if providedValues, ok := c.providedFlags[metadata.Type]; ok {
			// Filter out empty strings
			nonEmpty := make([]string, 0, len(providedValues))
			for _, v := range providedValues {
				if v != "" {
					nonEmpty = append(nonEmpty, v)
				}
			}
			if len(nonEmpty) > 0 {
				resources.Set(metadata.Type, nonEmpty...)
				continue
			}
		}

		// In interactive mode, prompt for resource
		if c.isInteractive {
			value, err := handler.PromptForValues(ctx)
			if err != nil {
				return nil, err
			}
			if value != nil && !value.IsEmpty() {
				resources.Set(metadata.Type, value.Values...)
			}
			continue
		}

		// Non-interactive mode without value - return error
		return nil, errors.New(handler.ErrorMessage())
	}

	return resources, nil
}

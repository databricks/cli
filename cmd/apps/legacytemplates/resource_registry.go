package legacytemplates

import (
	"context"

	"github.com/databricks/cli/cmd/apps/internal"
	"github.com/databricks/cli/libs/apps/prompt"
)

// ResourceRegistry maintains a registry of all available resource handlers.
type ResourceRegistry struct {
	handlers map[ResourceType]ResourceHandler
}

// NewResourceRegistry creates a new resource registry.
func NewResourceRegistry() *ResourceRegistry {
	return &ResourceRegistry{
		handlers: make(map[ResourceType]ResourceHandler),
	}
}

// Register adds a resource handler to the registry.
func (r *ResourceRegistry) Register(handler ResourceHandler) {
	r.handlers[handler.Metadata().Type] = handler
}

// Get retrieves a resource handler by type.
func (r *ResourceRegistry) Get(resourceType ResourceType) (ResourceHandler, bool) {
	handler, ok := r.handlers[resourceType]
	return handler, ok
}

// All returns all registered handlers.
func (r *ResourceRegistry) All() []ResourceHandler {
	handlers := make([]ResourceHandler, 0, len(r.handlers))
	for _, handler := range r.handlers {
		handlers = append(handlers, handler)
	}
	return handlers
}

// globalRegistry is the singleton resource registry.
var globalRegistry = NewResourceRegistry()

// GetGlobalRegistry returns the global resource registry.
func GetGlobalRegistry() *ResourceRegistry {
	return globalRegistry
}

func init() {
	// Register SQL Warehouse resource
	globalRegistry.Register(&BaseResourceHandler{
		metadata: &ResourceMetadata{
			Type:          ResourceTypeSQLWarehouse,
			SpecKey:       SpecKeySQLWarehouse,
			YamlName:      internal.ResourceNameSQLWarehouse,
			Description:   "SQL Warehouse for analytics",
			FlagNames:     []string{"warehouse-id"},
			VariableNames: []string{"warehouse_id"},
			BindingLines: func(values []string) []string {
				return []string{
					"          sql_warehouse:",
					"            id: ${var.warehouse_id}",
					"            permission: CAN_USE",
				}
			},
		},
		specKey:      SpecKeySQLWarehouse,
		promptFunc:   prompt.PromptForWarehouse,
		errorMessage: "template requires a SQL warehouse. Please provide --warehouse-id",
	})

	// Register Serving Endpoint resource
	globalRegistry.Register(&BaseResourceHandler{
		metadata: &ResourceMetadata{
			Type:          ResourceTypeServingEndpoint,
			SpecKey:       SpecKeyServingEndpoint,
			YamlName:      internal.ResourceNameServingEndpoint,
			Description:   "Model serving endpoint",
			FlagNames:     []string{"serving-endpoint"},
			VariableNames: []string{"serving_endpoint_name"},
			BindingLines: func(values []string) []string {
				return []string{
					"          serving_endpoint:",
					"            name: ${var.serving_endpoint_name}",
					"            permission: CAN_QUERY",
				}
			},
		},
		specKey:      SpecKeyServingEndpoint,
		promptFunc:   prompt.PromptForServingEndpoint,
		errorMessage: "template requires a serving endpoint. Please provide --serving-endpoint",
	})

	// Register Experiment resource
	globalRegistry.Register(&BaseResourceHandler{
		metadata: &ResourceMetadata{
			Type:          ResourceTypeExperiment,
			SpecKey:       SpecKeyExperiment,
			YamlName:      internal.ResourceNameExperiment,
			Description:   "MLflow experiment",
			FlagNames:     []string{"experiment-id"},
			VariableNames: []string{"experiment_id"},
			BindingLines: func(values []string) []string {
				return []string{
					"          experiment:",
					"            experiment_id: ${var.experiment_id}",
					"            permission: CAN_MANAGE",
				}
			},
		},
		specKey:      SpecKeyExperiment,
		promptFunc:   prompt.PromptForExperiment,
		errorMessage: "template requires an MLflow experiment. Please provide --experiment-id",
	})

	// Register Database resource (special case with two values)
	globalRegistry.Register(&DatabaseResourceHandler{
		metadata: &ResourceMetadata{
			Type:          ResourceTypeDatabase,
			SpecKey:       SpecKeyDatabase,
			YamlName:      internal.ResourceNameDatabase,
			Description:   "Lakebase database",
			FlagNames:     []string{"instance-name", "database-name"},
			VariableNames: []string{"instance_name", "database_name"},
			BindingLines: func(values []string) []string {
				return []string{
					"          database:",
					"            database_name: ${var.database_name}",
					"            instance_name: ${var.instance_name}",
					"            permission: CAN_CONNECT_AND_CREATE",
				}
			},
		},
		specKey: SpecKeyDatabase,
	})

	// Register UC Volume resource
	globalRegistry.Register(&BaseResourceHandler{
		metadata: &ResourceMetadata{
			Type:          ResourceTypeUCVolume,
			SpecKey:       SpecKeyUCVolume,
			YamlName:      internal.ResourceNameUCVolume,
			Description:   "Unity Catalog volume",
			FlagNames:     []string{"uc-volume"},
			VariableNames: []string{"uc_volume"},
			BindingLines: func(values []string) []string {
				return []string{
					"          volume:",
					"            name: ${var.uc_volume}",
					"            permission: READ_WRITE",
				}
			},
		},
		specKey:      SpecKeyUCVolume,
		promptFunc:   prompt.PromptForUCVolume,
		errorMessage: "template requires a Unity Catalog volume. Please provide --uc-volume",
	})
}

// DatabaseResourceHandler is a special handler for database resources that have two values.
type DatabaseResourceHandler struct {
	metadata *ResourceMetadata
	specKey  ResourceSpecKey
}

// Metadata returns the resource metadata.
func (h *DatabaseResourceHandler) Metadata() *ResourceMetadata {
	return h.metadata
}

// IsRequired checks if the template requires this resource.
func (h *DatabaseResourceHandler) IsRequired(tmpl *AppTemplateManifest) bool {
	return hasResourceSpec(tmpl, func(s *resourceSpec) bool {
		return hasSpecKey(s, h.specKey)
	})
}

// PromptForValues prompts for both instance name and database name.
func (h *DatabaseResourceHandler) PromptForValues(ctx context.Context) (*ResourceValue, error) {
	instanceName, err := prompt.PromptForDatabaseInstance(ctx)
	if err != nil {
		return nil, err
	}

	databaseName, err := prompt.PromptForDatabaseName(ctx, instanceName)
	if err != nil {
		return nil, err
	}

	if instanceName == "" && databaseName == "" {
		return nil, nil
	}

	return &ResourceValue{
		Type:   h.metadata.Type,
		Values: []string{instanceName, databaseName},
	}, nil
}

// ErrorMessage returns the error message for missing resource.
func (h *DatabaseResourceHandler) ErrorMessage() string {
	return "template requires a database. Please provide both --instance-name and --database-name"
}

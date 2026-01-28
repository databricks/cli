package legacytemplates

import (
	"strings"
)

// Resource name constants for databricks.yml resource bindings.
const (
	resourceNameSQLWarehouse    = "sql-warehouse"
	resourceNameServingEndpoint = "serving-endpoint"
	resourceNameExperiment      = "experiment"
	resourceNameDatabase        = "database"
	resourceNameDatabaseName    = "database-name"
	resourceNameUCVolume        = "uc-volume"
)

// resourceBinding represents a single resource binding in databricks.yml.
type resourceBinding struct {
	name        string
	description string
	lines       []string // Comment lines for the binding
}

// resourceBindingsBuilder constructs databricks.yml resource bindings.
// Each add method appends a binding for a specific resource type.
type resourceBindingsBuilder struct {
	bindings []resourceBinding
}

func newResourceBindingsBuilder() *resourceBindingsBuilder {
	return &resourceBindingsBuilder{bindings: make([]resourceBinding, 0)}
}

func (b *resourceBindingsBuilder) addWarehouse(warehouseID string) {
	if warehouseID == "" {
		return
	}
	b.bindings = append(b.bindings, resourceBinding{
		name:        resourceNameSQLWarehouse,
		description: "SQL Warehouse for analytics",
		lines: []string{
			"          sql_warehouse:",
			"            id: ${var.warehouse_id}",
			"            permission: CAN_USE",
		},
	})
}

func (b *resourceBindingsBuilder) addServingEndpoint(endpoint string) {
	if endpoint == "" {
		return
	}
	b.bindings = append(b.bindings, resourceBinding{
		name:        resourceNameServingEndpoint,
		description: "Model serving endpoint",
		lines: []string{
			"          serving_endpoint:",
			"            name: ${var.serving_endpoint_name}",
			"            permission: CAN_QUERY",
		},
	})
}

func (b *resourceBindingsBuilder) addExperiment(experimentID string) {
	if experimentID == "" {
		return
	}
	b.bindings = append(b.bindings, resourceBinding{
		name:        resourceNameExperiment,
		description: "MLflow experiment",
		lines: []string{
			"          experiment:",
			"            experiment_id: ${var.experiment_id}",
			"            permission: CAN_MANAGE",
		},
	})
}

func (b *resourceBindingsBuilder) addDatabase(instanceName, databaseName string) {
	if instanceName == "" || databaseName == "" {
		return
	}
	b.bindings = append(b.bindings, resourceBinding{
		name:        resourceNameDatabase,
		description: "Lakebase database",
		lines: []string{
			"          database:",
			"            database_name: ${var.database_name}",
			"            instance_name: ${var.instance_name}",
			"            permission: CAN_CONNECT_AND_CREATE",
		},
	})
}

// build generates the resource bindings content for databricks.yml.
func (b *resourceBindingsBuilder) build() string {
	if len(b.bindings) == 0 {
		return ""
	}

	var result []string
	for _, binding := range b.bindings {
		result = append(result, "        - name: "+binding.name)
		result = append(result, "          description: "+binding.description)
		result = append(result, binding.lines...)
	}

	return strings.Join(result, "\n")
}

// variablesBuilder builds bundle variables for databricks.yml.
type variablesBuilder struct {
	variables []struct {
		name  string
		value string
	}
}

func newVariablesBuilder() *variablesBuilder {
	return &variablesBuilder{}
}

func (b *variablesBuilder) addWarehouse(warehouseID string) {
	if warehouseID != "" {
		b.variables = append(b.variables, struct {
			name  string
			value string
		}{"warehouse_id", warehouseID})
	}
}

func (b *variablesBuilder) addServingEndpoint(endpoint string) {
	if endpoint != "" {
		b.variables = append(b.variables, struct {
			name  string
			value string
		}{"serving_endpoint_name", endpoint})
	}
}

func (b *variablesBuilder) addExperiment(experimentID string) {
	if experimentID != "" {
		b.variables = append(b.variables, struct {
			name  string
			value string
		}{"experiment_id", experimentID})
	}
}

func (b *variablesBuilder) addDatabase(instanceName, databaseName string) {
	if databaseName != "" {
		b.variables = append(b.variables, struct {
			name  string
			value string
		}{"database_name", databaseName})
	}
	if instanceName != "" {
		b.variables = append(b.variables, struct {
			name  string
			value string
		}{"instance_name", instanceName})
	}
}

func (b *variablesBuilder) addUCVolume(volume string) {
	if volume != "" {
		b.variables = append(b.variables, struct {
			name  string
			value string
		}{"uc_volume", volume})
	}
}

// build generates the variables content for databricks.yml.
func (b *variablesBuilder) build() string {
	if len(b.variables) == 0 {
		return ""
	}

	var result []string
	for _, v := range b.variables {
		result = append(result, "  "+v.name+":")
		result = append(result, "    default: "+v.value)
	}

	return strings.Join(result, "\n")
}

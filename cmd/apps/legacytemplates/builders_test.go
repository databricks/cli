package legacytemplates

import (
	"strings"
	"testing"

	"github.com/databricks/cli/cmd/apps/internal"
	"github.com/stretchr/testify/assert"
)

func TestResourceBindingsBuilder(t *testing.T) {
	t.Run("empty builder", func(t *testing.T) {
		b := newResourceBindingsBuilder()
		result := b.build()
		assert.Equal(t, "", result)
	})

	t.Run("warehouse binding", func(t *testing.T) {
		b := newResourceBindingsBuilder()
		b.addWarehouse("abc123")
		result := b.build()

		assert.Contains(t, result, "- name: "+internal.ResourceNameSQLWarehouse)
		assert.Contains(t, result, "description: SQL Warehouse for analytics")
		assert.Contains(t, result, "sql_warehouse:")
		assert.Contains(t, result, "id: ${var.warehouse_id}")
		assert.Contains(t, result, "permission: CAN_USE")
	})

	t.Run("serving endpoint binding", func(t *testing.T) {
		b := newResourceBindingsBuilder()
		b.addServingEndpoint("my-endpoint")
		result := b.build()

		assert.Contains(t, result, "- name: "+internal.ResourceNameServingEndpoint)
		assert.Contains(t, result, "description: Model serving endpoint")
		assert.Contains(t, result, "serving_endpoint:")
		assert.Contains(t, result, "name: ${var.serving_endpoint_name}")
		assert.Contains(t, result, "permission: CAN_QUERY")
	})

	t.Run("experiment binding", func(t *testing.T) {
		b := newResourceBindingsBuilder()
		b.addExperiment("123456")
		result := b.build()

		assert.Contains(t, result, "- name: "+internal.ResourceNameExperiment)
		assert.Contains(t, result, "description: MLflow experiment")
		assert.Contains(t, result, "experiment:")
		assert.Contains(t, result, "experiment_id: ${var.experiment_id}")
		assert.Contains(t, result, "permission: CAN_MANAGE")
	})

	t.Run("database binding", func(t *testing.T) {
		b := newResourceBindingsBuilder()
		b.addDatabase("my-instance", "my-db")
		result := b.build()

		assert.Contains(t, result, "- name: "+internal.ResourceNameDatabase)
		assert.Contains(t, result, "description: Lakebase database")
		assert.Contains(t, result, "database:")
		assert.Contains(t, result, "database_name: ${var.database_name}")
		assert.Contains(t, result, "instance_name: ${var.instance_name}")
		assert.Contains(t, result, "permission: CAN_CONNECT_AND_CREATE")
	})

	t.Run("multiple bindings", func(t *testing.T) {
		b := newResourceBindingsBuilder()
		b.addWarehouse("wh-123")
		b.addExperiment("exp-456")
		result := b.build()

		// Check both bindings are present
		assert.Contains(t, result, internal.ResourceNameSQLWarehouse)
		assert.Contains(t, result, internal.ResourceNameExperiment)

		// Verify ordering (warehouse should come before experiment)
		warehouseIndex := strings.Index(result, internal.ResourceNameSQLWarehouse)
		experimentIndex := strings.Index(result, internal.ResourceNameExperiment)
		assert.Less(t, warehouseIndex, experimentIndex)
	})

	t.Run("skip empty values", func(t *testing.T) {
		b := newResourceBindingsBuilder()
		b.addWarehouse("")
		b.addServingEndpoint("")
		b.addExperiment("")
		result := b.build()

		assert.Equal(t, "", result)
	})

	t.Run("database requires both instance and database name", func(t *testing.T) {
		b := newResourceBindingsBuilder()
		b.addDatabase("instance-only", "")
		assert.Equal(t, "", b.build())

		b = newResourceBindingsBuilder()
		b.addDatabase("", "db-only")
		assert.Equal(t, "", b.build())

		b = newResourceBindingsBuilder()
		b.addDatabase("instance", "database")
		assert.NotEqual(t, "", b.build())
	})
}

func TestVariablesBuilder(t *testing.T) {
	t.Run("empty builder", func(t *testing.T) {
		b := newVariablesBuilder()
		result := b.build()
		assert.Equal(t, "", result)
	})

	t.Run("warehouse variable", func(t *testing.T) {
		b := newVariablesBuilder()
		b.addWarehouse("abc123")
		result := b.build()

		assert.Contains(t, result, "warehouse_id:")
		assert.Contains(t, result, "default: abc123")
	})

	t.Run("serving endpoint variable", func(t *testing.T) {
		b := newVariablesBuilder()
		b.addServingEndpoint("my-endpoint")
		result := b.build()

		assert.Contains(t, result, "serving_endpoint_name:")
		assert.Contains(t, result, "default: my-endpoint")
	})

	t.Run("experiment variable", func(t *testing.T) {
		b := newVariablesBuilder()
		b.addExperiment("123456")
		result := b.build()

		assert.Contains(t, result, "experiment_id:")
		assert.Contains(t, result, "default: 123456")
	})

	t.Run("database variables", func(t *testing.T) {
		b := newVariablesBuilder()
		b.addDatabase("my-instance", "my-db")
		result := b.build()

		assert.Contains(t, result, "database_name:")
		assert.Contains(t, result, "default: my-db")
		assert.Contains(t, result, "instance_name:")
		assert.Contains(t, result, "default: my-instance")
	})

	t.Run("uc volume variable", func(t *testing.T) {
		b := newVariablesBuilder()
		b.addUCVolume("/Volumes/catalog/schema/volume")
		result := b.build()

		assert.Contains(t, result, "uc_volume:")
		assert.Contains(t, result, "default: /Volumes/catalog/schema/volume")
	})

	t.Run("multiple variables", func(t *testing.T) {
		b := newVariablesBuilder()
		b.addWarehouse("wh-123")
		b.addExperiment("exp-456")
		b.addUCVolume("/Volumes/cat/sch/vol")
		result := b.build()

		assert.Contains(t, result, "warehouse_id:")
		assert.Contains(t, result, "experiment_id:")
		assert.Contains(t, result, "uc_volume:")
	})

	t.Run("skip empty values", func(t *testing.T) {
		b := newVariablesBuilder()
		b.addWarehouse("")
		b.addServingEndpoint("")
		b.addExperiment("")
		b.addDatabase("", "")
		b.addUCVolume("")
		result := b.build()

		assert.Equal(t, "", result)
	})

	t.Run("database with only database name", func(t *testing.T) {
		b := newVariablesBuilder()
		b.addDatabase("", "my-db")
		result := b.build()

		assert.Contains(t, result, "database_name:")
		assert.NotContains(t, result, "instance_name:")
	})

	t.Run("database with only instance name", func(t *testing.T) {
		b := newVariablesBuilder()
		b.addDatabase("my-instance", "")
		result := b.build()

		assert.Contains(t, result, "instance_name:")
		assert.NotContains(t, result, "database_name:")
	})
}

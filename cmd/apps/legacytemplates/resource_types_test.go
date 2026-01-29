package legacytemplates_test

import (
	"testing"

	"github.com/databricks/cli/cmd/apps/legacytemplates"
	"github.com/stretchr/testify/assert"
)

func TestResourceValueSingleValue(t *testing.T) {
	rv := &legacytemplates.ResourceValue{
		Type:   legacytemplates.ResourceTypeSQLWarehouse,
		Values: []string{"warehouse123"},
	}

	assert.Equal(t, "warehouse123", rv.SingleValue())
	assert.False(t, rv.IsEmpty())
}

func TestResourceValueMultipleValues(t *testing.T) {
	rv := &legacytemplates.ResourceValue{
		Type:   legacytemplates.ResourceTypeDatabase,
		Values: []string{"instance1", "db1"},
	}

	assert.Equal(t, "instance1", rv.SingleValue()) // Returns first value
	assert.False(t, rv.IsEmpty())
	assert.Len(t, rv.Values, 2)
}

func TestResourceValueEmpty(t *testing.T) {
	rv := &legacytemplates.ResourceValue{
		Type:   legacytemplates.ResourceTypeSQLWarehouse,
		Values: []string{},
	}

	assert.Equal(t, "", rv.SingleValue())
	assert.True(t, rv.IsEmpty())
}

func TestResourceValuesSetAndGet(t *testing.T) {
	rv := legacytemplates.NewResourceValues()

	// Set a single value
	rv.Set(legacytemplates.ResourceTypeSQLWarehouse, "warehouse123")

	// Get the value
	val := rv.Get(legacytemplates.ResourceTypeSQLWarehouse)
	assert.NotNil(t, val)
	assert.Equal(t, "warehouse123", val.SingleValue())
	assert.True(t, rv.Has(legacytemplates.ResourceTypeSQLWarehouse))
}

func TestResourceValuesSetMultiple(t *testing.T) {
	rv := legacytemplates.NewResourceValues()

	// Set multiple values for database
	rv.Set(legacytemplates.ResourceTypeDatabase, "instance1", "db1")

	// Get the value
	val := rv.Get(legacytemplates.ResourceTypeDatabase)
	assert.NotNil(t, val)
	assert.Len(t, val.Values, 2)
	assert.Equal(t, "instance1", val.Values[0])
	assert.Equal(t, "db1", val.Values[1])
}

func TestResourceValuesSetEmptyStrings(t *testing.T) {
	rv := legacytemplates.NewResourceValues()

	// Set with empty strings - should be filtered out
	rv.Set(legacytemplates.ResourceTypeSQLWarehouse, "", "warehouse123", "")

	val := rv.Get(legacytemplates.ResourceTypeSQLWarehouse)
	assert.NotNil(t, val)
	assert.Len(t, val.Values, 1)
	assert.Equal(t, "warehouse123", val.SingleValue())
}

func TestResourceValuesSetOnlyEmptyStrings(t *testing.T) {
	rv := legacytemplates.NewResourceValues()

	// Set with only empty strings - should not create entry
	rv.Set(legacytemplates.ResourceTypeSQLWarehouse, "", "")

	assert.False(t, rv.Has(legacytemplates.ResourceTypeSQLWarehouse))
	assert.Nil(t, rv.Get(legacytemplates.ResourceTypeSQLWarehouse))
}

func TestResourceValuesHas(t *testing.T) {
	rv := legacytemplates.NewResourceValues()

	// Initially no resources
	assert.False(t, rv.Has(legacytemplates.ResourceTypeSQLWarehouse))

	// Add a resource
	rv.Set(legacytemplates.ResourceTypeSQLWarehouse, "warehouse123")
	assert.True(t, rv.Has(legacytemplates.ResourceTypeSQLWarehouse))

	// Other resources still not present
	assert.False(t, rv.Has(legacytemplates.ResourceTypeExperiment))
}

func TestResourceValuesAll(t *testing.T) {
	rv := legacytemplates.NewResourceValues()

	// Set multiple resources
	rv.Set(legacytemplates.ResourceTypeSQLWarehouse, "warehouse123")
	rv.Set(legacytemplates.ResourceTypeExperiment, "exp456")

	all := rv.All()
	assert.Len(t, all, 2)
	assert.NotNil(t, all[legacytemplates.ResourceTypeSQLWarehouse])
	assert.NotNil(t, all[legacytemplates.ResourceTypeExperiment])
}

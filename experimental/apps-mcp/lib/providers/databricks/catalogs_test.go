package databricks

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatchFilter(t *testing.T) {
	t.Run("ExactMatch", func(t *testing.T) {
		assert.True(t, matchFilter("test_table", "test_table"))
		assert.True(t, matchFilter("TEST_TABLE", "test_table"))
		assert.True(t, matchFilter("test_table", "TEST_TABLE"))
	})

	t.Run("SubstringMatch", func(t *testing.T) {
		assert.True(t, matchFilter("test_table", "test"))
		assert.True(t, matchFilter("test_table", "table"))
		assert.True(t, matchFilter("test_table", "st_tab"))
		assert.False(t, matchFilter("test_table", "xyz"))
	})

	t.Run("WildcardAtEnd", func(t *testing.T) {
		assert.True(t, matchFilter("test_table", "test*"))
		assert.True(t, matchFilter("test_table_v2", "test*"))
		assert.False(t, matchFilter("prod_table", "test*"))
	})

	t.Run("WildcardAtStart", func(t *testing.T) {
		assert.True(t, matchFilter("test_table", "*table"))
		assert.True(t, matchFilter("my_test_table", "*table"))
		assert.False(t, matchFilter("table_test", "*table"))
	})

	t.Run("WildcardInMiddle", func(t *testing.T) {
		assert.True(t, matchFilter("test_user_table", "test*table"))
		assert.True(t, matchFilter("test_some_other_table", "test*table"))
		assert.False(t, matchFilter("test_table_v2", "test*table"))
		assert.False(t, matchFilter("my_test_table", "test*table"))
	})

	t.Run("MultipleWildcards", func(t *testing.T) {
		assert.True(t, matchFilter("test_user_data_table", "test*data*table"))
		assert.True(t, matchFilter("test_x_data_y_table", "test*data*table"))
		assert.False(t, matchFilter("test_table", "test*data*table"))
	})

	t.Run("OnlyWildcard", func(t *testing.T) {
		assert.True(t, matchFilter("anything", "*"))
		assert.True(t, matchFilter("", "*"))
	})

	t.Run("CaseInsensitive", func(t *testing.T) {
		assert.True(t, matchFilter("Test_Table", "test*table"))
		assert.True(t, matchFilter("TEST_TABLE", "test*table"))
		assert.True(t, matchFilter("test_table", "TEST*TABLE"))
	})

	t.Run("EmptyFilter", func(t *testing.T) {
		assert.True(t, matchFilter("test_table", ""))
	})

	t.Run("EmptyName", func(t *testing.T) {
		assert.False(t, matchFilter("", "test"))
		assert.True(t, matchFilter("", ""))
	})

	t.Run("ComplexPatterns", func(t *testing.T) {
		// Real-world patterns
		assert.True(t, matchFilter("user_events_2024", "user*2024"))
		assert.True(t, matchFilter("sales_fact_table", "*fact*"))
		assert.True(t, matchFilter("dim_customer", "dim_*"))
		assert.False(t, matchFilter("fact_sales", "dim_*"))
	})

	t.Run("WildcardEdgeCases", func(t *testing.T) {
		// Multiple consecutive wildcards
		assert.True(t, matchFilter("test_table", "test**table"))

		// Wildcard at both ends
		assert.True(t, matchFilter("test_user_table", "*user*"))
		assert.True(t, matchFilter("user", "*user*"))

		// Pattern longer than name
		assert.False(t, matchFilter("test", "test_table"))
	})

	t.Run("SpecialCharacters", func(t *testing.T) {
		assert.True(t, matchFilter("test-table", "test-table"))
		assert.True(t, matchFilter("test.table", "test.table"))
		assert.True(t, matchFilter("test_table_123", "*123"))
	})
}

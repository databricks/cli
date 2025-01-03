package template

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTemplateHelpDescriptions(t *testing.T) {
	expected := `- default-python: The default Python template for Notebooks / Delta Live Tables / Workflows
- default-sql: The default SQL template for .sql files that run with Databricks SQL
- dbt-sql: The dbt SQL template (databricks.com/blog/delivering-cost-effective-data-real-time-dbt-and-databricks)
- mlops-stacks: The Databricks MLOps Stacks template (github.com/databricks/mlops-stacks)`
	assert.Equal(t, expected, HelpDescriptions())
}

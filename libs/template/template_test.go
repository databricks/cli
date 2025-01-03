package template

import (
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/stretchr/testify/assert"
)

func TestTemplateHelpDescriptions(t *testing.T) {
	expected := `- default-python: The default Python template for Notebooks / Delta Live Tables / Workflows
- default-sql: The default SQL template for .sql files that run with Databricks SQL
- dbt-sql: The dbt SQL template (databricks.com/blog/delivering-cost-effective-data-real-time-dbt-and-databricks)
- mlops-stacks: The Databricks MLOps Stacks template (github.com/databricks/mlops-stacks)`
	assert.Equal(t, expected, HelpDescriptions())
}

func TestTemplateOptions(t *testing.T) {
	expected := []cmdio.Tuple{
		{Name: "default-python", Id: "The default Python template for Notebooks / Delta Live Tables / Workflows"},
		{Name: "default-sql", Id: "The default SQL template for .sql files that run with Databricks SQL"},
		{Name: "dbt-sql", Id: "The dbt SQL template (databricks.com/blog/delivering-cost-effective-data-real-time-dbt-and-databricks)"},
		{Name: "mlops-stacks", Id: "The Databricks MLOps Stacks template (github.com/databricks/mlops-stacks)"},
		{Name: "custom", Id: "Bring your own template"},
	}
	assert.Equal(t, expected, options())
}

func TestBundleInitIsRepoUrl(t *testing.T) {
	assert.True(t, isRepoUrl("git@github.com:databricks/cli.git"))
	assert.True(t, isRepoUrl("https://github.com/databricks/cli.git"))

	assert.False(t, isRepoUrl("./local"))
	assert.False(t, isRepoUrl("foo"))
}

func TestBundleInitRepoName(t *testing.T) {
	// Test valid URLs
	assert.Equal(t, "cli.git", repoName("git@github.com:databricks/cli.git"))
	assert.Equal(t, "cli", repoName("https://github.com/databricks/cli/"))

	// test invalid URLs. In these cases the error would be floated when the
	// git clone operation fails.
	assert.Equal(t, "git@github.com:databricks", repoName("git@github.com:databricks"))
	assert.Equal(t, "invalid-url", repoName("invalid-url"))
	assert.Equal(t, "www.github.com", repoName("https://www.github.com"))
}

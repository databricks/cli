package bundle

import (
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/stretchr/testify/assert"
)

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

func TestNativeTemplateOptions(t *testing.T) {
	expected := []cmdio.Tuple{
		{Name: "default-python", Id: "The default Python template for Notebooks / Delta Live Tables / Workflows"},
		{Name: "default-sql", Id: "The default SQL template for .sql files that run with Databricks SQL"},
		{Name: "dbt-sql", Id: "The dbt SQL template (databricks.com/blog/delivering-cost-effective-data-real-time-dbt-and-databricks)"},
		{Name: "mlops-stacks", Id: "The Databricks MLOps Stacks template (github.com/databricks/mlops-stacks)"},
		{Name: "custom...", Id: "Bring your own template"},
	}
	assert.Equal(t, expected, nativeTemplateOptions())
}

func TestGetNativeTemplateByName(t *testing.T) {
	assert.Equal(t, "https://github.com/databricks/mlops-stacks", getNativeTemplateByName("mlops-stacks").gitUrl)
	assert.Equal(t, "https://github.com/databricks/mlops-stacks", getNativeTemplateByName("mlops-stack").gitUrl)

	assert.Equal(t, "default-python", getNativeTemplateByName("default-python").name)
	assert.Equal(t, "The default Python template for Notebooks / Delta Live Tables / Workflows", getNativeTemplateByName("default-python").description)
	assert.Equal(t, "", getNativeTemplateByName("default-python").gitUrl)

	assert.Nil(t, getNativeTemplateByName("invalid"))
}

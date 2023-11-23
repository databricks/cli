package bundle

import (
	"testing"

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
	assert.Equal(t, []string{"default-python", "mlops-stacks"}, nativeTemplateOptions())
}

func TestNativeTemplateDescriptions(t *testing.T) {
	assert.Equal(t, "- default-python: The default Python template\n- mlops-stacks: The Databricks MLOps Stacks template (https://github.com/databricks/mlops-stacks)", nativeTemplateDescriptions())
}

func TestGetUrlForNativeTemplate(t *testing.T) {
	assert.Equal(t, "https://github.com/databricks/mlops-stacks", getUrlForNativeTemplate("mlops-stacks"))
	assert.Equal(t, "https://github.com/databricks/mlops-stacks", getUrlForNativeTemplate("mlops-stack"))
	assert.Equal(t, "", getUrlForNativeTemplate("default-python"))
	assert.Equal(t, "", getUrlForNativeTemplate("invalid"))
}

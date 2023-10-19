package tags

import (
	"testing"

	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
)

func TestForCloudAws(t *testing.T) {
	c := &config.Config{
		Host: "https://dbc-XXXXXXXX-YYYY.cloud.databricks.com/",
	}

	assert.Equal(t, awsTag, ForCloud(c))
}

func TestForCloudAzure(t *testing.T) {
	c := &config.Config{
		Host: "https://adb-xxx.y.azuredatabricks.net/",
	}

	assert.Equal(t, azureTag, ForCloud(c))
}

func TestForCloudGcp(t *testing.T) {
	c := &config.Config{
		Host: "https://123.4.gcp.databricks.com/",
	}

	assert.Equal(t, gcpTag, ForCloud(c))
}

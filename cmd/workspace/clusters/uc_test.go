package clusters

import (
	"testing"

	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/stretchr/testify/assert"
)

func TestIsCompatible(t *testing.T) {
	assert.True(t, IsUnityCatalogCompatible(&compute.ClusterDetails{
		SparkVersion:     "13.2.x-aarch64-scala2.12",
		DataSecurityMode: compute.DataSecurityModeUserIsolation,
	}, "13.0"))
	assert.False(t, IsUnityCatalogCompatible(&compute.ClusterDetails{
		SparkVersion:     "13.2.x-aarch64-scala2.12",
		DataSecurityMode: compute.DataSecurityModeNone,
	}, "13.0"))
	assert.False(t, IsUnityCatalogCompatible(&compute.ClusterDetails{
		SparkVersion:     "9.1.x-photon-scala2.12",
		DataSecurityMode: compute.DataSecurityModeNone,
	}, "13.0"))
	assert.False(t, IsUnityCatalogCompatible(&compute.ClusterDetails{
		SparkVersion:     "9.1.x-photon-scala2.12",
		DataSecurityMode: compute.DataSecurityModeNone,
	}, "10.0"))
	assert.False(t, IsUnityCatalogCompatible(&compute.ClusterDetails{
		SparkVersion:     "custom-9.1.x-photon-scala2.12",
		DataSecurityMode: compute.DataSecurityModeNone,
	}, "14.0"))
}

package dockercredentials

import (
	"testing"

	"github.com/databricks/databricks-sdk-go/common/environment"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRegistryHost(t *testing.T) {
	cases := []string{
		"123456789.container.us-west-2.cloud.databricks.com",
		"https://123456789.container.us-west-2.cloud.databricks.com",
		"123456789.container.us-west-2.cloud.databricks.com/v2/",
	}

	for _, input := range cases {
		t.Run(input, func(t *testing.T) {
			got, err := ParseRegistryHost(input)
			require.NoError(t, err)
			assert.Equal(t, Registry{
				WorkspaceID: "123456789",
				Host:        "123456789.container.us-west-2.cloud.databricks.com",
			}, got)
		})
	}
}

func TestParseRegistryHostSupportsAllDatabricksEnvironmentZones(t *testing.T) {
	for _, env := range environment.AllEnvironments() {
		dnsZone := env.DnsZone
		if dnsZone == "" {
			continue
		}
		t.Run(dnsZone, func(t *testing.T) {
			wantHost := "123456789.container.test-region" + dnsZone
			registry, err := ParseRegistryHost("https://" + wantHost + "/v2/")
			require.NoError(t, err)
			assert.Equal(t, Registry{
				WorkspaceID: "123456789",
				Host:        wantHost,
			}, registry)
		})
	}
}

func TestParseRegistryHostUsesLongestDNSZoneSuffix(t *testing.T) {
	got, err := ParseRegistryHost("123456789.container.us-west-2.staging.cloud.databricks.com")
	require.NoError(t, err)
	assert.Equal(t, Registry{
		WorkspaceID: "123456789",
		Host:        "123456789.container.us-west-2.staging.cloud.databricks.com",
	}, got)
}

func TestParseRegistryHostRejectsNonDARHost(t *testing.T) {
	_, err := ParseRegistryHost("registry.example.com")
	assert.ErrorContains(t, err, `"registry.example.com" is not a Databricks Artifact Registry host`)
}

func TestParseRegistryHostRejectsPluralContainersInfix(t *testing.T) {
	_, err := ParseRegistryHost("123.containers.us-west-2.cloud.databricks.com")
	assert.ErrorContains(t, err, `"123.containers.us-west-2.cloud.databricks.com" is not a Databricks Artifact Registry host`)
}

func TestParseRegistryHostRejectsInvalidLabels(t *testing.T) {
	_, err := ParseRegistryHost("-123.container.us-west-2.cloud.databricks.com")
	assert.ErrorContains(t, err, `"-123.container.us-west-2.cloud.databricks.com" is not a Databricks Artifact Registry host`)

	_, err = ParseRegistryHost("123.container.-us-west-2.cloud.databricks.com")
	assert.ErrorContains(t, err, `"123.container.-us-west-2.cloud.databricks.com" is not a Databricks Artifact Registry host`)
}

func TestNormalizeServerAddress(t *testing.T) {
	got, err := normalizeServerAddress("HTTPS://123.container.US-WEST-2.cloud.databricks.com/v2/")
	require.NoError(t, err)
	assert.Equal(t, "123.container.us-west-2.cloud.databricks.com", got)
}

func TestNormalizeServerAddressRejectsNonHTTPSURL(t *testing.T) {
	_, err := normalizeServerAddress("http://123.container.us-west-2.cloud.databricks.com")
	assert.ErrorContains(t, err, "unsupported registry URL scheme")
}

func TestNormalizeServerAddressRejectsPort(t *testing.T) {
	_, err := normalizeServerAddress("https://123.container.us-west-2.cloud.databricks.com:443")
	assert.ErrorContains(t, err, "registry address must not include a port")
}

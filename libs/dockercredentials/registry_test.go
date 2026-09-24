package dockercredentials

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistryDNSZoneMatching(t *testing.T) {
	zones := []string{".prod.test", ".dev.test", ".staging.prod.test"}
	for _, tt := range []struct {
		host string
		want string
	}{
		{host: "workspace.prod.test", want: ".prod.test"},
		{host: "workspace.dev.test", want: ".dev.test"},
		{host: "workspace.staging.prod.test", want: ".staging.prod.test"},
	} {
		got, ok := matchingDNSZone(tt.host, zones)
		require.True(t, ok)
		assert.Equal(t, tt.want, got)
	}
}

func TestRegistryHostForZone(t *testing.T) {
	assert.Equal(t, "123456789.container.us-west-2.prod.test", registryHostForZone("123456789", "us-west-2", ".prod.test"))
}

func TestRegistryHostInZones(t *testing.T) {
	got, err := registryHostInZones("123456789", "us-west-2", "workspace.prod.test", []string{".prod.test", ".dev.test"})
	require.NoError(t, err)
	assert.Equal(t, "123456789.container.us-west-2.prod.test", got)
}

func TestRegistryServesWorkspaceHostRequiresExactZone(t *testing.T) {
	tests := []struct {
		name     string
		registry Registry
		host     string
		zones    []string
		want     bool
	}{
		{name: "same AWS production zone", registry: Registry{DNSZone: ".prod.test"}, host: "https://workspace.prod.test", zones: []string{".prod.test", ".dev.test"}, want: true},
		{name: "AWS production versus development", registry: Registry{DNSZone: ".prod.test"}, host: "https://workspace.dev.test", zones: []string{".prod.test", ".dev.test"}},
		{name: "Azure production versus development", registry: Registry{DNSZone: ".azprod.test"}, host: "https://workspace.azdev.test", zones: []string{".azprod.test", ".azdev.test"}},
		{name: "GCP production versus development", registry: Registry{DNSZone: ".gcpprod.test"}, host: "https://workspace.gcpdev.test", zones: []string{".gcpprod.test", ".gcpdev.test"}},
		{name: "same development zone", registry: Registry{DNSZone: ".dev.test"}, host: "https://workspace.dev.test", zones: []string{".prod.test", ".dev.test"}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.registry.servesWorkspaceHostInZones(tt.host, tt.zones))
		})
	}
}

func TestRegistryHostRejectsEmptyParts(t *testing.T) {
	_, err := registryHostInZones("", "us-west-2", "workspace.prod.test", []string{".prod.test"})
	assert.ErrorContains(t, err, "workspace ID is required")
	_, err = registryHostInZones("123456789", "", "workspace.prod.test", []string{".prod.test"})
	assert.ErrorContains(t, err, "region is required")
	_, err = registryHostInZones("123456789", "us-west-2", "workspace.example.test", []string{".prod.test"})
	assert.ErrorContains(t, err, "is not a supported Databricks workspace host")
}

func TestRegistryHostForLoopbackTestServer(t *testing.T) {
	got, err := RegistryHost("123456789", "us-west-2", "http://127.0.0.1:8080")
	require.NoError(t, err)
	registry, err := ParseRegistryHost(got)
	require.NoError(t, err)
	assert.Equal(t, Registry{WorkspaceID: "123456789", Host: got, DNSZone: ".localhost"}, registry)
	assert.True(t, registry.ServesWorkspaceHost("http://127.0.0.1:8080"))
	assert.False(t, registry.ServesWorkspaceHost("https://workspace.example.test"))
}

func TestParseRegistryHostUsesInjectedZones(t *testing.T) {
	got, err := parseRegistryHost("https://123456789.container.us-west-2.staging.prod.test/v2/", []string{".prod.test", ".staging.prod.test"})
	require.NoError(t, err)
	assert.Equal(t, Registry{WorkspaceID: "123456789", Host: "123456789.container.us-west-2.staging.prod.test", DNSZone: ".staging.prod.test"}, got)
}

func TestParseRegistryHostRejectsInvalidInputs(t *testing.T) {
	zones := []string{".prod.test"}
	for _, input := range []string{
		"registry.example.test",
		"123.containers.us-west-2.prod.test",
		"-123.container.us-west-2.prod.test",
		"123.container.-us-west-2.prod.test",
	} {
		_, err := parseRegistryHost(input, zones)
		assert.ErrorContains(t, err, "is not a Databricks Artifact Registry host")
	}
}

func TestNormalizeServerAddress(t *testing.T) {
	got, err := normalizeServerAddress("HTTPS://123.container.US-WEST-2.prod.test/v2/")
	require.NoError(t, err)
	assert.Equal(t, "123.container.us-west-2.prod.test", got)
	_, err = normalizeServerAddress("http://123.container.prod.test")
	assert.ErrorContains(t, err, "unsupported registry URL scheme")
	_, err = normalizeServerAddress("https://123.container.prod.test:443")
	assert.ErrorContains(t, err, "registry address must not include a port")
}

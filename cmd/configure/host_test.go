package configure

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeHost(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Empty input
		{"", ""},
		{"   ", ""},

		// Already has https://
		{"https://example.databricks.com", "https://example.databricks.com"},
		{"https://example.databricks.com/", "https://example.databricks.com"},

		// Missing protocol (should add https://)
		{"example.databricks.com", "https://example.databricks.com"},
		{"  example.databricks.com  ", "https://example.databricks.com"},
		{"subdomain.example.databricks.com", "https://subdomain.example.databricks.com"},
		{"example.databricks.com/", "https://example.databricks.com"},

		// Paths, query strings, and anchors are stripped
		{"https://example.databricks.com/path", "https://example.databricks.com"},
		{"https://example.databricks.com/path/", "https://example.databricks.com"},
		{"https://example.databricks.com?query", "https://example.databricks.com"},
		{"https://example.databricks.com#anchor", "https://example.databricks.com"},

		// Edge cases
		{"https://", "https://"},
		{"example.com", "https://example.com"},
		{"http://localhost:8080", "http://localhost:8080"},
		{"http://localhost:8080/", "http://localhost:8080"},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result := normalizeHost(test.input)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestValidateHost(t *testing.T) {
	var err error

	// Must start with https://
	err = validateHost("/path")
	assert.ErrorContains(t, err, "must start with https://")
	err = validateHost("http://host")
	assert.ErrorContains(t, err, "must start with https://")
	err = validateHost("ftp://host")
	assert.ErrorContains(t, err, "must start with https://")

	// Must use empty path
	err = validateHost("https://host/path")
	assert.ErrorContains(t, err, "must use empty path")

	// Valid hosts
	err = validateHost("https://host")
	assert.NoError(t, err)
}

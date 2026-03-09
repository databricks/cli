package databrickscfg

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeHost(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Empty and whitespace.
		{"", ""},
		{"  ", ""},

		// Bare hostnames (no scheme).
		{"foo.com", "https://foo.com"},
		{"foo.com:8080", "https://foo.com:8080"},
		{"e2-dogfood.staging.cloud.databricks.com", "https://e2-dogfood.staging.cloud.databricks.com"},

		// With https:// scheme.
		{"https://foo.com", "https://foo.com"},
		{"https://foo.com/", "https://foo.com"},
		{"https://foo.com/path", "https://foo.com"},
		{"https://foo.com?q=1", "https://foo.com"},
		{"https://foo.com#frag", "https://foo.com"},
		{"https://foo.com:443", "https://foo.com:443"},

		// With http:// scheme (preserved for local dev).
		{"http://foo.com", "http://foo.com"},
		{"http://localhost:8080", "http://localhost:8080"},
		{"http://foo.com/path", "http://foo.com"},

		// Port preserved.
		{"http://foo:123", "http://foo:123"},

		// Whitespace trimmed.
		{"  https://foo.com  ", "https://foo.com"},
		{"  foo.com  ", "https://foo.com"},

		// Scheme is lowercased; host case is preserved (Go's url package behavior).
		{"HTTPS://FOO.COM", "https://FOO.COM"},

		// Idempotent.
		{"https://foo.com", "https://foo.com"},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			result := NormalizeHost(test.input)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestNormalizeHostIdempotent(t *testing.T) {
	inputs := []string{
		"foo.com",
		"https://foo.com/path?q=1#frag",
		"http://localhost:8080",
		"  HTTPS://FOO.COM  ",
	}

	for _, input := range inputs {
		first := NormalizeHost(input)
		second := NormalizeHost(first)
		assert.Equal(t, first, second, "NormalizeHost should be idempotent for input: %s", input)
	}
}

package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCamelToSnake(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"valueFrom", "value_from"},
		{"myValue", "my_value"},
		{"value", "value"},
		{"ID", "id"},
		{"myID", "my_id"},
		{"IDValue", "id_value"},
		{"HTTPServer", "http_server"},
		{"getHTTPResponseCode", "get_http_response_code"},
		{"HTTPSConnection", "https_connection"},
		{"", ""},
		{"a", "a"},
		{"A", "a"},
		{"aB", "a_b"},
		{"AB", "ab"},
		{"ABC", "abc"},
		{"ABc", "a_bc"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := CamelToSnake(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

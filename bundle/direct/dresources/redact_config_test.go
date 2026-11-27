package dresources

import (
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/config/variable"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedactSensitiveConfigValues(t *testing.T) {
	secrets := map[string]*resources.Secret{
		"s": {Secret: catalog.Secret{Name: "s", Value: "top-secret"}},
	}

	tests := []struct {
		name     string
		root     config.Root
		expected map[string]*variable.Variable
	}{
		{
			name: "sensitive variables are redacted, default included, others are not",
			root: config.Root{
				Variables: map[string]*variable.Variable{
					"catalog": {Default: "main", Value: "main"},
					"count":   {Default: 1, Value: 1},
					"secret":  {Sensitive: new(true), Value: "top-secret"},
					"header":  {Sensitive: new(true), Default: "Bearer top-secret", Value: "Bearer top-secret"},
				},
				Resources: config.Resources{Secrets: secrets},
			},
			expected: map[string]*variable.Variable{
				"catalog": {Default: "main", Value: "main"},
				"count":   {Default: 1, Value: 1},
				"secret":  {Sensitive: new(true), Value: "[redacted]"},
				"header":  {Sensitive: new(true), Default: "[redacted]", Value: "[redacted]"},
			},
		},
		{
			name: "sensitive variables are redacted without sensitive resource fields",
			root: config.Root{
				Variables: map[string]*variable.Variable{
					"catalog": {Default: "main", Value: "main"},
					"secret":  {Sensitive: new(true), Value: "top-secret"},
				},
			},
			expected: map[string]*variable.Variable{
				"catalog": {Default: "main", Value: "main"},
				"secret":  {Sensitive: new(true), Value: "[redacted]"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root, err := RedactSensitiveConfigValues(&tc.root)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, root.Variables)
			for _, s := range root.Resources.Secrets {
				assert.Equal(t, "[redacted]", s.Value)
			}
		})
	}
}

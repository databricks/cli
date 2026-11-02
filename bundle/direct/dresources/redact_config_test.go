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
			name: "no sensitive variables: every string variable is redacted",
			root: config.Root{
				Variables: map[string]*variable.Variable{
					"catalog": {Default: "main", Value: "main"},
					"count":   {Default: 1, Value: 1},
					"secret":  {Value: "top-secret"},
				},
				Resources: config.Resources{Secrets: secrets},
			},
			expected: map[string]*variable.Variable{
				"catalog": {Default: "main", Value: "[redacted]"},
				"count":   {Default: 1, Value: 1},
				"secret":  {Value: "[redacted]"},
			},
		},
		{
			name: "sensitive variables: only those are redacted",
			root: config.Root{
				Variables: map[string]*variable.Variable{
					"catalog": {Default: "main", Value: "main"},
					"secret":  {Sensitive: true, Value: "top-secret"},
					"header":  {Sensitive: true, Default: "Bearer top-secret", Value: "Bearer top-secret"},
				},
				Resources: config.Resources{Secrets: secrets},
			},
			expected: map[string]*variable.Variable{
				"catalog": {Default: "main", Value: "main"},
				"secret":  {Sensitive: true, Value: "[redacted]"},
				"header":  {Sensitive: true, Default: "[redacted]", Value: "[redacted]"},
			},
		},
		{
			name: "no sensitive fields: nothing is redacted",
			root: config.Root{
				Variables: map[string]*variable.Variable{
					"catalog": {Default: "main", Value: "main"},
				},
			},
			expected: map[string]*variable.Variable{
				"catalog": {Default: "main", Value: "main"},
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

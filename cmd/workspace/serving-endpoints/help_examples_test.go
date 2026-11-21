package serving_endpoints

import (
	"encoding/json"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Every example's --json body must be valid JSON, so users can copy it as is.
func TestHelpExamplesAreValidJSON(t *testing.T) {
	body := regexp.MustCompile(`(?s)--json '(.*?)'`)
	for name, text := range map[string]string{
		"create":                 createExamples,
		"update-config":          updateConfigExamples,
		"put-ai-gateway":         putAiGatewayExamples,
		"patch-telemetry-config": patchTelemetryConfigExamples,
		"query":                  queryExamples,
	} {
		t.Run(name, func(t *testing.T) {
			matches := body.FindAllStringSubmatch(text, -1)
			require.NotEmpty(t, matches)
			for _, m := range matches {
				assert.True(t, json.Valid([]byte(m[1])), "invalid JSON: %s", m[1])
			}
		})
	}
}

func TestNewCommandHelpIncludesExamples(t *testing.T) {
	cmd := New()
	for _, sub := range []string{"create", "update-config", "put-ai-gateway", "patch-telemetry-config", "query"} {
		c, _, err := cmd.Find([]string{sub})
		require.NoError(t, err)
		assert.Contains(t, c.Long, "Example", sub)
	}
}

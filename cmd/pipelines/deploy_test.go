package pipelines

import (
	"testing"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
)

func TestFormatOSSTemplateWarningMessage(t *testing.T) {
	d := diag.Diagnostic{
		Severity: diag.Warning,
		Summary:  "unknown field: definitions",
		Locations: []dyn.Location{
			{File: "test-pipeline.yml"},
		},
	}

	message := formatOSSTemplateWarningMessage(d)
	assert.Contains(t, message, "Detected test-pipeline.yml is formatted for OSS Spark pipelines")
}

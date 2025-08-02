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
		Paths:    []dyn.Path{dyn.EmptyPath},
		Locations: []dyn.Location{
			{File: "test-pipeline.yml"},
		},
	}

	message := formatOSSTemplateWarningMessage(d)
	assert.Contains(t, message, "test-pipeline.yml seems to be formatted for open-source Spark Declarative Pipelines.")
}

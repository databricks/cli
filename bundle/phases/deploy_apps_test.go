package phases

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/stretchr/testify/assert"
)

func TestSkippedAppsMessage(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Apps: map[string]*resources.App{
					"my_app":    {},
					"other_app": {},
				},
			},
		},
	}

	msg := skippedAppsMessage(b)

	expected := "Bundle contains 2 Apps, but --deploy-apps was not set, not deploying apps. To deploy, run:\n" +
		"  databricks bundle run my_app\n" +
		"  databricks bundle run other_app"
	assert.Equal(t, expected, msg)
}

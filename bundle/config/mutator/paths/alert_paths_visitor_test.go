package paths

import (
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
)

func TestVisitAlertPaths(t *testing.T) {
	root := config.Root{
		Resources: config.Resources{
			Alerts: map[string]*resources.Alert{
				"alert0": {
					FilePath: "foo.dbalert.json",
				},
			},
		},
	}

	actual := collectVisitedPaths(t, root, VisitAlertPaths)
	expected := []dyn.Path{
		dyn.MustPathFromString("resources.alerts.alert0.file_path"),
	}

	assert.ElementsMatch(t, expected, actual)
}

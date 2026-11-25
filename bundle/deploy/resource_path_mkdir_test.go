package deploy

import (
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/stretchr/testify/assert"
)

func TestNeedsResourcePath(t *testing.T) {
	assert.False(t, needsResourcePath(&config.Resources{}))
	assert.False(t, needsResourcePath(&config.Resources{Jobs: map[string]*resources.Job{"job": {}}}))
	assert.True(t, needsResourcePath(&config.Resources{Alerts: map[string]*resources.Alert{"alert": {}}}))
	assert.True(t, needsResourcePath(&config.Resources{Dashboards: map[string]*resources.Dashboard{"dashboard": {}}}))
}

package run

import (
	"testing"

	"github.com/databricks/bricks/bundle"
	"github.com/databricks/bricks/bundle/config"
	"github.com/databricks/bricks/bundle/config/resources"
	"github.com/stretchr/testify/assert"
)

func TestResourceCompletionsUnique(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"foo": {},
					"bar": {},
				},
			},
		},
	}

	assert.ElementsMatch(t, []string{"foo", "bar"}, ResourceCompletions(b))
}

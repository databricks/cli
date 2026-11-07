package deployplan_test

import (
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/stretchr/testify/assert"
)

func TestAppliedLine(t *testing.T) {
	for action, expected := range map[deployplan.ActionType]string{
		deployplan.Create:       "Created jobs.foo",
		deployplan.Update:       "Updated jobs.foo",
		deployplan.UpdateWithID: "Updated jobs.foo",
		deployplan.Resize:       "Resized jobs.foo",
		deployplan.Recreate:     "Recreated jobs.foo",
		deployplan.Delete:       "Deleted jobs.foo",
	} {
		t.Run(string(action), func(t *testing.T) {
			assert.Equal(t, expected, deployplan.AppliedLine("resources.jobs.foo", action))
		})
	}
}

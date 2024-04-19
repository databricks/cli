package mutator

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
)

func TestDefaultQueueing(t *testing.T) {
	m := DefaultQueueing()
	assert.IsType(t, &defaultQueueing{}, m)
}

func TestName(t *testing.T) {
	m := DefaultQueueing()
	assert.Equal(t, "DefaultQueueing", m.Name())
}

func TestApplyNoJobs(t *testing.T) {
	m := DefaultQueueing().(*defaultQueueing)
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{},
		},
	}
	d := m.Apply(context.Background(), b)
	assert.Len(t, d, 0)
	assert.Len(t, b.Config.Resources.Jobs, 0)
}

func TestApplyJobsAlreadyEnabled(t *testing.T) {
	m := DefaultQueueing().(*defaultQueueing)
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job": {
						JobSettings: &jobs.JobSettings{
							Queue: &jobs.QueueSettings{Enabled: true},
						},
					},
				},
			},
		},
	}
	d := m.Apply(context.Background(), b)
	assert.Len(t, d, 0)
	assert.True(t, b.Config.Resources.Jobs["job"].Queue.Enabled)
}

func TestApplyEnableQueueing(t *testing.T) {
	m := DefaultQueueing().(*defaultQueueing)
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job": {
						JobSettings: &jobs.JobSettings{},
					},
				},
			},
		},
	}
	d := m.Apply(context.Background(), b)
	assert.Len(t, d, 0)
	assert.NotNil(t, b.Config.Resources.Jobs["job"].Queue)
	assert.True(t, b.Config.Resources.Jobs["job"].Queue.Enabled)
}

func TestApplyWithMultipleJobs(t *testing.T) {
	m := DefaultQueueing().(*defaultQueueing)
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": {
						JobSettings: &jobs.JobSettings{
							Queue: &jobs.QueueSettings{Enabled: false},
						},
					},
					"job2": {
						JobSettings: &jobs.JobSettings{},
					},
					"job3": {
						JobSettings: &jobs.JobSettings{
							Queue: &jobs.QueueSettings{Enabled: true},
						},
					},
				},
			},
		},
	}
	d := m.Apply(context.Background(), b)
	assert.Len(t, d, 0)
	assert.False(t, b.Config.Resources.Jobs["job1"].Queue.Enabled)
	assert.True(t, b.Config.Resources.Jobs["job2"].Queue.Enabled)
	assert.True(t, b.Config.Resources.Jobs["job3"].Queue.Enabled)
}

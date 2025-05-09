package resourcemutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

type defaultQueueing struct{}

func DefaultQueueing() bundle.Mutator {
	return &defaultQueueing{}
}

func (m *defaultQueueing) Name() string {
	return "DefaultQueueing"
}

// Enable queueing for jobs by default, following the behavior from API 2.2+.
// As of 2024-04, we're still using API 2.1 which has queueing disabled by default.
// This mutator makes sure queueing is enabled by default before we can adopt API 2.2.
func (m *defaultQueueing) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	r := b.Config.Resources
	for i := range r.Jobs {
		if r.Jobs[i].Queue != nil {
			continue
		}
		r.Jobs[i].Queue = &jobs.QueueSettings{
			Enabled: true,
		}
	}
	return nil
}

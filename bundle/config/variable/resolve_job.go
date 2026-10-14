package variable

import (
	"context"
	"strconv"

	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/sdk-go/jobs/v2"
)

type resolveJob struct {
	name string
}

func (l resolveJob) Resolve(ctx context.Context, w *sdkclients.Clients) (string, error) {
	entity, err := findByName(w.Jobs.ListJobsIter(ctx, jobs.ListJobsRequest{}), l.name, "BaseJob", func(j *jobs.BaseJob) string {
		if j.Settings == nil {
			return ""
		}
		return utils.Deref(j.Settings.Name)
	})
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(utils.Deref(entity.JobId), 10), nil
}

func (l resolveJob) String() string {
	return "job: " + l.name
}

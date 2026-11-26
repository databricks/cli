package resourcemutator

import (
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
)

func init() {
	registerTargetModeExamples("postgres_snapshot_schedules", func(r *config.Resources) {
		r.PostgresSnapshotSchedules = map[string]*resources.PostgresSnapshotSchedule{
			"postgres_snapshot_schedule1": {
				PostgresSnapshotScheduleConfig: resources.PostgresSnapshotScheduleConfig{
					Branch: "projects/project1/branches/branch1",
				},
			},
		}
	})
}

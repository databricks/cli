package resources

import (
	"context"
	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/sdktypes/postgres/v1"
	sdkpostgres "github.com/databricks/sdk-go/postgres/v1"
	"net/url"

	"github.com/databricks/cli/libs/log"
)

type PostgresSnapshotScheduleConfig struct {
	// Branch is the branch whose automatic-snapshot schedule this resource manages.
	// Format: "projects/{project_id}/branches/{branch_id}". The schedule's resource
	// name (and this resource's ID) is "{branch}/snapshot-schedule".
	Branch string `json:"branch"`

	// Schedule is the set of cadences at which automatic snapshots are taken. An
	// empty set disables automatic snapshots. Order is not significant; when
	// several cadences fire together a single snapshot is taken, retained for the
	// longest of their retentions.
	Schedule []postgres.ScheduleCadence `json:"schedule,omitempty"`
}

type PostgresSnapshotSchedule struct {
	BaseResource
	ID string `json:"id,omitempty" bundle:"readonly"`
	PostgresSnapshotScheduleConfig
}

func (b *PostgresSnapshotSchedule) Exists(ctx context.Context, w *sdkclients.Clients, name string) (bool, error) {
	_, err := w.Postgres.GetSnapshotSchedule(ctx, sdkpostgres.GetSnapshotScheduleRequest{Name: new(name)})
	if err != nil {
		log.Debugf(ctx, "postgres snapshot schedule %s does not exist", name)
		return false, err
	}
	return true, nil
}

func (b *PostgresSnapshotSchedule) ResourceDescription() ResourceDescription {
	return ResourceDescription{
		SingularName:  "postgres_snapshot_schedule",
		PluralName:    "postgres_snapshot_schedules",
		SingularTitle: "Postgres snapshot schedule",
		PluralTitle:   "Postgres snapshot schedules",
	}
}

func (b *PostgresSnapshotSchedule) GetName() string {
	// The schedule has no name field of its own; its resource name is its ID
	// ("{branch}/snapshot-schedule").
	return b.ID
}

func (b *PostgresSnapshotSchedule) GetURL() (string, bool) {
	// The IDs in the API do not (yet) map to IDs in the web UI.
	return "", false
}

func (b *PostgresSnapshotSchedule) InitializeURL(_ url.URL) {
	// The IDs in the API do not (yet) map to IDs in the web UI.
}

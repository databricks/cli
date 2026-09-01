package resources

import (
	"context"
	"net/url"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/postgres"
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

	// ForceSendFields tracks zero-value top-level fields (branch) for the SDK's
	// marshal package.
	ForceSendFields []string `json:"-" url:"-"`
}

func (c *PostgresSnapshotScheduleConfig) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, c)
}

func (c *PostgresSnapshotScheduleConfig) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(c)
}

type PostgresSnapshotSchedule struct {
	BaseResource
	PostgresSnapshotScheduleConfig
}

func (b *PostgresSnapshotSchedule) Exists(ctx context.Context, w *databricks.WorkspaceClient, name string) (bool, error) {
	_, err := w.Postgres.GetSnapshotSchedule(ctx, postgres.GetSnapshotScheduleRequest{Name: name})
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
	// Snapshot schedules don't have a user-visible name field.
	return ""
}

func (b *PostgresSnapshotSchedule) GetURL() string {
	// The IDs in the API do not (yet) map to IDs in the web UI.
	return ""
}

func (b *PostgresSnapshotSchedule) InitializeURL(_ url.URL) {
	// The IDs in the API do not (yet) map to IDs in the web UI.
}

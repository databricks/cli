package dresources

import (
	"context"
	"strings"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/common/types/fieldmask"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/postgres"
)

// snapshotScheduleSuffix is the final segment of a snapshot schedule's resource
// name: "projects/{project_id}/branches/{branch_id}/snapshot-schedule".
const snapshotScheduleSuffix = "/snapshot-schedule"

// snapshotScheduleName returns the schedule's resource name for a branch. It
// trims any trailing slash on the branch name (e.g. a user-supplied
// "projects/p/branches/b/") so the suffix is not doubled up.
func snapshotScheduleName(branch string) string {
	return strings.TrimRight(branch, "/") + snapshotScheduleSuffix
}

// PostgresSnapshotScheduleRemote is the return type for DoRead. It carries all
// paths present in StateType (branch, schedule) so drift detection works, plus
// the schedule's resource name.
type PostgresSnapshotScheduleRemote struct {
	Branch   string                     `json:"branch,omitempty"`
	Schedule []postgres.ScheduleCadence `json:"schedule,omitempty"`
	Name     string                     `json:"name,omitempty"`
}

func (s *PostgresSnapshotScheduleRemote) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s PostgresSnapshotScheduleRemote) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

type ResourcePostgresSnapshotSchedule struct {
	client *databricks.WorkspaceClient
}

type PostgresSnapshotScheduleState = resources.PostgresSnapshotScheduleConfig

func (*ResourcePostgresSnapshotSchedule) New(client *databricks.WorkspaceClient) *ResourcePostgresSnapshotSchedule {
	return &ResourcePostgresSnapshotSchedule{client: client}
}

func (*ResourcePostgresSnapshotSchedule) PrepareState(input *resources.PostgresSnapshotSchedule) *PostgresSnapshotScheduleState {
	return &PostgresSnapshotScheduleState{
		Branch:          input.Branch,
		Schedule:        input.Schedule,
		ForceSendFields: input.ForceSendFields,
	}
}

func (*ResourcePostgresSnapshotSchedule) RemapState(remote *PostgresSnapshotScheduleRemote) *PostgresSnapshotScheduleState {
	return &PostgresSnapshotScheduleState{
		Branch:          remote.Branch,
		Schedule:        remote.Schedule,
		ForceSendFields: nil,
	}
}

// makePostgresSnapshotScheduleRemote converts the SDK SnapshotSchedule into the
// remote shape. The API addresses the schedule by "{branch}/snapshot-schedule";
// branch is derived by stripping that suffix so it participates in drift detection.
func makePostgresSnapshotScheduleRemote(schedule *postgres.SnapshotSchedule) *PostgresSnapshotScheduleRemote {
	return &PostgresSnapshotScheduleRemote{
		Branch:   strings.TrimSuffix(schedule.Name, snapshotScheduleSuffix),
		Schedule: schedule.Schedule,
		Name:     schedule.Name,
	}
}

func (r *ResourcePostgresSnapshotSchedule) DoRead(ctx context.Context, id string) (*PostgresSnapshotScheduleRemote, error) {
	schedule, err := r.client.Postgres.GetSnapshotSchedule(ctx, postgres.GetSnapshotScheduleRequest{Name: id})
	if err != nil {
		return nil, err
	}
	return makePostgresSnapshotScheduleRemote(schedule), nil
}

// updateSnapshotSchedule sets the branch's schedule to the given cadences and
// waits for the long-running operation to complete. It is the single API call
// behind create, update, and delete: there is no Create/DeleteSnapshotSchedule
// endpoint, so the schedule is managed entirely through UpdateSnapshotSchedule.
// schedule is the only updatable path, so a static mask is used.
func (r *ResourcePostgresSnapshotSchedule) updateSnapshotSchedule(ctx context.Context, name string, cadences []postgres.ScheduleCadence) (*postgres.SnapshotSchedule, error) {
	waiter, err := r.client.Postgres.UpdateSnapshotSchedule(ctx, postgres.UpdateSnapshotScheduleRequest{
		Name: name,
		SnapshotSchedule: postgres.SnapshotSchedule{
			Schedule: cadences,

			// Name is carried in the request's Name field, not the body.
			Name:            "",
			ForceSendFields: nil,
		},
		UpdateMask: fieldmask.FieldMask{
			Paths: []string{"schedule"},
		},
	})
	if err != nil {
		return nil, err
	}
	return waiter.Wait(ctx)
}

func (r *ResourcePostgresSnapshotSchedule) DoCreate(ctx context.Context, config *PostgresSnapshotScheduleState) (string, *PostgresSnapshotScheduleRemote, error) {
	// The schedule exists implicitly for every branch; "creating" the resource
	// means setting its cadences via UpdateSnapshotSchedule.
	result, err := r.updateSnapshotSchedule(ctx, snapshotScheduleName(config.Branch), config.Schedule)
	if err != nil {
		return "", nil, err
	}
	remote := makePostgresSnapshotScheduleRemote(result)
	return remote.Name, remote, nil
}

func (r *ResourcePostgresSnapshotSchedule) DoUpdate(ctx context.Context, id string, config *PostgresSnapshotScheduleState, _ *PlanEntry) (*PostgresSnapshotScheduleRemote, error) {
	result, err := r.updateSnapshotSchedule(ctx, id, config.Schedule)
	if err != nil {
		return nil, err
	}
	return makePostgresSnapshotScheduleRemote(result), nil
}

func (r *ResourcePostgresSnapshotSchedule) DoDelete(ctx context.Context, id string, _ *PostgresSnapshotScheduleState) error {
	// There is no DeleteSnapshotSchedule endpoint. DoDelete fires whenever the
	// resource leaves the desired state (removed from config, or bundle destroy),
	// so disable automatic snapshots by setting an empty cadence set — otherwise
	// the branch would keep taking snapshots after the resource is gone.
	_, err := r.updateSnapshotSchedule(ctx, id, nil)
	return err
}

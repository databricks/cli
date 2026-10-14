package dresources

import (
	"context"
	"strings"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/sdkclients"
	postgres "github.com/databricks/cli/libs/sdktypes/postgres/v1"
	"github.com/databricks/cli/libs/utils"
	sdkpostgres "github.com/databricks/sdk-go/postgres/v1"
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

type ResourcePostgresSnapshotSchedule struct {
	client *sdkclients.Clients
}

type PostgresSnapshotScheduleState = resources.PostgresSnapshotScheduleConfig

func (*ResourcePostgresSnapshotSchedule) New(client *sdkclients.Clients) *ResourcePostgresSnapshotSchedule {
	return &ResourcePostgresSnapshotSchedule{client: client}
}

func (*ResourcePostgresSnapshotSchedule) PrepareState(input *resources.PostgresSnapshotSchedule) *PostgresSnapshotScheduleState {
	return &PostgresSnapshotScheduleState{
		Branch:   input.Branch,
		Schedule: input.Schedule,
	}
}

func (*ResourcePostgresSnapshotSchedule) RemapState(remote *PostgresSnapshotScheduleRemote) *PostgresSnapshotScheduleState {
	return &PostgresSnapshotScheduleState{
		Branch:   remote.Branch,
		Schedule: remote.Schedule,
	}
}

// makePostgresSnapshotScheduleRemote converts the SDK SnapshotSchedule into the
// remote shape. The API addresses the schedule by "{branch}/snapshot-schedule";
// branch is derived by stripping that suffix so it participates in drift detection.
func makePostgresSnapshotScheduleRemote(resp *sdkpostgres.SnapshotSchedule) (*PostgresSnapshotScheduleRemote, error) {
	schedule, err := postgres.SnapshotScheduleFromSDK(resp)
	if err != nil {
		return nil, err
	}
	return &PostgresSnapshotScheduleRemote{
		Branch:   strings.TrimSuffix(utils.Deref(schedule.Name), snapshotScheduleSuffix),
		Schedule: schedule.Schedule,
		Name:     utils.Deref(schedule.Name),
	}, nil
}

func (r *ResourcePostgresSnapshotSchedule) DoRead(ctx context.Context, id string) (*PostgresSnapshotScheduleRemote, error) {
	schedule, err := r.client.Postgres.GetSnapshotSchedule(ctx, sdkpostgres.GetSnapshotScheduleRequest{Name: new(id)})
	if err != nil {
		return nil, err
	}
	return makePostgresSnapshotScheduleRemote(schedule)
}

// updateSnapshotSchedule sets the branch's schedule to the given cadences and
// waits for the long-running operation to complete. It is the single API call
// behind create, update, and delete: there is no Create/DeleteSnapshotSchedule
// endpoint, so the schedule is managed entirely through UpdateSnapshotSchedule.
// schedule is the only updatable path, so a static mask is used.
func (r *ResourcePostgresSnapshotSchedule) updateSnapshotSchedule(ctx context.Context, name string, cadences []postgres.ScheduleCadence) (*sdkpostgres.SnapshotSchedule, error) {
	request, err := postgres.UpdateSnapshotScheduleRequestToSDK(&postgres.UpdateSnapshotScheduleRequest{
		SnapshotSchedule: &postgres.SnapshotSchedule{Name: new(name), Schedule: cadences},
		UpdateMask:       new("schedule"),
	})
	if err != nil {
		return nil, err
	}
	waiter, err := r.client.Postgres.UpdateSnapshotSchedule(ctx, *request)
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
	remote, err := makePostgresSnapshotScheduleRemote(result)
	if err != nil {
		return "", nil, err
	}
	return remote.Name, remote, nil
}

func (r *ResourcePostgresSnapshotSchedule) DoUpdate(ctx context.Context, id string, config *PostgresSnapshotScheduleState, _ *PlanEntry) (*PostgresSnapshotScheduleRemote, error) {
	result, err := r.updateSnapshotSchedule(ctx, id, config.Schedule)
	if err != nil {
		return nil, err
	}
	return makePostgresSnapshotScheduleRemote(result)
}

func (r *ResourcePostgresSnapshotSchedule) DoDelete(ctx context.Context, id string, _ *PostgresSnapshotScheduleState) error {
	// There is no DeleteSnapshotSchedule endpoint. DoDelete fires whenever the
	// resource leaves the desired state (removed from config, or bundle destroy),
	// so disable automatic snapshots by setting an empty cadence set — otherwise
	// the branch would keep taking snapshots after the resource is gone.
	_, err := r.updateSnapshotSchedule(ctx, id, nil)
	return err
}

package dresources

import (
	"context"
	"fmt"
	"strconv"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

// JobRunState is what we persist for a triggered run: the RunNow request.
type JobRunState struct {
	jobs.RunNow
}

func (s *JobRunState) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s JobRunState) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

// JobRunRemote embeds RunNow so every StateType path is a valid RemoteType path
// (see TestRemoteSuperset), plus the run's output-only fields for a faithful view.
type JobRunRemote struct {
	jobs.RunNow

	RunId      int64          `json:"run_id,omitempty"`
	RunName    string         `json:"run_name,omitempty"`
	State      *jobs.RunState `json:"state,omitempty"`
	RunPageUrl string         `json:"run_page_url,omitempty"`
	RunType    jobs.RunType   `json:"run_type,omitempty"`
}

// Custom marshaler needed because embedded RunNow's MarshalJSON would otherwise
// take over and drop the additional fields.
func (s *JobRunRemote) UnmarshalJSON(b []byte) error {
	return marshal.Unmarshal(b, s)
}

func (s JobRunRemote) MarshalJSON() ([]byte, error) {
	return marshal.Marshal(s)
}

type ResourceJobRun struct {
	client *databricks.WorkspaceClient
}

func (*ResourceJobRun) New(client *databricks.WorkspaceClient) *ResourceJobRun {
	return &ResourceJobRun{
		client: client,
	}
}

func (*ResourceJobRun) PrepareState(input *resources.JobRun) *JobRunState {
	return &JobRunState{
		RunNow: input.RunNow,
	}
}

// makeJobRunRemote maps the GetRun response into the RunNow-shaped remote: GET
// nests the params under overriding_parameters and returns job_parameters as a
// list, so both are flattened back into RunNow.
func makeJobRunRemote(run *jobs.Run) *JobRunRemote {
	var overriding jobs.RunParameters
	if run.OverridingParameters != nil {
		overriding = *run.OverridingParameters
	}
	var jobParameters map[string]string
	if len(run.JobParameters) > 0 {
		jobParameters = make(map[string]string, len(run.JobParameters))
		for _, p := range run.JobParameters {
			jobParameters[p.Name] = p.Value
		}
	}
	return &JobRunRemote{
		RunNow: jobs.RunNow{
			JobId:             run.JobId,
			JobParameters:     jobParameters,
			DbtCommands:       overriding.DbtCommands,
			JarParams:         overriding.JarParams,
			NotebookParams:    overriding.NotebookParams,
			PipelineParams:    overriding.PipelineParams,
			PythonNamedParams: overriding.PythonNamedParams,
			PythonParams:      overriding.PythonParams,
			SparkSubmitParams: overriding.SparkSubmitParams,
			SqlParams:         overriding.SqlParams,
			// Request-only fields GetRun never reports; listed so exhaustruct
			// flags any new SDK field.
			IdempotencyToken:  "",
			Only:              nil,
			PerformanceTarget: "",
			Queue:             nil,
			ForceSendFields:   nil,
		},
		RunId:      run.RunId,
		RunName:    run.RunName,
		State:      run.State,
		RunPageUrl: run.RunPageUrl,
		RunType:    run.RunType,
	}
}

// DoRead returns the run as GetRun reports it; a 404 lets the planner
// re-trigger. Root ignore_remote_changes suppresses all remote drift, so a run
// is recreated only on a local config change.
func (r *ResourceJobRun) DoRead(ctx context.Context, id string) (*JobRunRemote, error) {
	runID, err := parseRunID(id)
	if err != nil {
		return nil, err
	}
	// var + field set (not a literal) avoids listing GetRunRequest's other fields.
	var req jobs.GetRunRequest
	req.RunId = runID
	run, err := r.client.Jobs.GetRun(ctx, req)
	if err != nil {
		return nil, err
	}
	return makeJobRunRemote(run), nil
}

// RemapState extracts the embedded RunNow as the state used for diffing.
func (*ResourceJobRun) RemapState(remote *JobRunRemote) *JobRunState {
	return &JobRunState{RunNow: remote.RunNow}
}

func (r *ResourceJobRun) DoCreate(ctx context.Context, _ *StateSaver, config *JobRunState) (string, *JobRunRemote, error) {
	// RunNow returns only the new run id, so we return a nil remote and let the
	// framework read it back via DoRead.
	wait, err := r.client.Jobs.RunNow(ctx, config.RunNow)
	if err != nil {
		return "", nil, err
	}
	return strconv.FormatInt(wait.RunId, 10), nil, nil
}

// DoUpdate is intentionally not implemented: a run can't be modified in place,
// so any change recreates it (delete + a fresh RunNow).

// DoDelete deletes the run via jobs/runs/delete, on both destroy and the
// recreate path. The API rejects a still-active run; this milestone doesn't
// await completion, so that error surfaces to the user.
func (r *ResourceJobRun) DoDelete(ctx context.Context, id string, _ *JobRunState) error {
	runID, err := parseRunID(id)
	if err != nil {
		return err
	}
	return r.client.Jobs.DeleteRunByRunId(ctx, runID)
}

func parseRunID(id string) (int64, error) {
	result, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("internal error: run id is not integer: %q: %w", id, err)
	}
	return result, nil
}

package testserver

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"

	"github.com/databricks/databricks-sdk-go/service/jobs"
)

func (s *FakeWorkspace) JobsCreate(req Request) Response {
	var request jobs.CreateJob
	if err := json.Unmarshal(req.Body, &request); err != nil {
		return Response{
			StatusCode: 400,
			Body:       fmt.Sprintf("request parsing error: %s", err),
		}
	}

	defer s.LockUnlock()()

	jobId := nextID()

	jobSettings := jobs.JobSettings{}
	if err := jsonConvert(request, &jobSettings); err != nil {
		return Response{
			StatusCode: 400,
			Body:       fmt.Sprintf("Cannot convert request to jobSettings: %s", err),
		}
	}

	// CreatorUserName field is used by TF to check if the resource exists or not. CreatorUserName should be non-empty for the resource to be considered as "exists"
	// https://github.com/databricks/terraform-provider-databricks/blob/main/permissions/permission_definitions.go#L108
	s.Jobs[jobId] = jobs.Job{JobId: jobId, Settings: &jobSettings, CreatorUserName: TestUser.UserName}
	return Response{Body: jobs.CreateResponse{JobId: jobId}}
}

func (s *FakeWorkspace) JobsReset(req Request) Response {
	var request jobs.ResetJob
	if err := json.Unmarshal(req.Body, &request); err != nil {
		return Response{
			StatusCode: 400,
			Body:       fmt.Sprintf("request parsing error: %s", err),
		}
	}

	defer s.LockUnlock()()

	jobId := request.JobId
	if _, ok := s.Jobs[jobId]; !ok {
		return Response{StatusCode: 403, Body: "{}"}
	}

	s.Jobs[jobId] = jobs.Job{JobId: jobId, Settings: &request.NewSettings}
	return Response{Body: ""}
}

func (s *FakeWorkspace) JobsGet(req Request) Response {
	id := req.URL.Query().Get("job_id")
	jobIdInt, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return Response{
			StatusCode: 400,
			Body:       fmt.Sprintf("Failed to parse job id: %s: %v", err, id),
		}
	}

	defer s.LockUnlock()()

	job, ok := s.Jobs[jobIdInt]
	if !ok {
		return Response{StatusCode: 404}
	}

	job = setSourceIfNotSet(job)
	return Response{Body: job}
}

func (s *FakeWorkspace) JobsList() Response {
	defer s.LockUnlock()()

	list := make([]jobs.BaseJob, 0, len(s.Jobs))
	for _, job := range s.Jobs {
		job = setSourceIfNotSet(job)
		baseJob := jobs.BaseJob{}
		if err := jsonConvert(job, &baseJob); err != nil {
			return Response{
				StatusCode: 400,
				Body:       fmt.Sprintf("failed to convert job to base job: %s", err),
			}
		}
		list = append(list, baseJob)
	}

	sort.Slice(list, func(i, j int) bool { return list[i].JobId < list[j].JobId })
	return Response{Body: jobs.ListJobsResponse{Jobs: list}}
}

func (s *FakeWorkspace) JobsRunNow(req Request) Response {
	var request jobs.RunNow
	if err := json.Unmarshal(req.Body, &request); err != nil {
		return Response{
			StatusCode: 400,
			Body:       fmt.Sprintf("request parsing error: %s", err),
		}
	}

	defer s.LockUnlock()()

	if _, ok := s.Jobs[request.JobId]; !ok {
		return Response{StatusCode: 404}
	}

	runId := nextID()
	s.JobRuns[runId] = jobs.Run{
		RunId:      runId,
		State:      &jobs.RunState{LifeCycleState: jobs.RunLifeCycleStateRunning},
		RunPageUrl: fmt.Sprintf("%s/job/run/%d", s.url, runId),
		RunType:    jobs.RunTypeJobRun,
		RunName:    "run-name",
	}

	return Response{Body: jobs.RunNowResponse{RunId: runId}}
}

func (s *FakeWorkspace) JobsGetRun(req Request) Response {
	runId := req.URL.Query().Get("run_id")
	runIdInt, err := strconv.ParseInt(runId, 10, 64)
	if err != nil {
		return Response{
			StatusCode: 400,
			Body:       fmt.Sprintf("Failed to parse job id: %s: %v", err, runId),
		}
	}

	defer s.LockUnlock()()

	run, ok := s.JobRuns[runIdInt]
	if !ok {
		return Response{StatusCode: 404}
	}

	run.State.LifeCycleState = jobs.RunLifeCycleStateTerminated
	return Response{Body: run}
}

func setSourceIfNotSet(job jobs.Job) jobs.Job {
	if job.Settings != nil {
		source := "WORKSPACE"
		if job.Settings.GitSource != nil {
			source = "GIT"
		}
		for _, task := range job.Settings.Tasks {
			if task.NotebookTask != nil {
				if task.NotebookTask.Source == "" {
					task.NotebookTask.Source = jobs.Source(source)
				}
				if task.DbtTask != nil {
					if task.DbtTask.Source == "" {
						task.DbtTask.Source = jobs.Source(source)
					}
				}
				if task.SparkPythonTask != nil {
					if task.SparkPythonTask.Source == "" {
						task.SparkPythonTask.Source = jobs.Source(source)
					}
				}
			}
		}
	}
	return job
}

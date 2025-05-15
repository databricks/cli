package testserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/google/uuid"
)

// FakeWorkspace holds a state of a workspace for acceptance tests.
type FakeWorkspace struct {
	mu  sync.Mutex
	url string

	directories map[string]bool
	files       map[string][]byte
	// normally, ids are not sequential, but we make them sequential for deterministic diff
	nextJobId    int64
	nextJobRunId int64
	Jobs         map[int64]jobs.Job
	jobRuns      map[int64]jobs.Run

	Pipelines map[string]pipelines.PipelineSpec
	Monitors  map[string]catalog.MonitorInfo
	Apps      map[string]apps.App
	Schemas   map[string]catalog.SchemaInfo
}

func (w *FakeWorkspace) LockUnlock() func() {
	if w == nil {
		panic("LockUnlock called on nil FakeWorkspace")
	}
	w.mu.Lock()
	return func() { w.mu.Unlock() }
}

// Generic functions to handle map operations
func MapGet[T any](w *FakeWorkspace, collection map[string]T, key string) Response {
	defer w.LockUnlock()()

	value, ok := collection[key]
	if !ok {
		return Response{
			StatusCode: 404,
		}
	}
	return Response{
		Body: value,
	}
}

func MapDelete[K comparable, V any](w *FakeWorkspace, collection map[K]V, key K) Response {
	defer w.LockUnlock()()

	_, ok := collection[key]
	if !ok {
		return Response{
			StatusCode: 404,
		}
	}
	delete(collection, key)
	return Response{}
}

func NewFakeWorkspace(url string) *FakeWorkspace {
	return &FakeWorkspace{
		url: url,
		directories: map[string]bool{
			"/Workspace": true,
		},
		files:        map[string][]byte{},
		Jobs:         map[int64]jobs.Job{},
		jobRuns:      map[int64]jobs.Run{},
		nextJobId:    1,
		nextJobRunId: 1,
		Pipelines:    map[string]pipelines.PipelineSpec{},
		Monitors:     map[string]catalog.MonitorInfo{},
		Apps:         map[string]apps.App{},
		Schemas:      map[string]catalog.SchemaInfo{},
	}
}

func (s *FakeWorkspace) WorkspaceGetStatus(path string) Response {
	defer s.LockUnlock()()

	if s.directories[path] {
		return Response{
			Body: &workspace.ObjectInfo{
				ObjectType: "DIRECTORY",
				Path:       path,
			},
		}
	} else if _, ok := s.files[path]; ok {
		return Response{
			Body: &workspace.ObjectInfo{
				ObjectType: "FILE",
				Path:       path,
				Language:   "SCALA",
			},
		}
	} else {
		return Response{
			StatusCode: 404,
			Body:       map[string]string{"message": "Workspace path not found"},
		}
	}
}

func (s *FakeWorkspace) WorkspaceMkdirs(request workspace.Mkdirs) {
	defer s.LockUnlock()()
	s.directories[request.Path] = true
}

func (s *FakeWorkspace) WorkspaceExport(path string) []byte {
	defer s.LockUnlock()()
	return s.files[path]
}

func (s *FakeWorkspace) WorkspaceDelete(path string, recursive bool) {
	defer s.LockUnlock()()
	if !recursive {
		s.files[path] = nil
	} else {
		for key := range s.files {
			if strings.HasPrefix(key, path) {
				s.files[key] = nil
			}
		}
	}
}

func (s *FakeWorkspace) WorkspaceFilesImportFile(filePath string, body []byte) {
	if !strings.HasPrefix(filePath, "/") {
		filePath = "/" + filePath
	}

	defer s.LockUnlock()()

	s.files[filePath] = body

	// Add all directories in the path to the directories map
	for dir := path.Dir(filePath); dir != "/"; dir = path.Dir(dir) {
		s.directories[dir] = true
	}
}

func (s *FakeWorkspace) WorkspaceFilesExportFile(path string) []byte {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	defer s.LockUnlock()()

	return s.files[path]
}

func (s *FakeWorkspace) JobsCreate(request jobs.CreateJob) Response {
	defer s.LockUnlock()()

	jobId := s.nextJobId
	s.nextJobId++

	jobSettings := jobs.JobSettings{}
	err := jsonConvert(request, &jobSettings)
	if err != nil {
		return Response{
			StatusCode: 400,
			Body:       fmt.Sprintf("Cannot convert request to jobSettings: %s", err),
		}
	}

	s.Jobs[jobId] = jobs.Job{
		JobId:    jobId,
		Settings: &jobSettings,
	}

	return Response{
		Body: jobs.CreateResponse{JobId: jobId},
	}
}

func (s *FakeWorkspace) JobsReset(request jobs.ResetJob) Response {
	defer s.LockUnlock()()

	jobId := request.JobId

	_, ok := s.Jobs[request.JobId]
	if !ok {
		return Response{
			StatusCode: 403,
			Body:       "{}",
		}
	}

	s.Jobs[jobId] = jobs.Job{
		JobId:    jobId,
		Settings: &request.NewSettings,
	}

	return Response{
		Body: "",
	}
}

func (s *FakeWorkspace) PipelinesCreate(r pipelines.PipelineSpec) Response {
	defer s.LockUnlock()()

	pipelineId := uuid.New().String()

	r.Id = pipelineId
	s.Pipelines[pipelineId] = r

	return Response{
		Body: pipelines.CreatePipelineResponse{
			PipelineId: pipelineId,
		},
	}
}

func (s *FakeWorkspace) JobsGet(jobId string) Response {
	id := jobId

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
		return Response{
			StatusCode: 404,
		}
	}

	return Response{
		Body: job,
	}
}

func (s *FakeWorkspace) JobsRunNow(jobId int64) Response {
	defer s.LockUnlock()()

	_, ok := s.Jobs[jobId]
	if !ok {
		return Response{
			StatusCode: 404,
		}
	}

	runId := s.nextJobRunId
	s.nextJobRunId++
	s.jobRuns[runId] = jobs.Run{
		RunId: runId,
		State: &jobs.RunState{
			LifeCycleState: jobs.RunLifeCycleStateRunning,
		},
		RunPageUrl: fmt.Sprintf("%s/job/run/%d", s.url, runId),
		RunType:    jobs.RunTypeJobRun,
		RunName:    "run-name",
	}

	return Response{
		Body: jobs.RunNowResponse{
			RunId: runId,
		},
	}
}

func (s *FakeWorkspace) JobsGetRun(runId int64) Response {
	defer s.LockUnlock()()

	run, ok := s.jobRuns[runId]
	if !ok {
		return Response{
			StatusCode: 404,
		}
	}

	// Mark the run as terminated.
	run.State.LifeCycleState = jobs.RunLifeCycleStateTerminated
	return Response{
		Body: run,
	}
}

func (s *FakeWorkspace) PipelinesGet(pipelineId string) Response {
	defer s.LockUnlock()()

	spec, ok := s.Pipelines[pipelineId]
	if !ok {
		return Response{
			StatusCode: 404,
		}
	}

	return Response{
		Body: pipelines.GetPipelineResponse{
			PipelineId: pipelineId,
			Spec:       &spec,
		},
	}
}

func (s *FakeWorkspace) JobsList() Response {
	defer s.LockUnlock()()

	list := make([]jobs.BaseJob, 0, len(s.Jobs))
	for _, job := range s.Jobs {
		baseJob := jobs.BaseJob{}
		err := jsonConvert(job, &baseJob)
		if err != nil {
			return Response{
				StatusCode: 400,
				Body:       fmt.Sprintf("failed to convert job to base job: %s", err),
			}
		}

		list = append(list, baseJob)
	}

	// sort to have less non-determinism in tests
	sort.Slice(list, func(i, j int) bool {
		return list[i].JobId < list[j].JobId
	})

	return Response{
		Body: jobs.ListJobsResponse{
			Jobs: list,
		},
	}
}

// jsonConvert saves input to a value pointed by output
func jsonConvert(input, output any) error {
	writer := new(bytes.Buffer)
	encoder := json.NewEncoder(writer)
	err := encoder.Encode(input)
	if err != nil {
		return fmt.Errorf("failed to encode: %w", err)
	}

	decoder := json.NewDecoder(writer)
	err = decoder.Decode(output)
	if err != nil {
		return fmt.Errorf("failed to decode: %w", err)
	}

	return nil
}

package testserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/google/uuid"
)

// FakeWorkspace holds a state of a workspace for acceptance tests.
type FakeWorkspace struct {
	directories map[string]bool
	files       map[string][]byte
	// normally, ids are not sequential, but we make them sequential for deterministic diff
	nextJobId int64
	jobs      map[int64]jobs.Job

	Pipelines map[string]pipelines.PipelineSpec
	Monitors  map[string]catalog.MonitorInfo
	Apps      map[string]apps.App
	Schemas   map[string]catalog.SchemaInfo
}

// Generic functions to handle map operations
func MapGet[T any](collection map[string]T, key string) Response {
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

func MapDelete[T any](collection map[string]T, key string) Response {
	_, ok := collection[key]
	if !ok {
		return Response{
			StatusCode: 404,
		}
	}
	delete(collection, key)
	return Response{}
}

func NewFakeWorkspace() *FakeWorkspace {
	return &FakeWorkspace{
		directories: map[string]bool{
			"/Workspace": true,
		},
		files:     map[string][]byte{},
		jobs:      map[int64]jobs.Job{},
		nextJobId: 1,

		Pipelines: map[string]pipelines.PipelineSpec{},
		Monitors:  map[string]catalog.MonitorInfo{},
		Apps:      map[string]apps.App{},
		Schemas:   map[string]catalog.SchemaInfo{},
	}
}

func (s *FakeWorkspace) WorkspaceGetStatus(path string) Response {
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
	s.directories[request.Path] = true
}

func (s *FakeWorkspace) WorkspaceExport(path string) []byte {
	return s.files[path]
}

func (s *FakeWorkspace) WorkspaceDelete(path string, recursive bool) {
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
	return s.files[path]
}

func (s *FakeWorkspace) JobsCreate(request jobs.CreateJob) Response {
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

	s.jobs[jobId] = jobs.Job{
		JobId:    jobId,
		Settings: &jobSettings,
	}

	return Response{
		Body: jobs.CreateResponse{JobId: jobId},
	}
}

func (s *FakeWorkspace) PipelinesCreate(r pipelines.PipelineSpec) Response {
	pipelineId := uuid.New().String()

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

	job, ok := s.jobs[jobIdInt]
	if !ok {
		return Response{
			StatusCode: 404,
		}
	}

	return Response{
		Body: job,
	}
}

func (s *FakeWorkspace) PipelinesGet(pipelineId string) Response {
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
	list := make([]jobs.BaseJob, 0, len(s.jobs))
	for _, job := range s.jobs {
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

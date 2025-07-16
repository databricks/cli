package testserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/dashboards"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/databricks/databricks-sdk-go/service/workspace"
)

// 4611686018427387911 == 2 ** 62 + 7
// 2305843009213693969 == 2 ** 61 + 17
// This values cannot be represented by float64, so they can test incorrect use of json parsing
// (encoding/json without options parses numbers into float64)
// These are also easier to spot / replace in test output compared to numbers with one or few digits.
const (
	TestJobID = 4611686018427387911
	TestRunID = 2305843009213693969
)

type FileEntry struct {
	Info workspace.ObjectInfo
	Data []byte
}

// FakeWorkspace holds a state of a workspace for acceptance tests.
type FakeWorkspace struct {
	mu  sync.Mutex
	url string

	directories  map[string]bool
	files        map[string]FileEntry
	repoIdByPath map[string]int64

	// normally, ids are not sequential, but we make them sequential for deterministic diff
	nextJobId    int64
	nextJobRunId int64
	Jobs         map[int64]jobs.Job
	JobRuns      map[int64]jobs.Run

	Pipelines       map[string]pipelines.GetPipelineResponse
	PipelineUpdates map[string]bool
	Monitors        map[string]catalog.MonitorInfo
	Apps            map[string]apps.App
	Schemas         map[string]catalog.SchemaInfo
	Volumes         map[string]catalog.VolumeInfo
	Dashboards      map[string]dashboards.Dashboard
	SqlWarehouses   map[string]sql.GetWarehouseResponse

	nextRepoId int64
	Repos      map[string]workspace.RepoInfo
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
			Body:       map[string]string{"message": fmt.Sprintf("Resource %T not found: %v", value, key)},
		}
	}
	return Response{
		Body: value,
	}
}

func MapList[K comparable, T any](w *FakeWorkspace, collection map[K]T, responseFieldName string) Response {
	defer w.LockUnlock()()

	items := make([]T, 0, len(collection))

	for _, value := range collection {
		items = append(items, value)
	}

	// Create a map with the provided field name containing the items
	wrapper := map[string]any{
		responseFieldName: items,
	}

	return Response{
		Body: wrapper,
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
		files:        make(map[string]FileEntry),
		repoIdByPath: make(map[string]int64),

		Jobs:            map[int64]jobs.Job{},
		JobRuns:         map[int64]jobs.Run{},
		nextJobId:       TestJobID,
		nextJobRunId:    TestRunID,
		Pipelines:       map[string]pipelines.GetPipelineResponse{},
		PipelineUpdates: map[string]bool{},
		Monitors:        map[string]catalog.MonitorInfo{},
		Apps:            map[string]apps.App{},
		Schemas:         map[string]catalog.SchemaInfo{},
		Volumes:         map[string]catalog.VolumeInfo{},
		Dashboards:      map[string]dashboards.Dashboard{},
		SqlWarehouses:   map[string]sql.GetWarehouseResponse{},
		Repos:           map[string]workspace.RepoInfo{},
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
	} else if entry, ok := s.files[path]; ok {
		return Response{
			Body: entry.Info,
		}
	} else if repoId, ok := s.repoIdByPath[path]; ok {
		return Response{
			Body: workspace.ObjectInfo{
				ObjectType: "REPO",
				Path:       path,
				ObjectId:   repoId,
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
	return s.files[path].Data
}

func (s *FakeWorkspace) WorkspaceDelete(path string, recursive bool) {
	defer s.LockUnlock()()
	if !recursive {
		delete(s.files, path)
	} else {
		for key := range s.files {
			if strings.HasPrefix(key, path) {
				delete(s.files, key)
			}
		}
	}
}

func (s *FakeWorkspace) WorkspaceFilesImportFile(filePath string, body []byte, overwrite bool) Response {
	if !strings.HasPrefix(filePath, "/") {
		filePath = "/" + filePath
	}

	defer s.LockUnlock()()

	workspacePath := filePath

	if !overwrite {
		if _, exists := s.files[workspacePath]; exists {
			return Response{
				StatusCode: 409,
				Body:       map[string]string{"message": fmt.Sprintf("File already exists at (%s).", workspacePath)},
			}
		}
	}

	// Note: Files with .py, .scala, .r or .sql extension can
	// be notebooks if they contain a magical "Databricks notebook source"
	// header comment. We omit support non-python extensions for now for simplicity.
	extension := filepath.Ext(filePath)
	if extension == ".py" && strings.HasPrefix(string(body), "# Databricks notebook source") {
		// Notebooks are stripped of their extension by the workspace import API.
		workspacePath = strings.TrimSuffix(filePath, extension)
		s.files[workspacePath] = FileEntry{
			Info: workspace.ObjectInfo{
				ObjectType: "NOTEBOOK",
				Path:       workspacePath,
				Language:   "PYTHON",
			},
			Data: body,
		}
	} else {
		// The endpoint does not set language for files, so we omit that
		// here as well.
		// ref: https://docs.databricks.com/api/workspace/workspace/getstatus#language
		s.files[workspacePath] = FileEntry{
			Info: workspace.ObjectInfo{
				ObjectType: "FILE",
				Path:       workspacePath,
			},
			Data: body,
		}
	}

	// Add all directories in the path to the directories map
	for dir := path.Dir(workspacePath); dir != "/"; dir = path.Dir(dir) {
		s.directories[dir] = true
	}

	return Response{}
}

func (s *FakeWorkspace) WorkspaceFilesExportFile(path string) []byte {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	defer s.LockUnlock()()

	return s.files[path].Data
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
	s.JobRuns[runId] = jobs.Run{
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

	run, ok := s.JobRuns[runId]
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

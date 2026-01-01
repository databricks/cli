package testserver

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/dashboards"
	"github.com/databricks/databricks-sdk-go/service/database"
	"github.com/google/uuid"

	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/ml"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/databricks/databricks-sdk-go/service/serving"
	"github.com/databricks/databricks-sdk-go/service/sql"
	"github.com/databricks/databricks-sdk-go/service/workspace"
)

const (
	UserNameTokenPrefix         = "dbapi0"
	ServicePrincipalTokenPrefix = "dbapi1"
	UserID                      = "1000012345"
)

var TestUser = iam.User{
	Id:       UserID,
	UserName: "tester@databricks.com",
}

var TestUserSP = iam.User{
	Id:       UserID,
	UserName: "aaaaaaaa-bbbb-4ccc-dddd-eeeeeeeeeeee",
}

var (
	idMutex      sync.Mutex
	lastNowNano  int64
	lastNowMilli int64
)

// IDs are prefixed with 7 and padded to avoid matching regex 1[78]\d{14}
func nextID() int64 {
	// offset enough so that it does not match UNIX_TIME_NANO regex
	return nowNano() + 7000000000000000000
}

// nextID returns nanosecond timestamp but offset but strictly incremental
// (saves last value, protects with mutex and ensures next value is at least last+1)
func nowNano() int64 {
	idMutex.Lock()
	defer idMutex.Unlock()

	newTime := time.Now().UnixNano()
	if newTime <= lastNowNano {
		lastNowNano++
	} else {
		lastNowNano = newTime
	}

	return lastNowNano
}

func nowMilli() int64 {
	idMutex.Lock()
	defer idMutex.Unlock()

	newTime := time.Now().UnixMilli()
	if newTime <= lastNowMilli {
		lastNowMilli++
	} else {
		lastNowMilli = newTime
	}

	return lastNowMilli
}

func nextUUID() string {
	var b [16]byte
	binary.BigEndian.PutUint64(b[0:8], uint64(nextID()))
	binary.BigEndian.PutUint64(b[8:16], uint64(nextID()))
	u := uuid.Must(uuid.FromBytes(b[:]))
	return u.String()
}

type FileEntry struct {
	Info workspace.ObjectInfo
	Data []byte
}

type fakeDashboard struct {
	dashboards.Dashboard

	// Input value of the serialized dashboard provided. This is used
	// to detect if the etag needs to be updated.
	InputSerializedDashboard string `json:"-"`
}

// FakeWorkspace holds a state of a workspace for acceptance tests.
type FakeWorkspace struct {
	mu                 sync.Mutex
	url                string
	isServicePrincipal bool

	directories  map[string]workspace.ObjectInfo
	files        map[string]FileEntry
	repoIdByPath map[string]int64

	Jobs                map[int64]jobs.Job
	JobRuns             map[int64]jobs.Run
	Pipelines           map[string]pipelines.GetPipelineResponse
	PipelineUpdates     map[string]bool
	Monitors            map[string]catalog.MonitorInfo
	Apps                map[string]apps.App
	Schemas             map[string]catalog.SchemaInfo
	Grants              map[string][]catalog.PrivilegeAssignment
	Volumes             map[string]catalog.VolumeInfo
	Dashboards          map[string]fakeDashboard
	PublishedDashboards map[string]dashboards.PublishedDashboard
	GenieSpaces         map[string]fakeGenieSpace
	SqlWarehouses       map[string]sql.GetWarehouseResponse
	Alerts              map[string]sql.AlertV2
	Experiments         map[string]ml.GetExperimentResponse
	ModelRegistryModels map[string]ml.Model
	Clusters            map[string]compute.ClusterDetails
	Catalogs            map[string]catalog.CatalogInfo
	RegisteredModels    map[string]catalog.RegisteredModelInfo
	ServingEndpoints    map[string]serving.ServingEndpointDetailed

	SecretScopes map[string]workspace.SecretScope
	Secrets      map[string]map[string]string // scope -> key -> value
	Acls         map[string][]workspace.AclItem

	// Generic permissions storage: key is "{object_type}:{object_id}"
	Permissions map[string]iam.ObjectPermissions

	Repos map[string]workspace.RepoInfo

	DatabaseInstances    map[string]database.DatabaseInstance
	DatabaseCatalogs     map[string]database.DatabaseCatalog
	SyncedDatabaseTables map[string]database.SyncedDatabaseTable
}

func (s *FakeWorkspace) LockUnlock() func() {
	if s == nil {
		panic("LockUnlock called on nil FakeWorkspace")
	}
	s.mu.Lock()
	return func() { s.mu.Unlock() }
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

func NewFakeWorkspace(url, token string) *FakeWorkspace {
	return &FakeWorkspace{
		url:                url,
		isServicePrincipal: strings.HasPrefix(token, ServicePrincipalTokenPrefix),
		directories: map[string]workspace.ObjectInfo{
			"/Workspace": {
				ObjectType: "DIRECTORY",
				Path:       "/Workspace",
				ObjectId:   nextID(),
			},
		},
		files:        make(map[string]FileEntry),
		repoIdByPath: make(map[string]int64),

		Jobs:                 map[int64]jobs.Job{},
		JobRuns:              map[int64]jobs.Run{},
		Grants:               map[string][]catalog.PrivilegeAssignment{},
		Pipelines:            map[string]pipelines.GetPipelineResponse{},
		PipelineUpdates:      map[string]bool{},
		Monitors:             map[string]catalog.MonitorInfo{},
		Apps:                 map[string]apps.App{},
		Catalogs:             map[string]catalog.CatalogInfo{},
		Schemas:              map[string]catalog.SchemaInfo{},
		RegisteredModels:     map[string]catalog.RegisteredModelInfo{},
		Volumes:              map[string]catalog.VolumeInfo{},
		Dashboards:           map[string]fakeDashboard{},
		PublishedDashboards:  map[string]dashboards.PublishedDashboard{},
		GenieSpaces:          map[string]fakeGenieSpace{},
		SqlWarehouses:        map[string]sql.GetWarehouseResponse{},
		ServingEndpoints:     map[string]serving.ServingEndpointDetailed{},
		Repos:                map[string]workspace.RepoInfo{},
		SecretScopes:         map[string]workspace.SecretScope{},
		Secrets:              map[string]map[string]string{},
		Acls:                 map[string][]workspace.AclItem{},
		Permissions:          map[string]iam.ObjectPermissions{},
		DatabaseInstances:    map[string]database.DatabaseInstance{},
		DatabaseCatalogs:     map[string]database.DatabaseCatalog{},
		SyncedDatabaseTables: map[string]database.SyncedDatabaseTable{},
		Alerts:               map[string]sql.AlertV2{},
		Experiments:          map[string]ml.GetExperimentResponse{},
		ModelRegistryModels:  map[string]ml.Model{},
		Clusters:             map[string]compute.ClusterDetails{},
	}
}

func (s *FakeWorkspace) CurrentUser() iam.User {
	if s != nil && s.isServicePrincipal {
		return TestUserSP
	} else {
		return TestUser
	}
}

func (s *FakeWorkspace) WorkspaceGetStatus(path string) Response {
	defer s.LockUnlock()()

	if dirInfo, ok := s.directories[path]; ok {
		return Response{
			Body: &dirInfo,
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
	s.directories[request.Path] = workspace.ObjectInfo{
		ObjectType: "DIRECTORY",
		Path:       request.Path,
		ObjectId:   nextID(),
	}
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
				ObjectId:   nextID(),
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
				ObjectId:   nextID(),
			},
			Data: body,
		}
	}

	// Add all directories in the path to the directories map
	for dir := path.Dir(workspacePath); dir != "/"; dir = path.Dir(dir) {
		if _, exists := s.directories[dir]; !exists {
			s.directories[dir] = workspace.ObjectInfo{
				ObjectType: "DIRECTORY",
				Path:       dir,
				ObjectId:   nextID(),
			}
		}
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

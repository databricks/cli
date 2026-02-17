package testserver

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/dashboards"
	"github.com/databricks/databricks-sdk-go/service/database"
	"github.com/databricks/databricks-sdk-go/service/postgres"
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
	TestDefaultClusterId        = "0123-456789-cluster0"
	TestDefaultWarehouseId      = "8ec9edc1-db0c-40df-af8d-7580020fe61e"
	TestDefaultInstancePoolId   = "0123-456789-pool0"
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

func nextDashboardID() string {
	var b [16]byte
	binary.BigEndian.PutUint64(b[0:8], uint64(nextID()))
	binary.BigEndian.PutUint64(b[8:16], uint64(nextID()))
	return fmt.Sprintf("%032x", b)
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
	JobRunOutputs       map[int64]jobs.RunOutput
	Pipelines           map[string]pipelines.GetPipelineResponse
	PipelineUpdates     map[string]bool
	Monitors            map[string]catalog.MonitorInfo
	Apps                map[string]apps.App
	Schemas             map[string]catalog.SchemaInfo
	Grants              map[string][]catalog.PrivilegeAssignment
	Volumes             map[string]catalog.VolumeInfo
	Dashboards          map[string]fakeDashboard
	PublishedDashboards map[string]dashboards.PublishedDashboard
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

	Groups map[string]iam.Group

	Repos map[string]workspace.RepoInfo

	DatabaseInstances    map[string]database.DatabaseInstance
	DatabaseCatalogs     map[string]database.DatabaseCatalog
	SyncedDatabaseTables map[string]database.SyncedDatabaseTable

	PostgresProjects   map[string]postgres.Project
	PostgresBranches   map[string]postgres.Branch
	PostgresEndpoints  map[string]postgres.Endpoint
	PostgresOperations map[string]postgres.Operation

	// clusterVenvs caches Python venvs per existing cluster ID,
	// matching cloud behavior where libraries are cached on running clusters.
	clusterVenvs map[string]*clusterEnv
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
			"/Users": {
				ObjectType: "DIRECTORY",
				Path:       "/Users",
				ObjectId:   nextID(),
			},
			"/Users/" + TestUser.UserName: {
				ObjectType: "DIRECTORY",
				Path:       "/Users/" + TestUser.UserName,
				ObjectId:   nextID(),
			},
			"/Users/" + TestUserSP.UserName: {
				ObjectType: "DIRECTORY",
				Path:       "/Users/" + TestUserSP.UserName,
				ObjectId:   nextID(),
			},
		},
		files:        make(map[string]FileEntry),
		repoIdByPath: make(map[string]int64),

		Jobs:                map[int64]jobs.Job{},
		JobRuns:             map[int64]jobs.Run{},
		JobRunOutputs:       map[int64]jobs.RunOutput{},
		Grants:              map[string][]catalog.PrivilegeAssignment{},
		Pipelines:           map[string]pipelines.GetPipelineResponse{},
		PipelineUpdates:     map[string]bool{},
		Monitors:            map[string]catalog.MonitorInfo{},
		Apps:                map[string]apps.App{},
		Catalogs:            map[string]catalog.CatalogInfo{},
		Schemas:             map[string]catalog.SchemaInfo{},
		RegisteredModels:    map[string]catalog.RegisteredModelInfo{},
		Volumes:             map[string]catalog.VolumeInfo{},
		Dashboards:          map[string]fakeDashboard{},
		PublishedDashboards: map[string]dashboards.PublishedDashboard{},
		SqlWarehouses: map[string]sql.GetWarehouseResponse{
			TestDefaultWarehouseId: {
				Id:    TestDefaultWarehouseId,
				Name:  "DEFAULT Test SQL Warehouse",
				State: sql.StateRunning,
			},
		},
		ServingEndpoints:     map[string]serving.ServingEndpointDetailed{},
		Repos:                map[string]workspace.RepoInfo{},
		SecretScopes:         map[string]workspace.SecretScope{},
		Secrets:              map[string]map[string]string{},
		Acls:                 map[string][]workspace.AclItem{},
		Permissions:          map[string]iam.ObjectPermissions{},
		Groups:               map[string]iam.Group{},
		DatabaseInstances:    map[string]database.DatabaseInstance{},
		DatabaseCatalogs:     map[string]database.DatabaseCatalog{},
		SyncedDatabaseTables: map[string]database.SyncedDatabaseTable{},
		PostgresProjects:     map[string]postgres.Project{},
		PostgresBranches:     map[string]postgres.Branch{},
		PostgresEndpoints:    map[string]postgres.Endpoint{},
		PostgresOperations:   map[string]postgres.Operation{},
		clusterVenvs:         map[string]*clusterEnv{},
		Alerts:               map[string]sql.AlertV2{},
		Experiments:          map[string]ml.GetExperimentResponse{},
		ModelRegistryModels:  map[string]ml.Model{},
		Clusters: map[string]compute.ClusterDetails{
			TestDefaultClusterId: {
				ClusterId:   TestDefaultClusterId,
				ClusterName: "DEFAULT Test Cluster",
			},
		},
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

	// Normalize path for lookup: remove leading // and /Workspace prefix
	originalPath := path
	if strings.HasPrefix(path, "//") {
		path = path[1:]
		originalPath = path
	}
	lookupPath, _ := strings.CutPrefix(path, "/Workspace")

	if dirInfo, ok := s.directories[lookupPath]; ok {
		// Return path with /Workspace prefix to match cloud behavior
		info := dirInfo
		info.Path = originalPath
		return Response{
			Body: &info,
		}
	} else if entry, ok := s.files[lookupPath]; ok {
		// Return path with /Workspace prefix to match cloud behavior
		info := entry.Info
		info.Path = originalPath
		return Response{
			Body: &info,
		}
	} else if repoId, ok := s.repoIdByPath[lookupPath]; ok {
		return Response{
			Body: workspace.ObjectInfo{
				ObjectType: "REPO",
				Path:       originalPath,
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
	// Normalize path for storage: strip /Workspace prefix if present
	storagePath, _ := strings.CutPrefix(request.Path, "/Workspace")
	s.directories[storagePath] = workspace.ObjectInfo{
		ObjectType: "DIRECTORY",
		Path:       request.Path, // Store original path
		ObjectId:   nextID(),
	}
}

func (s *FakeWorkspace) WorkspaceExport(path string) Response {
	defer s.LockUnlock()()
	// Normalize path: strip /Workspace prefix if present
	if strings.HasPrefix(path, "/Workspace/") {
		path = path[len("/Workspace"):]
	}
	if entry, ok := s.files[path]; ok {
		return Response{
			Body: entry.Data,
		}
	}
	return Response{
		StatusCode: 404,
		Body:       map[string]string{"message": fmt.Sprintf("File not found: %s", path)},
	}
}

func (s *FakeWorkspace) WorkspaceDelete(path string, recursive bool) {
	defer s.LockUnlock()()
	if !recursive {
		delete(s.files, path)
		delete(s.directories, path)
	} else {
		for key := range s.files {
			if strings.HasPrefix(key, path) {
				delete(s.files, key)
			}
		}
		for key := range s.directories {
			if strings.HasPrefix(key, path) {
				delete(s.directories, key)
			}
		}
	}
}

func (s *FakeWorkspace) WorkspaceFilesImportFile(filePath string, body []byte, overwrite bool) Response {
	if !strings.HasPrefix(filePath, "/") {
		filePath = "/" + filePath
	}

	// Normalize path for storage: strip /Workspace prefix if present
	storagePath, _ := strings.CutPrefix(filePath, "/Workspace")

	defer s.LockUnlock()()

	if !overwrite {
		if _, exists := s.files[storagePath]; exists {
			return Response{
				StatusCode: 409,
				Body:       map[string]string{"message": fmt.Sprintf("File already exists at (%s).", filePath)},
			}
		}
	}

	// Note: Files with .py, .scala, .r or .sql extension can
	// be notebooks if they contain a magical "Databricks notebook source"
	// header comment. We omit support non-python extensions for now for simplicity.
	extension := filepath.Ext(storagePath)
	if extension == ".py" && strings.HasPrefix(string(body), "# Databricks notebook source") {
		// Notebooks are stripped of their extension by the workspace import API.
		storagePathWithoutExt := strings.TrimSuffix(storagePath, extension)
		displayPath := strings.TrimSuffix(filePath, extension)
		s.files[storagePathWithoutExt] = FileEntry{
			Info: workspace.ObjectInfo{
				ObjectType: "NOTEBOOK",
				Path:       displayPath, // Use original path with /Workspace
				Language:   "PYTHON",
				ObjectId:   nextID(),
			},
			Data: body,
		}
		storagePath = storagePathWithoutExt // Update for directory creation below
	} else {
		// The endpoint does not set language for files, so we omit that
		// here as well.
		// ref: https://docs.databricks.com/api/workspace/workspace/getstatus#language
		s.files[storagePath] = FileEntry{
			Info: workspace.ObjectInfo{
				ObjectType: "FILE",
				Path:       filePath, // Use original path with /Workspace
				ObjectId:   nextID(),
			},
			Data: body,
		}
	}

	// Add all directories in the path to the directories map
	for dir := path.Dir(storagePath); dir != "/"; dir = path.Dir(dir) {
		if _, exists := s.directories[dir]; !exists {
			// Calculate display path for directory
			displayDir := dir
			if strings.HasPrefix(filePath, "/Workspace/") {
				displayDir = "/Workspace" + dir
			}
			s.directories[dir] = workspace.ObjectInfo{
				ObjectType: "DIRECTORY",
				Path:       displayDir,
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

	// Normalize path: strip /Workspace prefix if present
	if strings.HasPrefix(path, "/Workspace/") {
		path = path[len("/Workspace"):]
	}

	defer s.LockUnlock()()

	if entry, ok := s.files[path]; ok {
		return entry.Data
	}
	return nil
}

// FileExists checks if a file exists at the given path.
func (s *FakeWorkspace) FileExists(path string) bool {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	defer s.LockUnlock()()

	_, exists := s.files[path]
	return exists
}

// DirectoryExists checks if a directory exists at the given path.
func (s *FakeWorkspace) DirectoryExists(path string) bool {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	defer s.LockUnlock()()

	_, exists := s.directories[path]
	return exists
}

// clusterEnv represents a cached Python venv for an existing cluster.
type clusterEnv struct {
	dir           string          // base temp directory containing the venv
	venvDir       string          // path to .venv inside dir
	installedLibs map[string]bool // workspace paths of already-installed wheels
}

// Cleanup removes all cached cluster venvs.
func (s *FakeWorkspace) Cleanup() {
	for _, env := range s.clusterVenvs {
		os.RemoveAll(env.dir)
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

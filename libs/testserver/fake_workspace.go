package testserver

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
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
	"github.com/databricks/databricks-sdk-go/service/vectorsearch"
	"github.com/databricks/databricks-sdk-go/service/workspace"
)

const (
	UserNameTokenPrefix         = "dbapi0"
	ServicePrincipalTokenPrefix = "dbapi1"
	// GuestServicePrincipalTokenPrefix marks an as-test-sp guest sharing another
	// identity's workspace, kept distinct from a test whose primary identity is
	// itself a service principal.
	GuestServicePrincipalTokenPrefix = "dbapi2"
	UserID                           = "1000012345"
	TestDefaultClusterId             = "0123-456789-cluster0"
	TestDefaultWarehouseId           = "8ec9edc1-db0c-40df-af8d-7580020fe61e"
	TestDefaultInstancePoolId        = "0123-456789-pool0"
)

var TestUser = iam.User{
	Id:       UserID,
	UserName: "tester@databricks.com",
}

var TestUserSP = iam.User{
	Id:       UserID,
	UserName: "aaaaaaaa-bbbb-4ccc-dddd-eeeeeeeeeeee",
}

// guestServicePrincipalDisplayName is reported on /Me for the as-test-sp guest,
// matching the named SP used on cloud.
const guestServicePrincipalDisplayName = "deco-test-spn"

// isGuestToken reports whether a token is an as-test-sp guest. Job permission
// checks apply only to guests; the primary identity is treated as an admin.
func isGuestToken(token string) bool {
	return strings.HasPrefix(token, GuestServicePrincipalTokenPrefix)
}

// userForToken returns the identity behind a token: any service-principal token
// (primary or guest) is the SP, otherwise the user.
func userForToken(token string) iam.User {
	if strings.HasPrefix(token, ServicePrincipalTokenPrefix) || isGuestToken(token) {
		return TestUserSP
	}
	return TestUser
}

// MeUser returns the /Me identity for a token. Only the guest SP carries a
// display name, so single-identity SP tests are unaffected.
func (s *FakeWorkspace) MeUser(token string) iam.User {
	user := userForToken(token)
	if isGuestToken(token) {
		user.DisplayName = guestServicePrincipalDisplayName
	}
	return user
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

	Jobs                  map[int64]jobs.Job
	JobRuns               map[int64]jobs.Run
	JobRunOutputs         map[int64]jobs.RunOutput
	Pipelines             map[string]pipelines.GetPipelineResponse
	PipelineUpdates       map[string]bool
	Monitors              map[string]catalog.MonitorInfo
	Apps                  map[string]apps.App
	Schemas               map[string]catalog.SchemaInfo
	Grants                map[string][]catalog.PrivilegeAssignment
	Volumes               map[string]catalog.VolumeInfo
	Dashboards            map[string]fakeDashboard
	PublishedDashboards   map[string]dashboards.PublishedDashboard
	GenieSpaces           map[string]dashboards.GenieSpace
	SqlWarehouses         map[string]sql.GetWarehouseResponse
	Alerts                map[string]sql.AlertV2
	Experiments           map[string]ml.GetExperimentResponse
	ModelRegistryModels   map[string]ml.Model
	ModelRegistryModelIDs map[string]string // model name -> numeric ID
	Clusters              map[string]compute.ClusterDetails
	Catalogs              map[string]catalog.CatalogInfo
	ExternalLocations     map[string]catalog.ExternalLocationInfo
	RegisteredModels      map[string]catalog.RegisteredModelInfo
	ServingEndpoints      map[string]serving.ServingEndpointDetailed
	VectorSearchEndpoints map[string]vectorsearch.EndpointInfo
	VectorSearchIndexes   map[string]fakeVectorSearchIndex

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

	PostgresProjects     map[string]postgres.Project
	PostgresBranches     map[string]postgres.Branch
	PostgresCatalogs     map[string]postgres.Catalog
	PostgresDatabases    map[string]postgres.Database
	PostgresEndpoints    map[string]postgres.Endpoint
	PostgresRoles        map[string]postgres.Role
	PostgresSyncedTables map[string]postgres.SyncedTable
	PostgresOperations   map[string]postgres.Operation

	// Branches and endpoints that the server provisioned implicitly together
	// with their parent (e.g. the production branch on a new project, or the
	// primary endpoint on a new branch). The real backend rejects independent
	// deletion of these — they go away only when the parent is deleted.
	postgresImplicitBranches  map[string]bool
	postgresImplicitEndpoints map[string]bool

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
			// The user home also exists under the /Workspace alias on real
			// workspaces, so model it here too. Imports require the parent
			// directory to exist (see WorkspaceFilesImportFile).
			"/Workspace/Users": {
				ObjectType: "DIRECTORY",
				Path:       "/Workspace/Users",
				ObjectId:   nextID(),
			},
			"/Workspace/Users/" + TestUser.UserName: {
				ObjectType: "DIRECTORY",
				Path:       "/Workspace/Users/" + TestUser.UserName,
				ObjectId:   nextID(),
			},
			"/Workspace/Users/" + TestUserSP.UserName: {
				ObjectType: "DIRECTORY",
				Path:       "/Workspace/Users/" + TestUserSP.UserName,
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
		ExternalLocations:   map[string]catalog.ExternalLocationInfo{},
		Schemas:             map[string]catalog.SchemaInfo{},
		RegisteredModels:    map[string]catalog.RegisteredModelInfo{},
		Volumes:             map[string]catalog.VolumeInfo{},
		Dashboards:          map[string]fakeDashboard{},
		PublishedDashboards: map[string]dashboards.PublishedDashboard{},
		GenieSpaces:         map[string]dashboards.GenieSpace{},
		SqlWarehouses: map[string]sql.GetWarehouseResponse{
			TestDefaultWarehouseId: {
				Id:    TestDefaultWarehouseId,
				Name:  "DEFAULT Test SQL Warehouse",
				State: sql.StateRunning,
			},
		},
		ServingEndpoints:          map[string]serving.ServingEndpointDetailed{},
		VectorSearchEndpoints:     map[string]vectorsearch.EndpointInfo{},
		VectorSearchIndexes:       map[string]fakeVectorSearchIndex{},
		Repos:                     map[string]workspace.RepoInfo{},
		SecretScopes:              map[string]workspace.SecretScope{},
		Secrets:                   map[string]map[string]string{},
		Acls:                      map[string][]workspace.AclItem{},
		Permissions:               map[string]iam.ObjectPermissions{},
		Groups:                    map[string]iam.Group{},
		DatabaseInstances:         map[string]database.DatabaseInstance{},
		DatabaseCatalogs:          map[string]database.DatabaseCatalog{},
		SyncedDatabaseTables:      map[string]database.SyncedDatabaseTable{},
		PostgresProjects:          map[string]postgres.Project{},
		PostgresBranches:          map[string]postgres.Branch{},
		PostgresCatalogs:          map[string]postgres.Catalog{},
		PostgresDatabases:         map[string]postgres.Database{},
		PostgresEndpoints:         map[string]postgres.Endpoint{},
		PostgresRoles:             map[string]postgres.Role{},
		PostgresSyncedTables:      map[string]postgres.SyncedTable{},
		PostgresOperations:        map[string]postgres.Operation{},
		postgresImplicitBranches:  map[string]bool{},
		postgresImplicitEndpoints: map[string]bool{},
		clusterVenvs:              map[string]*clusterEnv{},
		Alerts:                    map[string]sql.AlertV2{},
		Experiments:               map[string]ml.GetExperimentResponse{},
		ModelRegistryModels:       map[string]ml.Model{},
		ModelRegistryModelIDs:     map[string]string{},
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

func (s *FakeWorkspace) WorkspaceGetStatus(requestPath string) Response {
	defer s.LockUnlock()()

	// The real API collapses duplicate slashes, so look up the cleaned path.
	cleaned := path.Clean(requestPath)

	var info workspace.ObjectInfo
	if dirInfo, ok := s.directories[cleaned]; ok {
		info = dirInfo
	} else if entry, ok := s.files[cleaned]; ok {
		info = entry.Info
	} else if repoId, ok := s.repoIdByPath[cleaned]; ok {
		info = workspace.ObjectInfo{ObjectType: "REPO", Path: cleaned, ObjectId: repoId}
	} else {
		// Match the real Workspace API wording, which echoes the requested path.
		return Response{
			StatusCode: 404,
			Body:       map[string]string{"message": fmt.Sprintf("Path (%s) doesn't exist.", requestPath)},
		}
	}

	// A doubled leading slash ("//Workspace/...", which some tests use to avoid
	// Windows path conversion) is sent to the backend verbatim, and it responds
	// with the "/Workspace" mount stripped from the path. A normal single-slash
	// "/Workspace/..." is preserved instead, so only strip the doubled form.
	if strings.HasPrefix(requestPath, "//Workspace/") {
		info.Path = strings.TrimPrefix(info.Path, "/Workspace")
	}

	return Response{Body: info}
}

func (s *FakeWorkspace) WorkspaceList(listPath string) Response {
	defer s.LockUnlock()()

	var objects []workspace.ObjectInfo

	for filePath, entry := range s.files {
		if path.Dir(filePath) == listPath {
			objects = append(objects, entry.Info)
		}
	}
	for dirPath, dirInfo := range s.directories {
		if dirPath != listPath && path.Dir(dirPath) == listPath {
			objects = append(objects, dirInfo)
		}
	}

	slices.SortFunc(objects, func(a, b workspace.ObjectInfo) int {
		return strings.Compare(a.Path, b.Path)
	})

	return Response{
		Body: workspace.ListResponse{Objects: objects},
	}
}

func (s *FakeWorkspace) WorkspaceMkdirs(request workspace.Mkdirs) {
	defer s.LockUnlock()()
	// The real mkdirs API creates all intermediate directories ("mkdir -p"),
	// so seed every ancestor up to the root.
	for dir := request.Path; dir != "/" && dir != "" && dir != "."; dir = path.Dir(dir) {
		if _, exists := s.directories[dir]; !exists {
			s.directories[dir] = workspace.ObjectInfo{
				ObjectType: "DIRECTORY",
				Path:       dir,
				ObjectId:   nextID(),
			}
		}
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

	defer s.LockUnlock()()

	workspacePath := filePath

	if resp, ok := s.requireParentDirectory(workspacePath); !ok {
		return resp
	}

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

	return Response{}
}

// requireParentDirectory returns a 404 response when objectPath's parent
// directory does not exist. The real import API does not create missing parents;
// callers get "mkdir -p" semantics only by first calling /workspace/mkdirs (see
// WorkspaceFilesClient.Write, which mkdirs and retries on this 404). ok is false
// when the returned response should be sent to the client. The caller must hold
// the lock.
func (s *FakeWorkspace) requireParentDirectory(objectPath string) (Response, bool) {
	parent := path.Dir(objectPath)
	if parent == "/" {
		return Response{}, true
	}
	if _, exists := s.directories[parent]; !exists {
		return Response{
			StatusCode: 404,
			Body:       map[string]string{"message": fmt.Sprintf("The parent folder (%s) does not exist.", parent)},
		}, false
	}
	return Response{}, true
}

// WorkspaceImportNotebook stores a notebook imported with the SOURCE format.
// Unlike AUTO format, SOURCE keeps the path as-is (no extension stripping) and
// the notebook language is provided explicitly rather than sniffed from a
// "# Databricks notebook source" header.
func (s *FakeWorkspace) WorkspaceImportNotebook(filePath string, body []byte, language workspace.Language, overwrite bool) Response {
	if !strings.HasPrefix(filePath, "/") {
		filePath = "/" + filePath
	}

	defer s.LockUnlock()()

	if resp, ok := s.requireParentDirectory(filePath); !ok {
		return resp
	}

	if !overwrite {
		if _, exists := s.files[filePath]; exists {
			return Response{
				StatusCode: 409,
				Body:       map[string]string{"message": fmt.Sprintf("File already exists at (%s).", filePath)},
			}
		}
	}

	s.files[filePath] = FileEntry{
		Info: workspace.ObjectInfo{
			ObjectType: "NOTEBOOK",
			Path:       filePath,
			Language:   language,
			ObjectId:   nextID(),
		},
		Data: body,
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

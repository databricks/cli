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
	"strconv"
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
	"github.com/databricks/databricks-sdk-go/service/files"
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
	// EventualConsistencyTokenPrefix identifies workspaces that simulate eventual
	// consistency: the first GET after a create returns 404 (not yet visible).
	EventualConsistencyTokenPrefix = "dbapi3"
	UserID                         = "1000012345"
	TestDefaultClusterId           = "0123-456789-cluster0"
	TestDefaultWarehouseId         = "8ec9edc1-db0c-40df-af8d-7580020fe61e"
	TestDefaultInstancePoolId      = "0123-456789-pool0"
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

// nextTimestamp returns a strictly-increasing RFC3339 timestamp with nanosecond
// precision. The sub-second component keeps distinct events ordered even within the
// same wall-clock second, which the dashboard publish lifecycle relies on to compare
// a draft's update_time against a published revision's revision_create_time.
func nextTimestamp() string {
	return time.Unix(0, nowNano()).UTC().Format(time.RFC3339Nano)
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
	JobRunIdempotency     map[string]int64
	Pipelines             map[string]pipelines.GetPipelineResponse
	PipelineUpdates       map[string]bool
	Monitors              map[string]catalog.MonitorInfo
	Apps                  map[string]apps.App
	Schemas               map[string]catalog.SchemaInfo
	Grants                map[string][]catalog.PrivilegeAssignment
	Volumes               map[string]catalog.VolumeInfo
	Dashboards            *EventualMap[string, *fakeDashboard]
	PublishedDashboards   map[string]dashboards.PublishedDashboard
	GenieSpaces           map[string]dashboards.GenieSpace
	SqlWarehouses         map[string]sql.GetWarehouseResponse
	Alerts                map[string]sql.AlertV2
	Experiments           map[string]ml.GetExperimentResponse
	ModelRegistryModels   map[string]ml.Model
	ModelRegistryModelIDs map[string]string // model name -> numeric ID
	Clusters              map[string]compute.ClusterDetails
	InstancePools         map[string]compute.GetInstancePool
	Catalogs              map[string]catalog.CatalogInfo
	ExternalLocations     map[string]catalog.ExternalLocationInfo
	RegisteredModels      map[string]catalog.RegisteredModelInfo
	ServingEndpoints      map[string]serving.ServingEndpointDetailed
	VectorSearchEndpoints map[string]vectorsearch.EndpointInfo
	VectorSearchIndexes   map[string]fakeVectorSearchIndex

	// VectorSearchIndexesPendingDeletion counts how many further CREATEs an
	// already-deleted index name must reject with "pending deletion". See
	// VectorSearchIndexDelete.
	VectorSearchIndexesPendingDeletion map[string]int

	SecretScopes map[string]workspace.SecretScope
	Secrets      map[string]map[string]string // scope -> key -> value
	Acls         map[string][]workspace.AclItem
	UCSecrets    map[string]catalog.Secret // full_name -> secret (Unity Catalog secrets)

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

	// dmsDeployments holds Deployment Metadata Service (DMS) records, keyed by
	// deployment ID. Each record carries its versions and latest resource state.
	dmsDeployments map[string]*dmsDeployment

	// dmsDeploymentNodes maps deployment ID to the workspace node CreateDeployment made for
	// it. An ID appears here before dmsDeployments has a record, which its first version
	// creates, so the node is what makes the ID valid in between.
	dmsDeploymentNodes map[string]string
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

// MapGetUC is MapGet for Unity Catalog securables. The CLI surfaces the API's
// message verbatim, and UC words it as "Volume 'main.s.v' does not exist."
func MapGetUC[T any](w *FakeWorkspace, collection map[string]T, key, securable string) Response {
	defer w.LockUnlock()()

	value, ok := collection[key]
	if !ok {
		return Response{
			StatusCode: 404,
			Body:       map[string]string{"message": fmt.Sprintf("%s '%s' does not exist.", securable, key)},
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
		JobRunIdempotency:   map[string]int64{},
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
		Dashboards:          NewEventualMap[string, *fakeDashboard](strings.HasPrefix(token, EventualConsistencyTokenPrefix)),
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
		dmsDeployments:            map[string]*dmsDeployment{},
		dmsDeploymentNodes:        map[string]string{},
		Alerts:                    map[string]sql.AlertV2{},
		Experiments:               map[string]ml.GetExperimentResponse{},
		ModelRegistryModels:       map[string]ml.Model{},
		ModelRegistryModelIDs:     map[string]string{},
		Clusters: map[string]compute.ClusterDetails{
			// A running dedicated single-user cluster: the shape `ssh connect --cluster`
			// requires (ValidateClusterAccess rejects anything else), matching the cloud
			// TEST_DEFAULT_CLUSTER_ID this stands in for.
			TestDefaultClusterId: {
				ClusterId:        TestDefaultClusterId,
				ClusterName:      "DEFAULT Test Cluster",
				State:            compute.StateRunning,
				DataSecurityMode: compute.DataSecurityModeSingleUser,
				SingleUserName:   TestUser.UserName,
			},
		},
		InstancePools:                      map[string]compute.GetInstancePool{},
		VectorSearchIndexesPendingDeletion: map[string]int{},
	}
}

func (s *FakeWorkspace) CurrentUser() iam.User {
	if s != nil && s.isServicePrincipal {
		return TestUserSP
	} else {
		return TestUser
	}
}

// gitInfoBlock is the git_info block get-status adds for return_git_info=true.
// workspace.ObjectInfo does not model it (the field is undocumented), so it is
// merged into the response separately, see withGitInfo.
//
// A Git folder that has Git CLI access does not store the git metadata on the
// workspace object, so Branch, HeadCommitID and URL are empty for one; only a
// standard Git folder reports them.
type gitInfoBlock struct {
	Branch       string `json:"branch,omitempty"`
	HeadCommitID string `json:"head_commit_id,omitempty"`
	ID           int64  `json:"id"`
	Path         string `json:"path"`
	URL          string `json:"url,omitempty"`
}

// withGitInfo returns info as an object with a git_info block added.
//
// It cannot be a struct embedding workspace.ObjectInfo: ObjectInfo declares
// MarshalJSON, which gets promoted to the embedding struct, so the outer
// git_info field would be dropped from the output without any error. The
// intermediate map holds json.RawMessage rather than any, so that large ids do
// not lose precision by passing through float64.
func withGitInfo(info workspace.ObjectInfo, gi gitInfoBlock) (any, error) {
	infoJSON, err := json.Marshal(info)
	if err != nil {
		return nil, err
	}
	out := map[string]json.RawMessage{}
	if err := json.Unmarshal(infoJSON, &out); err != nil {
		return nil, err
	}
	giJSON, err := json.Marshal(gi)
	if err != nil {
		return nil, err
	}
	out["git_info"] = giJSON
	return out, nil
}

// gitFolderFor returns the Git folder containing objectPath, which is the
// longest registered Git folder path that is a prefix of it. get-status reports
// the containing folder's metadata for paths inside a Git folder, not just for
// its root.
func (s *FakeWorkspace) gitFolderFor(objectPath string) (workspace.RepoInfo, bool) {
	longest := ""
	for repoPath := range s.repoIdByPath {
		if objectPath != repoPath && !strings.HasPrefix(objectPath, repoPath+"/") {
			continue
		}
		if len(repoPath) > len(longest) {
			longest = repoPath
		}
	}
	if longest == "" {
		return workspace.RepoInfo{}, false
	}
	return s.Repos[strconv.FormatInt(s.repoIdByPath[longest], 10)], true
}

// isGitCliFolder reports whether a Git folder at this path has Git CLI access.
// Those are materialized as plain DIRECTORY nodes outside /Repos, while a
// standard Git folder under /Repos keeps the REPO object type. The /Workspace
// mount prefix is optional on a request, so it is not part of the distinction.
func isGitCliFolder(repoPath string) bool {
	return !strings.HasPrefix(strings.TrimPrefix(repoPath, "/Workspace"), "/Repos/")
}

func (s *FakeWorkspace) WorkspaceGetStatus(requestPath string, returnGitInfo bool) Response {
	defer s.LockUnlock()()

	// The real API collapses duplicate slashes, so look up the cleaned path.
	cleaned := path.Clean(requestPath)

	var info workspace.ObjectInfo
	// A Git folder root is reported as the repo, before any directory entry for
	// the same path: mkdirs of a path inside a Git folder seeds its ancestors,
	// which would otherwise shadow the root and report the wrong object id.
	if repoId, ok := s.repoIdByPath[cleaned]; ok {
		// Control-plane repos (under /Repos) report the REPO object type, while
		// Git-CLI-enabled folders elsewhere are materialized as plain DIRECTORY
		// nodes. Both resolve to a valid repo ID via the repos API.
		objectType := workspace.ObjectTypeRepo
		if isGitCliFolder(cleaned) {
			objectType = workspace.ObjectTypeDirectory
		}
		info = workspace.ObjectInfo{ObjectType: objectType, Path: cleaned, ObjectId: repoId}
	} else if dirInfo, ok := s.directories[cleaned]; ok {
		info = dirInfo
	} else if entry, ok := s.files[cleaned]; ok {
		info = entry.Info
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

	if returnGitInfo {
		if repo, ok := s.gitFolderFor(cleaned); ok {
			// The real API reports the Git folder root without the /Workspace mount
			// prefix, whichever spelling the folder was created with.
			gi := gitInfoBlock{ID: repo.Id, Path: strings.TrimPrefix(repo.Path, "/Workspace")}
			if isGitCliFolder(repo.Path) {
				info.DirectoryInfo = &workspace.DirectoryInfo{IsGitFolder: true}
			} else {
				gi.Branch = repo.Branch
				gi.HeadCommitID = repo.HeadCommitId
				gi.URL = repo.Url
			}
			body, err := withGitInfo(info, gi)
			if err != nil {
				return Response{
					StatusCode: 500,
					Body:       fmt.Sprintf("internal error: %s", err),
				}
			}
			return Response{Body: body}
		}
	}

	return Response{Body: info}
}

func (s *FakeWorkspace) WorkspaceList(listPath string) Response {
	defer s.LockUnlock()()

	// The real API collapses duplicate slashes, so look up the cleaned path.
	cleaned := path.Clean(listPath)

	// The real API 404s on a missing path rather than reporting an empty directory.
	// Repos are listable but tracked outside s.directories, so admit them too.
	_, isDir := s.directories[cleaned]
	_, isRepo := s.repoIdByPath[cleaned]
	if !isDir && !isRepo {
		return Response{
			StatusCode: 404,
			Body:       map[string]string{"message": fmt.Sprintf("Path (%s) doesn't exist.", listPath)},
		}
	}

	var objects []workspace.ObjectInfo

	for filePath, entry := range s.files {
		if path.Dir(filePath) == cleaned {
			objects = append(objects, entry.Info)
		}
	}
	for dirPath, dirInfo := range s.directories {
		if dirPath != cleaned && path.Dir(dirPath) == cleaned {
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

// FsListDirectory implements GET /api/2.0/fs/directories/{path}. A path that is
// not a directory, including one pointing at a file, is a 404, as it is for HEAD.
func (s *FakeWorkspace) FsListDirectory(dirPath string) Response {
	if !strings.HasPrefix(dirPath, "/") {
		dirPath = "/" + dirPath
	}

	defer s.LockUnlock()()

	if _, isDir := s.directories[dirPath]; !isDir {
		return Response{
			StatusCode: 404,
			Body:       map[string]string{"message": "directory does not exist"},
		}
	}

	var contents []files.DirectoryEntry

	for filePath, entry := range s.files {
		if path.Dir(filePath) == dirPath {
			contents = append(contents, files.DirectoryEntry{
				Name:     path.Base(filePath),
				Path:     filePath,
				FileSize: int64(len(entry.Data)),
			})
		}
	}
	for childPath := range s.directories {
		if childPath != dirPath && path.Dir(childPath) == dirPath {
			contents = append(contents, files.DirectoryEntry{
				Name:        path.Base(childPath),
				Path:        childPath,
				IsDirectory: true,
			})
		}
	}

	slices.SortFunc(contents, func(a, b files.DirectoryEntry) int {
		return strings.Compare(a.Path, b.Path)
	})

	return Response{
		Body: files.ListDirectoryResponse{Contents: contents},
	}
}

// FsDeleteFile implements DELETE /api/2.0/fs/files/{path}.
func (s *FakeWorkspace) FsDeleteFile(filePath string) Response {
	if !strings.HasPrefix(filePath, "/") {
		filePath = "/" + filePath
	}

	defer s.LockUnlock()()

	if _, exists := s.files[filePath]; !exists {
		return Response{
			StatusCode: 404,
			Body:       map[string]string{"message": "file does not exist"},
		}
	}

	delete(s.files, filePath)
	return Response{}
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

package testserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/workspace"
)

// FakeWorkspace holds a state of a workspace for acceptance tests.
type FakeWorkspace struct {
	directories map[string]bool
	files       map[string][]byte
	// normally, ids are not sequential, but we make them sequential for deterministic diff
	nextJobId int64
	jobs      map[int64]jobs.Job
}

func NewFakeWorkspace() *FakeWorkspace {
	return &FakeWorkspace{
		directories: map[string]bool{
			"/Workspace": true,
		},
		files:     map[string][]byte{},
		jobs:      map[int64]jobs.Job{},
		nextJobId: 1,
	}
}

func (s *FakeWorkspace) Me() (iam.User, int) {
	return iam.User{
		Id:       "1000012345",
		UserName: "tester@databricks.com",
	}, http.StatusOK
}

func (s *FakeWorkspace) CurrentMetastoreAssignment() (catalog.MetastoreAssignment, int) {
	return catalog.MetastoreAssignment{
		DefaultCatalogName: "main",
	}, http.StatusOK
}

func (s *FakeWorkspace) DirectoryPermissions(objectId string) (workspace.WorkspaceObjectPermissions, int) {
	return workspace.WorkspaceObjectPermissions{
		ObjectId:   objectId,
		ObjectType: "DIRECTORY",
		AccessControlList: []workspace.WorkspaceObjectAccessControlResponse{
			{
				UserName: "tester@databricks.com",
				AllPermissions: []workspace.WorkspaceObjectPermission{
					{
						PermissionLevel: "CAN_MANAGE",
					},
				},
			},
		},
	}, http.StatusOK
}

func (s *FakeWorkspace) GetStatus(path string) (workspace.ObjectInfo, int) {
	if s.directories[path] {
		return workspace.ObjectInfo{
			ObjectType: "DIRECTORY",
			Path:       path,
		}, http.StatusOK
	} else if _, ok := s.files[path]; ok {
		return workspace.ObjectInfo{
			ObjectType: "FILE",
			Path:       path,
			Language:   "SCALA",
		}, http.StatusOK
	} else {
		return workspace.ObjectInfo{}, http.StatusNotFound
	}
}

func (s *FakeWorkspace) WorkspaceMkdirs(request workspace.Mkdirs) (string, int) {
	s.directories[request.Path] = true

	return "{}", http.StatusOK
}

func (s *FakeWorkspace) WorkspaceExport(path string) ([]byte, int) {
	file := s.files[path]

	if file == nil {
		return nil, http.StatusNotFound
	}

	return file, http.StatusOK
}

func (s *FakeWorkspace) WorkspaceDelete(path string, recursive bool) (string, int) {
	if !recursive {
		s.files[path] = nil
	} else {
		for key := range s.files {
			if strings.HasPrefix(key, path) {
				s.files[key] = nil
			}
		}
	}

	return "{}", http.StatusOK
}

func (s *FakeWorkspace) WorkspaceImportFiles(path string, body []byte) (any, int) {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	s.files[path] = body

	return "{}", http.StatusOK
}

func (s *FakeWorkspace) JobsCreate(request jobs.CreateJob) (any, int) {
	jobId := s.nextJobId
	s.nextJobId++

	jobSettings := jobs.JobSettings{}
	err := jsonConvert(request, &jobSettings)
	if err != nil {
		return internalError(err)
	}

	s.jobs[jobId] = jobs.Job{
		JobId:    jobId,
		Settings: &jobSettings,
	}

	return jobs.CreateResponse{JobId: jobId}, http.StatusOK
}

func (s *FakeWorkspace) JobsGet(jobId string) (any, int) {
	id := jobId

	jobIdInt, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return internalError(fmt.Errorf("failed to parse job id: %s", err))
	}

	job, ok := s.jobs[jobIdInt]
	if !ok {
		return jobs.Job{}, http.StatusNotFound
	}

	return job, http.StatusOK
}

func (s *FakeWorkspace) JobsList() (any, int) {
	list := make([]jobs.BaseJob, 0, len(s.jobs))
	for _, job := range s.jobs {
		baseJob := jobs.BaseJob{}
		err := jsonConvert(job, &baseJob)
		if err != nil {
			return internalError(fmt.Errorf("failed to convert job to base job: %w", err))
		}

		list = append(list, baseJob)
	}

	return jobs.ListJobsResponse{
		Jobs: list,
	}, http.StatusOK
}

func (s *FakeWorkspace) InstancePoolsList() (compute.ListInstancePools, int) {
	return compute.ListInstancePools{
		InstancePools: []compute.InstancePoolAndStats{
			{
				InstancePoolName: "some-test-instance-pool",
				InstancePoolId:   "1234",
			},
		},
	}, http.StatusOK
}

func (s *FakeWorkspace) ClustersList() (compute.ListClustersResponse, int) {
	return compute.ListClustersResponse{
		Clusters: []compute.ClusterDetails{
			{
				ClusterName: "some-test-cluster",
				ClusterId:   "4321",
			},
			{
				ClusterName: "some-other-cluster",
				ClusterId:   "9876",
			},
		},
	}, http.StatusOK
}

func (s *FakeWorkspace) PoliciesList() (any, int) {
	return compute.ListPoliciesResponse{
		Policies: []compute.Policy{
			{
				PolicyId: "5678",
				Name:     "wrong-cluster-policy",
			},
			{
				PolicyId: "9876",
				Name:     "some-test-cluster-policy",
			},
		},
	}, http.StatusOK
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

func internalError(err error) (string, int) {
	return fmt.Sprintf("internal error: %s", err), http.StatusInternalServerError
}

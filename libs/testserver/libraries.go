package testserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"

	"github.com/databricks/databricks-sdk-go/service/compute"
)

func init() {
	registerRoutes("libraries", addLibrariesRoutes)
	registerStore("libraries", newLibrariesStore)
}

type librariesStore struct {
	ClusterLibraries map[string][]compute.Library // cluster id -> installed libraries
}

func newLibrariesStore(*FakeWorkspace) *librariesStore {
	return &librariesStore{
		ClusterLibraries: map[string][]compute.Library{},
	}
}

func (s *FakeWorkspace) libraries() *librariesStore {
	return stateOf[librariesStore](s)
}

func addLibrariesRoutes(server *Server) {
	// Cluster libraries:
	server.Handle("POST", "/api/2.0/libraries/install", func(req Request) any {
		return req.Workspace.LibrariesInstall(req)
	})

	server.Handle("POST", "/api/2.0/libraries/uninstall", func(req Request) any {
		return req.Workspace.LibrariesUninstall(req)
	})

	server.Handle("GET", "/api/2.0/libraries/cluster-status", func(req Request) any {
		return req.Workspace.LibrariesClusterStatus(req, req.URL.Query().Get("cluster_id"))
	})
}

func (s *FakeWorkspace) LibrariesInstall(req Request) any {
	var request compute.InstallLibraries
	if err := json.Unmarshal(req.Body, &request); err != nil {
		return Response{StatusCode: http.StatusBadRequest, Body: fmt.Sprintf("request parsing error: %s", err)}
	}

	defer s.LockUnlock()()

	if _, ok := s.clusters().Clusters[request.ClusterId]; !ok {
		return Response{StatusCode: http.StatusNotFound}
	}

	// Install is additive and idempotent: installing an already-present library is a no-op.
	installed := s.libraries().ClusterLibraries[request.ClusterId]
	for _, lib := range request.Libraries {
		if !containsLibrary(installed, lib) {
			installed = append(installed, lib)
		}
	}
	s.libraries().ClusterLibraries[request.ClusterId] = installed

	return Response{}
}

func (s *FakeWorkspace) LibrariesUninstall(req Request) any {
	var request compute.UninstallLibraries
	if err := json.Unmarshal(req.Body, &request); err != nil {
		return Response{StatusCode: http.StatusBadRequest, Body: fmt.Sprintf("request parsing error: %s", err)}
	}

	defer s.LockUnlock()()

	installed := s.libraries().ClusterLibraries[request.ClusterId]
	remaining := make([]compute.Library, 0, len(installed))
	for _, lib := range installed {
		if !containsLibrary(request.Libraries, lib) {
			remaining = append(remaining, lib)
		}
	}
	s.libraries().ClusterLibraries[request.ClusterId] = remaining

	return Response{}
}

func (s *FakeWorkspace) LibrariesClusterStatus(req Request, clusterId string) any {
	defer s.LockUnlock()()

	if _, ok := s.clusters().Clusters[clusterId]; !ok {
		return Response{StatusCode: http.StatusNotFound}
	}

	installed := s.libraries().ClusterLibraries[clusterId]
	statuses := make([]compute.LibraryFullStatus, 0, len(installed))
	for i := range installed {
		statuses = append(statuses, compute.LibraryFullStatus{
			Library: &installed[i],
			Status:  compute.LibraryInstallStatusInstalled,
		})
	}

	return Response{
		Body: compute.ClusterLibraryStatuses{
			ClusterId:       clusterId,
			LibraryStatuses: statuses,
		},
	}
}

func containsLibrary(libs []compute.Library, target compute.Library) bool {
	for _, l := range libs {
		if reflect.DeepEqual(l, target) {
			return true
		}
	}
	return false
}

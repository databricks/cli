package testserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"

	clusterlibraries "github.com/databricks/cli/libs/sdktypes/clusterlibraries/v2"
	"github.com/databricks/cli/libs/utils"
)

func (s *FakeWorkspace) LibrariesInstall(req Request) any {
	var request clusterlibraries.InstallLibrariesRequest
	if err := json.Unmarshal(req.Body, &request); err != nil {
		return Response{StatusCode: http.StatusBadRequest, Body: fmt.Sprintf("request parsing error: %s", err)}
	}

	defer s.LockUnlock()()

	if _, ok := s.Clusters[utils.Deref(request.ClusterId)]; !ok {
		return Response{StatusCode: http.StatusNotFound}
	}

	// Install is additive and idempotent: installing an already-present library is a no-op.
	installed := s.ClusterLibraries[utils.Deref(request.ClusterId)]
	for _, lib := range request.Libraries {
		if !containsLibrary(installed, lib) {
			installed = append(installed, lib)
		}
	}
	s.ClusterLibraries[utils.Deref(request.ClusterId)] = installed

	return Response{}
}

func (s *FakeWorkspace) LibrariesUninstall(req Request) any {
	var request clusterlibraries.UninstallLibrariesRequest
	if err := json.Unmarshal(req.Body, &request); err != nil {
		return Response{StatusCode: http.StatusBadRequest, Body: fmt.Sprintf("request parsing error: %s", err)}
	}

	defer s.LockUnlock()()

	installed := s.ClusterLibraries[utils.Deref(request.ClusterId)]
	remaining := make([]clusterlibraries.Library, 0, len(installed))
	for _, lib := range installed {
		if !containsLibrary(request.Libraries, lib) {
			remaining = append(remaining, lib)
		}
	}
	s.ClusterLibraries[utils.Deref(request.ClusterId)] = remaining

	return Response{}
}

func (s *FakeWorkspace) LibrariesClusterStatus(req Request, clusterId string) any {
	defer s.LockUnlock()()

	if _, ok := s.Clusters[clusterId]; !ok {
		return Response{StatusCode: http.StatusNotFound}
	}

	installed := s.ClusterLibraries[clusterId]
	statuses := make([]clusterlibraries.LibraryFullStatus, 0, len(installed))
	for i := range installed {
		statuses = append(statuses, clusterlibraries.LibraryFullStatus{
			Library: &installed[i],
			Status:  clusterlibraries.LibraryInstallStatus_Installed,
		})
	}

	return Response{
		Body: clusterlibraries.ClusterLibraryStatuses{
			ClusterId:       utils.NonZero(clusterId),
			LibraryStatuses: statuses,
		},
	}
}

func containsLibrary(libs []clusterlibraries.Library, target clusterlibraries.Library) bool {
	for _, l := range libs {
		if reflect.DeepEqual(l, target) {
			return true
		}
	}
	return false
}

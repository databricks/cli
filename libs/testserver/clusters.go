package testserver

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/databricks/databricks-sdk-go/service/compute"
)

func (s *FakeWorkspace) ClustersCreate(req Request) any {
	var request compute.ClusterDetails
	if err := json.Unmarshal(req.Body, &request); err != nil {
		return Response{
			StatusCode: 400,
			Body:       fmt.Sprintf("request parsing error: %s", err),
		}
	}

	defer s.LockUnlock()()

	clusterId := nextUUID()
	request.ClusterId = clusterId

	// Match cloud behavior: SINGLE_USER clusters automatically get single_user_name set
	// to the current user. This enables terraform drift detection when the bundle config
	// doesn't specify single_user_name.
	if request.DataSecurityMode == compute.DataSecurityModeSingleUser && request.SingleUserName == "" {
		request.SingleUserName = s.CurrentUser().UserName
	}

	s.Clusters[clusterId] = request

	return Response{
		Body: compute.ClusterDetails{
			ClusterId: clusterId,
		},
	}
}

func (s *FakeWorkspace) ClustersResize(req Request) any {
	var request compute.ResizeCluster
	if err := json.Unmarshal(req.Body, &request); err != nil {
		return Response{
			StatusCode: 400,
			Body:       fmt.Sprintf("request parsing error: %s", err),
		}
	}

	defer s.LockUnlock()()
	cluster, ok := s.Clusters[request.ClusterId]
	if !ok {
		return Response{StatusCode: 404}
	}

	cluster.NumWorkers = request.NumWorkers
	cluster.Autoscale = request.Autoscale
	s.Clusters[request.ClusterId] = cluster

	return Response{}
}

func (s *FakeWorkspace) ClustersEdit(req Request) any {
	var request compute.ClusterDetails
	if err := json.Unmarshal(req.Body, &request); err != nil {
		return Response{
			StatusCode: 400,
			Body:       fmt.Sprintf("request parsing error: %s", err),
		}
	}

	defer s.LockUnlock()()
	_, ok := s.Clusters[request.ClusterId]
	if !ok {
		return Response{StatusCode: 404}
	}

	s.Clusters[request.ClusterId] = request

	// Clear venv cache when cluster is edited to match cloud behavior where
	// cluster edits trigger restarts that clear library caches.
	if env, ok := s.clusterVenvs[request.ClusterId]; ok {
		os.RemoveAll(env.dir)
		delete(s.clusterVenvs, request.ClusterId)
	}

	return Response{}
}

func (s *FakeWorkspace) ClustersGet(req Request, clusterId string) any {
	defer s.LockUnlock()()

	cluster, ok := s.Clusters[clusterId]
	if !ok {
		return Response{StatusCode: 404}
	}

	return Response{
		Body: cluster,
	}
}

func (s *FakeWorkspace) ClustersStart(req Request) any {
	var request compute.StartCluster
	if err := json.Unmarshal(req.Body, &request); err != nil {
		return Response{
			StatusCode: 400,
			Body:       fmt.Sprintf("request parsing error: %s", err),
		}
	}
	defer s.LockUnlock()()

	cluster, ok := s.Clusters[request.ClusterId]
	if !ok {
		return Response{StatusCode: 404}
	}

	cluster.State = compute.StateRunning
	s.Clusters[request.ClusterId] = cluster

	return Response{}
}

func (s *FakeWorkspace) ClustersPermanentDelete(req Request) any {
	var request compute.PermanentDeleteCluster
	if err := json.Unmarshal(req.Body, &request); err != nil {
		return Response{
			StatusCode: 400,
			Body:       fmt.Sprintf("request parsing error: %s", err),
		}
	}

	defer s.LockUnlock()()
	delete(s.Clusters, request.ClusterId)
	return Response{}
}

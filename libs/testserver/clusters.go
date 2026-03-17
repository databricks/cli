package testserver

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/databricks/cli/libs/structs/structaccess"
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

	clusterFixUps(&request)

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

	clusterFixUps(&request)
	s.Clusters[request.ClusterId] = request

	// Clear venv cache when cluster is edited to match cloud behavior where
	// cluster edits trigger restarts that clear library caches.
	if env, ok := s.clusterVenvs[request.ClusterId]; ok {
		os.RemoveAll(env.dir)
		delete(s.clusterVenvs, request.ClusterId)
	}

	return Response{}
}

func setDefault(obj any, path string, value any) {
	if val, _ := structaccess.GetByString(obj, path); val == nil {
		_ = structaccess.SetByString(obj, path, value)
	}
}

// clusterFixUps applies server-side defaults that the real API sets.
func clusterFixUps(cluster *compute.ClusterDetails) {
	gcp := cluster.GcpAttributes
	if gcp != nil {
		setDefault(gcp, "first_on_demand", 1)
		setDefault(gcp, "use_preemptible_executors", false)
	} else if cluster.AwsAttributes == nil {
		cluster.AwsAttributes = &compute.AwsAttributes{
			Availability: compute.AwsAvailabilitySpotWithFallback,
			ZoneId:       "us-east-1c",
		}
		cluster.AwsAttributes.ForceSendFields = append(
			cluster.AwsAttributes.ForceSendFields,
			"Availability",
			"ZoneId",
		)
	}

	cluster.ForceSendFields = append(cluster.ForceSendFields, "EnableElasticDisk")

	if cluster.DriverNodeTypeId == "" && cluster.NodeTypeId != "" {
		cluster.DriverNodeTypeId = cluster.NodeTypeId
	}
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

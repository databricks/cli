package testserver

import (
	"encoding/json"
	"fmt"
	clusters "github.com/databricks/cli/libs/sdktypes/clusters/v2"
	"os"

	"github.com/databricks/cli/libs/structs/structaccess"
)

func (s *FakeWorkspace) ClustersCreate(req Request) any {
	var request clusters.ClusterInfo
	if err := json.Unmarshal(req.Body, &request); err != nil {
		return Response{
			StatusCode: 400,
			Body:       fmt.Sprintf("request parsing error: %s", err),
		}
	}

	defer s.LockUnlock()()

	clusterId := nextUUID()
	request.ClusterId = nonZero(clusterId)
	// Clusters start in PENDING state when created; ClustersGet transitions them to RUNNING.
	request.State = compute.StatePending

	// Match cloud behavior: SINGLE_USER clusters automatically get single_user_name set
	// to the current user. This enables terraform drift detection when the bundle config
	// doesn't specify single_user_name.
	if request.DataSecurityMode == clusters.DataSecurityMode_SingleUser && val(request.SingleUserName) == "" {
		request.SingleUserName = s.CurrentUser().UserName
	}

	// Apply the attached cluster policy before computing defaults, matching the backend:
	// policy values feed the defaults below (e.g. driver_node_type_id from node_type_id).
	if msg := s.applyClusterPolicy(&request, val(request.PolicyId), applyPolicyDefaultValues(req.Body)); msg != "" {
		return Response{
			StatusCode: 400,
			Body: map[string]string{
				"error_code": "INVALID_PARAMETER_VALUE",
				"message":    msg,
			},
		}
	}
	clusterFixUps(&request)

	// The cluster GET API returns apply_policy_default_values only under .spec, not at the top
	// level (clusters.ClusterInfo has no such field, so it is dropped by the unmarshal above).
	// Snapshot it from the raw body so re-reads match cloud.
	request.Spec = specSnapshot(req.Body)

	s.Clusters[clusterId] = request

	return Response{
		Body: clusters.ClusterInfo{
			ClusterId: nonZero(clusterId),
		},
	}
}

// specSnapshot mirrors how the cluster GET API returns apply_policy_default_values under .spec
// (it is absent from clusters.ClusterInfo and thus invisible at the top level of a re-read).
// Returns nil when the field is unset so re-reads of clusters that don't use it are unchanged;
// a nil .spec reads back as false, which is what the field's absence means anyway.
func specSnapshot(body []byte) *clusters.ClusterSpec {
	var spec clusters.ClusterSpec
	if err := json.Unmarshal(body, &spec); err != nil {
		return nil
	}
	if !spec.ApplyPolicyDefaultValues {
		return nil
	}
	return &clusters.ClusterSpec{
		ApplyPolicyDefaultValues: true,
	}
}

func (s *FakeWorkspace) ClustersResize(req Request) any {
	var request clusters.ResizeClusterRequest
	if err := json.Unmarshal(req.Body, &request); err != nil {
		return Response{
			StatusCode: 400,
			Body:       fmt.Sprintf("request parsing error: %s", err),
		}
	}

	defer s.LockUnlock()()
	cluster, ok := s.Clusters[val(request.ClusterId)]
	if !ok {
		return Response{StatusCode: 404}
	}

	// Only running clusters can be resized; match the real API behavior.
	if cluster.State != compute.StateRunning {
		return Response{
			StatusCode: 400,
			Body:       map[string]string{"error_code": "INVALID_STATE", "message": "Cluster is not running"},
		}
	}

	cluster.NumWorkers = request.NumWorkers
	cluster.Autoscale = request.Autoscale
	s.Clusters[val(request.ClusterId)] = cluster

	return Response{}
}

func (s *FakeWorkspace) ClustersEdit(req Request) any {
	var request clusters.ClusterInfo
	if err := json.Unmarshal(req.Body, &request); err != nil {
		return Response{
			StatusCode: 400,
			Body:       fmt.Sprintf("request parsing error: %s", err),
		}
	}

	defer s.LockUnlock()()
	existing, ok := s.Clusters[val(request.ClusterId)]
	if !ok {
		return Response{StatusCode: 404}
	}

	// Preserve runtime-only fields that the Edit API request doesn't include.
	request.State = existing.State
	request.ClusterId = existing.ClusterId
	// Apply the attached cluster policy before computing defaults, matching the backend:
	// policy values feed the defaults below (e.g. driver_node_type_id from node_type_id).
	if msg := s.applyClusterPolicy(&request, val(request.PolicyId), applyPolicyDefaultValues(req.Body)); msg != "" {
		return Response{
			StatusCode: 400,
			Body: map[string]string{
				"error_code": "INVALID_PARAMETER_VALUE",
				"message":    msg,
			},
		}
	}
	clusterFixUps(&request)
	// Refresh the .spec snapshot from the new settings, matching cloud behavior on edit.
	request.Spec = specSnapshot(req.Body)
	s.Clusters[val(request.ClusterId)] = request

	// Clear venv cache when cluster is edited to match cloud behavior where
	// cluster edits trigger restarts that clear library caches.
	if env, ok := s.clusterVenvs[val(request.ClusterId)]; ok {
		os.RemoveAll(env.dir)
		delete(s.clusterVenvs, val(request.ClusterId))
	}

	return Response{}
}

func setDefault(obj any, path string, value any) {
	if val, _ := structaccess.GetByString(obj, path); val == nil {
		_ = structaccess.SetByString(obj, path, value)
	}
}

// clusterFixUps applies server-side defaults that the real API sets.
func clusterFixUps(cluster *clusters.ClusterInfo) {
	gcp := cluster.GcpAttributes
	if gcp != nil {
		setDefault(gcp, "first_on_demand", 1)
		setDefault(gcp, "use_preemptible_executors", false)
	} else if cluster.AwsAttributes == nil {
		cluster.AwsAttributes = &clusters.AwsAttributes{
			Availability: clusters.AwsAvailability_SpotWithFallback,
			ZoneId:       nonZero("us-east-1c"),
		}
		cluster.AwsAttributes.ForceSendFields = append(
			cluster.AwsAttributes.ForceSendFields,
			"Availability",
			"ZoneId",
		)
	}

	cluster.ForceSendFields = append(cluster.ForceSendFields, "EnableElasticDisk")

	if val(cluster.DriverNodeTypeId) == "" && val(cluster.NodeTypeId) != "" {
		cluster.DriverNodeTypeId = cluster.NodeTypeId
	}
}

func (s *FakeWorkspace) ClustersGet(req Request, clusterId string) any {
	defer s.LockUnlock()()

	cluster, ok := s.Clusters[clusterId]
	if !ok {
		return Response{StatusCode: 404}
	}

	// Simulate cluster startup: transition PENDING → RUNNING so that WaitGetClusterRunning
	// resolves without hanging. Real clusters move through these states asynchronously.
	if cluster.State == compute.StatePending {
		cluster.State = compute.StateRunning
		s.Clusters[clusterId] = cluster
	}

	return Response{
		Body: cluster,
	}
}

// ClustersDelete terminates a cluster (sets state to TERMINATED without removing it).
func (s *FakeWorkspace) ClustersDelete(req Request) any {
	var request clusters.DeleteClusterRequest
	if err := json.Unmarshal(req.Body, &request); err != nil {
		return Response{
			StatusCode: 400,
			Body:       fmt.Sprintf("request parsing error: %s", err),
		}
	}
	defer s.LockUnlock()()

	cluster, ok := s.Clusters[val(request.ClusterId)]
	if !ok {
		return Response{StatusCode: 404}
	}

	cluster.State = compute.StateTerminated
	s.Clusters[val(request.ClusterId)] = cluster

	return Response{}
}

func (s *FakeWorkspace) ClustersStart(req Request) any {
	var request clusters.StartClusterRequest
	if err := json.Unmarshal(req.Body, &request); err != nil {
		return Response{
			StatusCode: 400,
			Body:       fmt.Sprintf("request parsing error: %s", err),
		}
	}
	defer s.LockUnlock()()

	cluster, ok := s.Clusters[val(request.ClusterId)]
	if !ok {
		return Response{StatusCode: 404}
	}

	cluster.State = compute.StateRunning
	s.Clusters[val(request.ClusterId)] = cluster

	return Response{}
}

func (s *FakeWorkspace) ClustersPermanentDelete(req Request) any {
	var request clusters.PermanentDeleteClusterRequest
	if err := json.Unmarshal(req.Body, &request); err != nil {
		return Response{
			StatusCode: 400,
			Body:       fmt.Sprintf("request parsing error: %s", err),
		}
	}

	defer s.LockUnlock()()
	delete(s.Clusters, val(request.ClusterId))
	return Response{}
}

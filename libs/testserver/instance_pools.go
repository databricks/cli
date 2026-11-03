package testserver

import (
	"encoding/json"
	"fmt"

	"github.com/databricks/databricks-sdk-go/service/compute"
)

func (s *FakeWorkspace) InstancePoolsCreate(req Request) any {
	// Unmarshal into the stored (GET) type directly: CreateInstancePool and
	// GetInstancePool share JSON field names, so every config field is carried over.
	var pool compute.GetInstancePool
	if err := json.Unmarshal(req.Body, &pool); err != nil {
		return Response{StatusCode: 400, Body: fmt.Sprintf("request parsing error: %s", err)}
	}

	defer s.LockUnlock()()

	id := nextUUID()
	pool.InstancePoolId = id
	s.InstancePools[id] = pool

	return Response{Body: compute.CreateInstancePoolResponse{InstancePoolId: id}}
}

func (s *FakeWorkspace) InstancePoolsGet(req Request, instancePoolId string) any {
	defer s.LockUnlock()()

	pool, ok := s.InstancePools[instancePoolId]
	if !ok {
		return Response{StatusCode: 404}
	}

	return Response{Body: pool}
}

func (s *FakeWorkspace) InstancePoolsEdit(req Request) any {
	var request compute.EditInstancePool
	if err := json.Unmarshal(req.Body, &request); err != nil {
		return Response{StatusCode: 400, Body: fmt.Sprintf("request parsing error: %s", err)}
	}

	defer s.LockUnlock()()

	pool, ok := s.InstancePools[request.InstancePoolId]
	if !ok {
		return Response{StatusCode: 404}
	}

	// Edit only accepts a subset of fields; overwrite those and keep the rest
	// (immutable fields like aws_attributes, disk_spec) as stored.
	pool.InstancePoolName = request.InstancePoolName
	pool.NodeTypeId = request.NodeTypeId
	pool.MinIdleInstances = request.MinIdleInstances
	pool.MaxCapacity = request.MaxCapacity
	pool.IdleInstanceAutoterminationMinutes = request.IdleInstanceAutoterminationMinutes
	pool.CustomTags = request.CustomTags
	pool.RemoteDiskThroughput = request.RemoteDiskThroughput
	pool.TotalInitialRemoteDiskSize = request.TotalInitialRemoteDiskSize
	s.InstancePools[request.InstancePoolId] = pool

	return Response{}
}

func (s *FakeWorkspace) InstancePoolsDelete(req Request) any {
	var request compute.DeleteInstancePool
	if err := json.Unmarshal(req.Body, &request); err != nil {
		return Response{StatusCode: 400, Body: fmt.Sprintf("request parsing error: %s", err)}
	}

	defer s.LockUnlock()()

	if _, ok := s.InstancePools[request.InstancePoolId]; !ok {
		return Response{StatusCode: 404}
	}

	delete(s.InstancePools, request.InstancePoolId)

	return Response{}
}

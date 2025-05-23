package testserver

import (
	"encoding/json"
	"fmt"

	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/google/uuid"
)

func (s *FakeWorkspace) PipelineGet(pipelineId string) Response {
	defer s.LockUnlock()()

	spec, ok := s.Pipelines[pipelineId]
	if !ok {
		return Response{
			StatusCode: 404,
		}
	}

	return Response{
		Body: pipelines.GetPipelineResponse{
			PipelineId: pipelineId,
			Spec:       &spec,
		},
	}
}

func (s *FakeWorkspace) PipelineCreate(req Request) Response {
	defer s.LockUnlock()()

	var r pipelines.PipelineSpec
	err := json.Unmarshal(req.Body, &r)
	if err != nil {
		return Response{
			Body:       fmt.Sprintf("cannot unmarshal request body: %s", err),
			StatusCode: 400,
		}
	}

	pipelineId := uuid.New().String()
	r.Id = pipelineId

	// If the pipeline definition does not specify a catalog, it switches to Hive metastore mode
	// and if the storage location is not specified, API automatically generates a storage location
	// (ref: https://docs.databricks.com/gcp/en/dlt/hive-metastore#specify-a-storage-location)
	if r.Storage == "" && r.Catalog == "" {
		r.Storage = "dbfs:/pipelines/" + pipelineId
	}
	s.Pipelines[pipelineId] = r

	return Response{
		Body: pipelines.CreatePipelineResponse{
			PipelineId: pipelineId,
		},
	}
}

func (s *FakeWorkspace) PipelineUpdate(req Request, pipelineId string) Response {
	defer s.LockUnlock()()

	var request pipelines.PipelineSpec
	err := json.Unmarshal(req.Body, &request)
	if err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: 400,
		}
	}

	_, exists := s.Pipelines[pipelineId]
	if !exists {
		return Response{
			StatusCode: 404,
		}
	}

	s.Pipelines[pipelineId] = request

	return Response{
		Body: pipelines.EditPipelineResponse{},
	}
}

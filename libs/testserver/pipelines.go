package testserver

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/databricks/databricks-sdk-go/service/pipelines"
)

func (s *FakeWorkspace) PipelineGet(pipelineId string) Response {
	defer s.LockUnlock()()

	value, ok := s.Pipelines[pipelineId]
	if !ok {
		return Response{
			StatusCode: 404,
			Body:       map[string]string{"message": fmt.Sprintf("The specified pipeline %s was not found.", pipelineId)},
		}
	}
	return Response{
		Body: value,
	}
}

func (s *FakeWorkspace) PipelineCreate(req Request) Response {
	defer s.LockUnlock()()

	spec := pipelines.PipelineSpec{}
	err := json.Unmarshal(req.Body, &spec)
	if err != nil {
		return Response{
			Body:       fmt.Sprintf("cannot unmarshal request body: %s", err),
			StatusCode: 400,
		}
	}

	// Unity Catalog requires target_schema_name to be a single schema segment, so a
	// dotted catalog.schema value is rejected; the catalog belongs in the separate
	// catalog field. Only the dot trips this check, but the backend's canned error
	// also lists dashes and other characters as invalid.
	if strings.Contains(spec.Target, ".") {
		return Response{
			StatusCode: 400,
			Body: map[string]string{
				"error_code": "INVALID_PARAMETER_VALUE",
				"message":    fmt.Sprintf("CreatePipeline target_schema_name %q is not a valid name. Valid names must contain only alphanumeric characters and underscores, and cannot contain spaces, periods, forward slashes, or control characters.", spec.Target),
			},
		}
	}

	var r pipelines.GetPipelineResponse
	r.Spec = &spec

	pipelineId := nextUUID()
	r.PipelineId = pipelineId
	r.CreatorUserName = "tester@databricks.com"
	r.LastModified = nowMilli()
	r.Name = r.Spec.Name
	r.RunAsUserName = "tester@databricks.com"
	r.State = "IDLE"
	r.EffectivePublishingMode = pipelines.PublishingModeDefaultPublishingMode

	setSpecDefaults(&spec, pipelineId)
	s.Pipelines[pipelineId] = r

	return Response{
		Body: pipelines.CreatePipelineResponse{
			PipelineId: pipelineId,
		},
	}
}

func setSpecDefaults(spec *pipelines.PipelineSpec, pipelineId string) {
	spec.Id = pipelineId
	// If the pipeline definition does not specify a catalog, it switches to Hive metastore mode
	// and if the storage location is not specified, API automatically generates a storage location
	// (ref: https://docs.databricks.com/gcp/en/dlt/hive-metastore#specify-a-storage-location)
	if spec.Storage == "" && spec.Catalog == "" {
		spec.Storage = "dbfs:/pipelines/" + pipelineId
	}
}

func (s *FakeWorkspace) PipelineUpdate(req Request, pipelineId string) Response {
	defer s.LockUnlock()()

	var spec pipelines.PipelineSpec
	err := json.Unmarshal(req.Body, &spec)
	if err != nil {
		return Response{
			Body:       fmt.Sprintf("internal error: %s", err),
			StatusCode: 400,
		}
	}

	item, exists := s.Pipelines[pipelineId]
	if !exists {
		return Response{
			StatusCode: 404,
		}
	}

	item.Spec = &spec
	setSpecDefaults(&spec, pipelineId)
	s.Pipelines[pipelineId] = item

	return Response{}
}

func (s *FakeWorkspace) PipelineStartUpdate(pipelineId string) Response {
	defer s.LockUnlock()()

	_, exists := s.Pipelines[pipelineId]
	if !exists {
		return Response{
			StatusCode: 404,
			Body:       map[string]string{"message": fmt.Sprintf("The specified pipeline %s was not found.", pipelineId)},
		}
	}

	updateId := nextUUID()
	s.PipelineUpdates[updateId] = true

	return Response{
		Body: pipelines.StartUpdateResponse{
			UpdateId: updateId,
		},
	}
}

func (s *FakeWorkspace) PipelineEvents(pipelineId string) Response {
	defer s.LockUnlock()()

	_, exists := s.Pipelines[pipelineId]
	if !exists {
		return Response{
			StatusCode: 404,
			Body:       map[string]string{"message": fmt.Sprintf("The specified pipeline %s was not found.", pipelineId)},
		}
	}

	return Response{
		Body: map[string]any{
			"events": []pipelines.PipelineEvent{},
		},
	}
}

func (s *FakeWorkspace) PipelineGetUpdate(pipelineId, updateId string) Response {
	defer s.LockUnlock()()

	_, exists := s.Pipelines[pipelineId]
	if !exists {
		return Response{
			StatusCode: 404,
			Body:       map[string]string{"message": fmt.Sprintf("The specified pipeline %s was not found.", pipelineId)},
		}
	}

	// Check if the update exists
	_, updateExists := s.PipelineUpdates[updateId]
	if !updateExists {
		return Response{
			StatusCode: 404,
			Body:       map[string]string{"message": fmt.Sprintf("The specified update %s was not found.", updateId)},
		}
	}

	return Response{
		Body: pipelines.GetUpdateResponse{
			Update: &pipelines.UpdateInfo{
				UpdateId: updateId,
				State:    pipelines.UpdateInfoStateCompleted,
			},
		},
	}
}

func (s *FakeWorkspace) PipelineStop(pipelineId string) Response {
	defer s.LockUnlock()()

	_, exists := s.Pipelines[pipelineId]
	if !exists {
		return Response{
			StatusCode: 404,
			Body:       map[string]string{"message": fmt.Sprintf("The specified pipeline %s was not found.", pipelineId)},
		}
	}

	return Response{
		Body: pipelines.GetPipelineResponse{
			PipelineId: pipelineId,
			State:      pipelines.PipelineStateIdle,
		},
	}
}
